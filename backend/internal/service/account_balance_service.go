package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// NewAPI internal quota units per USD.
const quotaPerUnit = 500_000.0

// BalanceInfo contains upstream account balance information.
type BalanceInfo struct {
	Balance    float64 `json:"balance"`
	UsedQuota  float64 `json:"used_quota"`
	TotalQuota float64 `json:"total_quota"`
	PanelType  string  `json:"panel_type"`
	UserRole   string  `json:"user_role,omitempty"`
	UserName   string  `json:"user_name,omitempty"`
	UserEmail  string  `json:"user_email,omitempty"`
	UpdatedAt  string  `json:"updated_at"`
	Error      string  `json:"error,omitempty"`
	ErrorCode  int     `json:"error_code,omitempty"` // HTTP status code on auth failure (401/403)
}

// BatchBalanceResult is the response for batch balance queries.
type BatchBalanceResult struct {
	Results map[int64]*BalanceInfo `json:"results"`
	Success int                   `json:"success"`
	Failed  int                   `json:"failed"`
}

// AccountBalanceService queries upstream panel balance for apikey/upstream accounts.
type AccountBalanceService struct {
	accountRepo AccountRepository
	cfg         *config.Config
	httpClient  *http.Client
}

// NewAccountBalanceService creates a new AccountBalanceService.
func NewAccountBalanceService(accountRepo AccountRepository, cfg *config.Config) *AccountBalanceService {
	return &AccountBalanceService{
		accountRepo: accountRepo,
		cfg:         cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// CanQueryBalance checks whether an account supports upstream balance queries.
func CanQueryBalance(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Type != AccountTypeAPIKey && account.Type != AccountTypeUpstream {
		return false
	}
	apiKey := account.GetCredential("api_key")
	baseURL := account.GetCredential("base_url")
	return apiKey != "" && baseURL != ""
}

// QueryBalance queries the upstream balance for a single account.
func (s *AccountBalanceService) QueryBalance(ctx context.Context, accountID int64) (*BalanceInfo, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	if !CanQueryBalance(account) {
		return nil, fmt.Errorf("account does not support balance query")
	}

	apiKey := account.GetCredential("api_key")
	baseURL := strings.TrimRight(account.GetCredential("base_url"), "/")
	// Strip trailing /v1 if present
	if strings.HasSuffix(baseURL, "/v1") {
		baseURL = baseURL[:len(baseURL)-3]
	}

	info := s.probeBalanceWithRetry(ctx, baseURL, apiKey, 2)

	// Cache result in account extra
	updates := map[string]any{
		"upstream_balance":            info.Balance,
		"upstream_used_quota":         info.UsedQuota,
		"upstream_total_quota":        info.TotalQuota,
		"upstream_balance_panel_type": info.PanelType,
		"upstream_balance_user_role":  info.UserRole,
		"upstream_balance_user_name":  info.UserName,
		"upstream_balance_user_email": info.UserEmail,
		"upstream_balance_updated_at": info.UpdatedAt,
		"upstream_balance_error":      info.Error,
		"upstream_balance_error_code": info.ErrorCode,
	}
	if updateErr := s.accountRepo.UpdateExtra(ctx, accountID, updates); updateErr != nil {
		fmt.Printf("[AccountBalanceService] Warning: failed to cache balance for account %d: %v\n", accountID, updateErr)
	}

	return info, nil
}

// QueryBalanceBatch queries balance for multiple accounts concurrently.
func (s *AccountBalanceService) QueryBalanceBatch(ctx context.Context, accountIDs []int64) *BatchBalanceResult {
	result := &BatchBalanceResult{
		Results: make(map[int64]*BalanceInfo),
	}
	var mu sync.Mutex

	sem := make(chan struct{}, 5) // concurrency limit
	var wg sync.WaitGroup

	for _, id := range accountIDs {
		wg.Add(1)
		go func(accountID int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			info, err := s.QueryBalance(ctx, accountID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Results[accountID] = &BalanceInfo{
					Error:     err.Error(),
					UpdatedAt: time.Now().UTC().Format(time.RFC3339),
				}
				result.Failed++
			} else if info.Error != "" {
				result.Results[accountID] = info
				result.Failed++
			} else {
				result.Results[accountID] = info
				result.Success++
			}
		}(id)
	}

	wg.Wait()
	return result
}

// probeBalanceWithRetry wraps probeBalance with retry logic for transient failures.
func (s *AccountBalanceService) probeBalanceWithRetry(ctx context.Context, baseURL, apiKey string, maxRetries int) *BalanceInfo {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &BalanceInfo{
					Error:     "request cancelled",
					UpdatedAt: time.Now().UTC().Format(time.RFC3339),
				}
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		info, err := s.probeBalance(ctx, baseURL, apiKey)
		if err == nil {
			return info
		}
		lastErr = err

		// Don't retry on auth failures (401/403) — these are permanent
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			break
		}
	}

	errMsg := "unable to detect panel type"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	return &BalanceInfo{
		Error:     errMsg,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// probeBalance tries multiple upstream API strategies to detect panel type and query balance.
func (s *AccountBalanceService) probeBalance(ctx context.Context, baseURL, apiKey string) (*BalanceInfo, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
	now := time.Now().UTC().Format(time.RFC3339)

	// Strategy A: NewAPI / OpenAI billing endpoints
	info, err := s.tryBillingSubscription(ctx, baseURL, headers, now)
	if err == nil {
		return info, nil
	}

	// Strategy B: Generic panel /v1/user/info (NewAPI / OneAPI / VoAPI)
	info, err = s.tryUserInfo(ctx, baseURL, headers, now)
	if err == nil {
		return info, nil
	}

	// Strategy C: Wrapped /api/user/self (some OneAPI forks)
	info, err = s.tryUserSelf(ctx, baseURL, headers, now)
	if err == nil {
		return info, nil
	}

	// Strategy D: JWT panel /api/v1/auth/me
	info, err = s.tryAuthMe(ctx, baseURL, headers, now)
	if err == nil {
		return info, nil
	}

	return nil, fmt.Errorf("all %d strategies failed for %s", 4, baseURL)
}

// tryBillingSubscription tries GET {base}/v1/dashboard/billing/subscription + usage
func (s *AccountBalanceService) tryBillingSubscription(ctx context.Context, baseURL string, headers map[string]string, now string) (*BalanceInfo, error) {
	subURL := baseURL + "/v1/dashboard/billing/subscription"
	body, statusCode, err := s.doGet(ctx, subURL, headers)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401)")
	}
	if statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("access forbidden (403)")
	}
	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	var sub map[string]any
	if err := json.Unmarshal(body, &sub); err != nil {
		return nil, fmt.Errorf("invalid JSON response")
	}
	if _, hasErr := sub["error"]; hasErr {
		return nil, fmt.Errorf("subscription endpoint returned error")
	}

	hardLimit := toFloat64(sub["hard_limit_usd"])

	// Extract plan name if available
	planName, _ := sub["plan"].(map[string]any)
	planTitle := ""
	if planName != nil {
		planTitle, _ = planName["title"].(string)
	}
	if planTitle == "" {
		planTitle, _ = sub["plan_id"].(string)
	}

	// Fetch usage
	usageURL := baseURL + "/v1/dashboard/billing/usage"
	usageBody, usageStatus, err := s.doGet(ctx, usageURL, headers)
	if err != nil || usageStatus != http.StatusOK {
		return &BalanceInfo{
			Balance:    hardLimit,
			TotalQuota: hardLimit,
			PanelType:  "newapi",
			UserRole:   planTitle,
			UpdatedAt:  now,
		}, nil
	}

	var usage map[string]any
	if err := json.Unmarshal(usageBody, &usage); err != nil {
		return &BalanceInfo{
			Balance:    hardLimit,
			TotalQuota: hardLimit,
			PanelType:  "newapi",
			UserRole:   planTitle,
			UpdatedAt:  now,
		}, nil
	}

	totalUsage := toFloat64(usage["total_usage"]) / 100.0
	return &BalanceInfo{
		Balance:    hardLimit - totalUsage,
		UsedQuota:  totalUsage,
		TotalQuota: hardLimit,
		PanelType:  "newapi",
		UserRole:   planTitle,
		UpdatedAt:  now,
	}, nil
}

// tryUserInfo tries GET {base}/v1/user/info (generic panel: NewAPI/OneAPI/VoAPI)
func (s *AccountBalanceService) tryUserInfo(ctx context.Context, baseURL string, headers map[string]string, now string) (*BalanceInfo, error) {
	url := baseURL + "/v1/user/info"
	body, statusCode, err := s.doGet(ctx, url, headers)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401)")
	}
	if statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("access forbidden (403)")
	}
	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}

	// Some panels wrap response in {"success": true, "data": {...}} or {"code": 0, "data": {...}}
	data := unwrapResponse(raw)

	if _, hasBalance := data["balance"]; !hasBalance {
		// Try "quota" as alias
		if _, hasQuota := data["quota"]; !hasQuota {
			return nil, fmt.Errorf("no balance field in response")
		}
	}

	balance := toFloat64(data["balance"])
	if balance == 0 {
		balance = toFloat64(data["quota"])
	}
	usedQuota := toFloat64(data["used_quota"])
	totalCost := toFloat64(data["totalCost"])
	if totalCost == 0 {
		totalCost = usedQuota
	}

	// Detect if values are in internal quota units (very large numbers > 1M suggest raw quota)
	// NewAPI uses 500,000 units = $1 USD
	if balance > 1_000_000 || totalCost > 1_000_000 {
		balance = balance / quotaPerUnit
		totalCost = totalCost / quotaPerUnit
	}

	userName, _ := data["name"].(string)
	if userName == "" {
		userName, _ = data["display_name"].(string)
	}
	userEmail, _ := data["email"].(string)
	status, _ := data["status"].(string)
	group, _ := data["group"].(string)

	userRole := buildUserRole(userName, status, group)

	return &BalanceInfo{
		Balance:    roundTo(balance, 4),
		UsedQuota:  roundTo(totalCost, 4),
		TotalQuota: roundTo(balance+totalCost, 4),
		PanelType:  "generic",
		UserRole:   userRole,
		UserName:   userName,
		UserEmail:  userEmail,
		UpdatedAt:  now,
	}, nil
}

// tryUserSelf tries GET {base}/api/user/self (some OneAPI forks)
func (s *AccountBalanceService) tryUserSelf(ctx context.Context, baseURL string, headers map[string]string, now string) (*BalanceInfo, error) {
	url := baseURL + "/api/user/self"
	body, statusCode, err := s.doGet(ctx, url, headers)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("auth error (%d)", statusCode)
	}
	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}

	data := unwrapResponse(raw)

	if _, hasQuota := data["quota"]; !hasQuota {
		if _, hasBal := data["balance"]; !hasBal {
			return nil, fmt.Errorf("no quota/balance field")
		}
	}

	quota := toFloat64(data["quota"])
	if quota == 0 {
		quota = toFloat64(data["balance"])
	}
	usedQuota := toFloat64(data["used_quota"])

	// Convert from internal units if needed
	if quota > 1_000_000 || usedQuota > 1_000_000 {
		quota = quota / quotaPerUnit
		usedQuota = usedQuota / quotaPerUnit
	}

	userName, _ := data["display_name"].(string)
	if userName == "" {
		userName, _ = data["username"].(string)
	}
	userEmail, _ := data["email"].(string)
	group, _ := data["group"].(string)
	role, _ := data["role"].(string)

	userRole := "-"
	if group != "" {
		userRole = group
	} else if role != "" {
		userRole = role
	}

	return &BalanceInfo{
		Balance:    roundTo(quota-usedQuota, 4),
		UsedQuota:  roundTo(usedQuota, 4),
		TotalQuota: roundTo(quota, 4),
		PanelType:  "oneapi",
		UserRole:   userRole,
		UserName:   userName,
		UserEmail:  userEmail,
		UpdatedAt:  now,
	}, nil
}

// tryAuthMe tries GET {base}/api/v1/auth/me (JWT panel)
func (s *AccountBalanceService) tryAuthMe(ctx context.Context, baseURL string, headers map[string]string, now string) (*BalanceInfo, error) {
	url := baseURL + "/api/v1/auth/me"
	body, statusCode, err := s.doGet(ctx, url, headers)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed (401)")
	}
	if statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("access forbidden (403)")
	}
	if statusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}

	code, _ := resp["code"].(float64)
	if code != 0 {
		return nil, fmt.Errorf("auth/me returned non-zero code")
	}

	userData, ok := resp["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("no data field in auth/me response")
	}

	balance := toFloat64(userData["balance"])

	// Convert if in internal units
	if balance > 1_000_000 {
		balance = balance / quotaPerUnit
	}

	username, _ := userData["username"].(string)
	email, _ := userData["email"].(string)
	role, _ := userData["role"].(string)
	display := username
	if display == "" {
		display = email
	}
	if display == "" {
		display = role
	}
	userRole := "-"
	if role != "" {
		if display != role {
			userRole = fmt.Sprintf("%s(%s)", display, role)
		} else {
			userRole = role
		}
	} else if display != "" {
		userRole = display
	}

	return &BalanceInfo{
		Balance:    roundTo(balance, 4),
		TotalQuota: roundTo(balance, 4),
		PanelType:  "jwt",
		UserRole:   userRole,
		UserName:   username,
		UserEmail:  email,
		UpdatedAt:  now,
	}, nil
}

// doGet performs an HTTP GET request and returns body bytes, status code, and error.
func (s *AccountBalanceService) doGet(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// unwrapResponse handles common API response wrappers.
// Some panels return {"success": true, "data": {...}} or {"code": 0, "data": {...}}.
func unwrapResponse(raw map[string]any) map[string]any {
	// Check if response has a "data" wrapper
	if dataField, ok := raw["data"]; ok {
		if dataMap, ok := dataField.(map[string]any); ok {
			// Verify this is a wrapper by checking for common wrapper fields
			_, hasSuccess := raw["success"]
			_, hasCode := raw["code"]
			_, hasMessage := raw["message"]
			_, hasMsg := raw["msg"]
			if hasSuccess || hasCode || hasMessage || hasMsg {
				return dataMap
			}
		}
	}
	return raw
}

// buildUserRole constructs a display string from name, status, and group.
func buildUserRole(name, status, group string) string {
	if group != "" {
		if name != "" {
			return fmt.Sprintf("%s(%s)", name, group)
		}
		return group
	}
	if status == "active" || status == "" {
		if name != "" {
			return name
		}
		return "active"
	}
	if name != "" {
		return fmt.Sprintf("%s(%s)", name, status)
	}
	return status
}

// toFloat64 converts various numeric types to float64.
func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// roundTo rounds a float64 to n decimal places.
func roundTo(val float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	return math.Round(val*pow) / pow
}

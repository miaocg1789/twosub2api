package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

// OpenAICompatForwardResult represents the result of forwarding an OpenAI-compatible request.
type OpenAICompatForwardResult struct {
	RequestID string
	Model     string
	Usage     OpenAICompatUsage
	ImageSize string // Image size category: "1K", "2K", "4K" (for image generation billing)
}

// OpenAICompatUsage represents token usage from an OpenAI-compatible API response.
type OpenAICompatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAICompatRecordUsageInput input for recording usage of OpenAI-compatible requests.
type OpenAICompatRecordUsageInput struct {
	Result       *OpenAICompatForwardResult
	APIKey       *APIKey
	User         *User
	Account      *Account
	Subscription *UserSubscription
	UserAgent    string
	IPAddress    string
}

// OpenAICompatGatewayService handles OpenAI-compatible API gateway operations
// for DeepSeek, Qwen, GLM, and other OpenAI-compatible upstreams.
type OpenAICompatGatewayService struct {
	accountRepo         AccountRepository
	usageLogRepo        UsageLogRepository
	userRepo            UserRepository
	userSubRepo         UserSubscriptionRepository
	cache               GatewayCache
	cfg                 *config.Config
	schedulerSnapshot   *SchedulerSnapshotService
	concurrencyService  *ConcurrencyService
	billingService      *BillingService
	rateLimitService    *RateLimitService
	billingCacheService *BillingCacheService
	httpUpstream        HTTPUpstream
}

// NewOpenAICompatGatewayService creates a new OpenAICompatGatewayService.
func NewOpenAICompatGatewayService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	cache GatewayCache,
	cfg *config.Config,
	schedulerSnapshot *SchedulerSnapshotService,
	concurrencyService *ConcurrencyService,
	billingService *BillingService,
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	httpUpstream HTTPUpstream,
) *OpenAICompatGatewayService {
	return &OpenAICompatGatewayService{
		accountRepo:         accountRepo,
		usageLogRepo:        usageLogRepo,
		userRepo:            userRepo,
		userSubRepo:         userSubRepo,
		cache:               cache,
		cfg:                 cfg,
		schedulerSnapshot:   schedulerSnapshot,
		concurrencyService:  concurrencyService,
		billingService:      billingService,
		rateLimitService:    rateLimitService,
		billingCacheService: billingCacheService,
		httpUpstream:        httpUpstream,
	}
}

// SelectAccountWithLoadAwareness selects an account for the OpenAI-compatible platform
// with load-aware scheduling support.
func (s *OpenAICompatGatewayService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, platform string, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	accounts, err := s.listSchedulableAccounts(ctx, groupID, platform)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, errors.New("no available accounts for platform " + platform)
	}

	// Filter by model support and exclusions
	var candidates []Account
	for _, acc := range accounts {
		if _, excluded := excludedIDs[acc.ID]; excluded {
			continue
		}
		if requestedModel != "" && !acc.IsModelSupported(requestedModel) {
			continue
		}
		candidates = append(candidates, acc)
	}

	if len(candidates) == 0 {
		return nil, errors.New("no available accounts supporting model " + requestedModel)
	}

	// Simple selection: pick the best candidate by priority + LRU
	best := &candidates[0]
	for i := 1; i < len(candidates); i++ {
		candidate := &candidates[i]
		if candidate.Priority > best.Priority {
			best = candidate
		} else if candidate.Priority == best.Priority {
			if best.LastUsedAt != nil && (candidate.LastUsedAt == nil || candidate.LastUsedAt.Before(*best.LastUsedAt)) {
				best = candidate
			}
		}
	}

	// Try to acquire concurrency slot
	if s.concurrencyService != nil && best.Concurrency > 0 {
		result, err := s.concurrencyService.AcquireAccountSlot(ctx, best.ID, best.Concurrency)
		if err == nil && result.Acquired {
			return &AccountSelectionResult{
				Account:     best,
				Acquired:    true,
				ReleaseFunc: result.ReleaseFunc,
			}, nil
		}
	}

	return &AccountSelectionResult{
		Account:  best,
		Acquired: true,
		ReleaseFunc: func() {},
	}, nil
}

func (s *OpenAICompatGatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64, platform string) ([]Account, error) {
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, false)
		return accounts, err
	}
	var accounts []Account
	var err error
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListSchedulableByPlatform(ctx, platform)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, platform)
	} else {
		accounts, err = s.accountRepo.ListSchedulableByPlatform(ctx, platform)
	}
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	return accounts, nil
}

// GetAPIKey returns the API key for an OpenAI-compatible account.
func (s *OpenAICompatGatewayService) GetAPIKey(account *Account) (string, error) {
	if account.Type == AccountTypeAPIKey {
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return "", errors.New("api_key not found in credentials")
		}
		return apiKey, nil
	}
	if account.Type == AccountTypeOAuth {
		accessToken := account.GetCredential("access_token")
		if accessToken == "" {
			return "", errors.New("access_token not found in OAuth credentials")
		}
		return accessToken, nil
	}
	return "", fmt.Errorf("unsupported account type for OpenAI-compatible platform: %s", account.Type)
}

// ForwardChatCompletion forwards a chat completion request to the upstream OpenAI-compatible API.
func (s *OpenAICompatGatewayService) ForwardChatCompletion(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAICompatForwardResult, error) {
	startTime := time.Now()

	// Parse request body
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}

	reqModel, _ := reqBody["model"].(string)
	reqStream, _ := reqBody["stream"].(bool)
	originalModel := reqModel

	// Apply model mapping
	mappedModel := account.GetMappedModel(reqModel)
	if mappedModel != reqModel {
		log.Printf("[OpenAICompat] Model mapping: %s -> %s (account: %s)", reqModel, mappedModel, account.Name)
		reqBody["model"] = mappedModel
		var err error
		body, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("re-serialize request: %w", err)
		}
	}

	// Get API key
	apiKey, err := s.GetAPIKey(account)
	if err != nil {
		return nil, err
	}

	// Build upstream URL
	baseURL := account.GetBaseURL()
	targetURL := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"

	// Build request
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Get proxy URL if configured
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// Execute request
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// Handle error responses
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		// Handle rate limit side effects
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		if s.shouldFailover(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:   resp.StatusCode,
				ResponseBody: respBody,
			}
		}
		// Forward error response to client
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return nil, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	result := &OpenAICompatForwardResult{
		RequestID: resp.Header.Get("x-request-id"),
		Model:     originalModel,
	}

	if reqStream {
		// Handle streaming response
		usage, err := s.handleStreamingResponse(resp, c, originalModel, mappedModel)
		if err != nil {
			return nil, err
		}
		if usage != nil {
			result.Usage = *usage
		}
	} else {
		// Handle non-streaming response
		usage, err := s.handleNonStreamingResponse(resp, c, originalModel, mappedModel)
		if err != nil {
			return nil, err
		}
		if usage != nil {
			result.Usage = *usage
		}
	}

	log.Printf("[OpenAICompat] Request completed: model=%s account=%s duration=%v input=%d output=%d",
		originalModel, account.Name, time.Since(startTime), result.Usage.PromptTokens, result.Usage.CompletionTokens)

	return result, nil
}

// ForwardImageGeneration forwards an image generation request to the upstream API.
func (s *OpenAICompatGatewayService) ForwardImageGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAICompatForwardResult, error) {
	// Get API key
	apiKey, err := s.GetAPIKey(account)
	if err != nil {
		return nil, err
	}

	// Parse request to extract model
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	reqModel, _ := reqBody["model"].(string)

	// Build upstream URL
	baseURL := account.GetBaseURL()
	targetURL := strings.TrimRight(baseURL, "/") + "/v1/images/generations"

	// Build request
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Get proxy URL if configured
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// Execute request
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		if s.shouldFailover(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:   resp.StatusCode,
				ResponseBody: respBody,
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return nil, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	// Forward response to client
	c.Data(resp.StatusCode, "application/json", respBody)

	// Extract image count for billing
	result := &OpenAICompatForwardResult{
		RequestID: resp.Header.Get("x-request-id"),
		Model:     reqModel,
	}

	// Determine image size category from request
	if sizeStr, ok := reqBody["size"].(string); ok {
		result.ImageSize = classifyImageSize(sizeStr)
	}

	// Parse response to count images
	var imgResp struct {
		Data []any `json:"data"`
	}
	if json.Unmarshal(respBody, &imgResp) == nil {
		// Use completion_tokens as image count for billing purposes
		result.Usage.CompletionTokens = len(imgResp.Data)
	}

	return result, nil
}

func (s *OpenAICompatGatewayService) shouldFailover(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}

// classifyImageSize converts a size string (e.g. "1024x1024") to a billing category ("1K", "2K", "4K").
func classifyImageSize(size string) string {
	switch size {
	case "256x256", "512x512", "1024x1024":
		return "1K"
	case "1024x1792", "1792x1024", "1536x1024", "1024x1536":
		return "2K"
	default:
		return "1K"
	}
}

func (s *OpenAICompatGatewayService) handleStreamingResponse(resp *http.Response, c *gin.Context, originalModel, mappedModel string) (*OpenAICompatUsage, error) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var usage OpenAICompatUsage
	needModelReplace := mappedModel != originalModel && mappedModel != ""

	for scanner.Scan() {
		line := scanner.Text()

		// Try to extract usage from SSE data lines
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data != "[DONE]" {
				s.parseSSEUsage(data, &usage)
				// Replace model name in response if needed
				if needModelReplace {
					line = s.replaceModelInSSELine(line, mappedModel, originalModel)
				}
			}
		}

		fmt.Fprintf(c.Writer, "%s\n", line)
	}
	// Ensure final newline and flush
	flusher.Flush()

	if err := scanner.Err(); err != nil {
		return &usage, nil // Return partial usage on scan error
	}

	return &usage, nil
}

func (s *OpenAICompatGatewayService) handleNonStreamingResponse(resp *http.Response, c *gin.Context, originalModel, mappedModel string) (*OpenAICompatUsage, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Extract usage
	var respObj struct {
		Usage OpenAICompatUsage `json:"usage"`
		Model string            `json:"model"`
	}
	var usage OpenAICompatUsage
	if json.Unmarshal(body, &respObj) == nil {
		usage = respObj.Usage
	}

	// Replace model name if needed
	if mappedModel != originalModel && mappedModel != "" {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}

	c.Data(resp.StatusCode, "application/json", body)
	return &usage, nil
}

func (s *OpenAICompatGatewayService) parseSSEUsage(data string, usage *OpenAICompatUsage) {
	var chunk struct {
		Usage *OpenAICompatUsage `json:"usage"`
	}
	if json.Unmarshal([]byte(data), &chunk) == nil && chunk.Usage != nil {
		*usage = *chunk.Usage
	}
}

func (s *OpenAICompatGatewayService) replaceModelInSSELine(line, fromModel, toModel string) string {
	if !strings.Contains(line, fromModel) {
		return line
	}
	return strings.Replace(line, `"model":"`+fromModel+`"`, `"model":"`+toModel+`"`, 1)
}

func (s *OpenAICompatGatewayService) replaceModelInResponseBody(body []byte, fromModel, toModel string) []byte {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return body
	}

	model, ok := resp["model"].(string)
	if !ok || model != fromModel {
		return body
	}

	resp["model"] = toModel
	newBody, err := json.Marshal(resp)
	if err != nil {
		return body
	}
	return newBody
}

// isOpenAICompatImageModel checks if a model is an image generation model
// that uses the OpenAI-compatible /v1/images/generations endpoint.
func isOpenAICompatImageModel(model string) bool {
	m := strings.ToLower(model)
	return strings.HasPrefix(m, "dall-e") ||
		strings.HasPrefix(m, "gpt-image") ||
		strings.HasPrefix(m, "wanx") ||
		strings.HasPrefix(m, "flux-") ||
		strings.HasPrefix(m, "stable-diffusion") ||
		strings.HasPrefix(m, "cogview") ||
		strings.HasPrefix(m, "imagen-")
}

// RecordUsage records usage for an OpenAI-compatible request.
func (s *OpenAICompatGatewayService) RecordUsage(ctx context.Context, input *OpenAICompatRecordUsageInput) error {
	if input == nil || input.Result == nil {
		return nil
	}

	result := input.Result
	account := input.Account
	apiKey := input.APIKey
	user := input.User

	rateMultiplier := account.BillingRateMultiplier()

	var breakdown *CostBreakdown

	// 按次计费优先（优先级：按次 > 图片 > token）
	if apiKey.Group != nil {
		if price, ok := apiKey.Group.GetPerRequestPrice(result.Model); ok {
			breakdown = s.billingService.CalculatePerRequestCost(price, rateMultiplier)
		}
	}

	// Check if this is an image generation model (CompletionTokens repurposed as image count)
	if breakdown == nil && isOpenAICompatImageModel(result.Model) && result.Usage.CompletionTokens > 0 {
		imageCount := result.Usage.CompletionTokens
		imageSize := "1K" // default
		if result.ImageSize != "" {
			imageSize = result.ImageSize
		}

		// Use group image pricing if available
		var groupConfig *ImagePriceConfig
		if apiKey.Group != nil {
			groupConfig = &ImagePriceConfig{}
			if apiKey.Group.ImagePrice1K != nil {
				groupConfig.Price1K = apiKey.Group.ImagePrice1K
			}
			if apiKey.Group.ImagePrice2K != nil {
				groupConfig.Price2K = apiKey.Group.ImagePrice2K
			}
			if apiKey.Group.ImagePrice4K != nil {
				groupConfig.Price4K = apiKey.Group.ImagePrice4K
			}
		}

		breakdown = s.billingService.CalculateImageCost(result.Model, imageSize, imageCount, groupConfig, rateMultiplier)
	} else if breakdown == nil {
		// Standard token-based billing
		tokens := UsageTokens{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		}

		var err error
		breakdown, err = s.billingService.CalculateCost(result.Model, tokens, rateMultiplier)
		if err != nil {
			log.Printf("[OpenAICompat] Calculate cost failed: %v (model=%s)", err, result.Model)
			breakdown = &CostBreakdown{
				TotalCost:  0,
				ActualCost: 0,
			}
		}
	}

	// Build usage log
	var userAgent, ipAddress *string
	if input.UserAgent != "" {
		ua := input.UserAgent
		userAgent = &ua
	}
	if input.IPAddress != "" {
		ip := input.IPAddress
		ipAddress = &ip
	}

	usageLog := &UsageLog{
		UserID:                user.ID,
		APIKeyID:              apiKey.ID,
		AccountID:             account.ID,
		RequestID:             result.RequestID,
		GroupID:               apiKey.GroupID,
		Model:                 result.Model,
		InputTokens:           result.Usage.PromptTokens,
		OutputTokens:          result.Usage.CompletionTokens,
		InputCost:             breakdown.InputCost,
		OutputCost:            breakdown.OutputCost,
		TotalCost:             breakdown.TotalCost,
		ActualCost:            breakdown.ActualCost,
		RateMultiplier:        rateMultiplier,
		AccountRateMultiplier: account.RateMultiplier,
		UserAgent:             userAgent,
		IPAddress:             ipAddress,
	}

	if input.Subscription != nil {
		usageLog.BillingType = BillingTypeSubscription
		usageLog.SubscriptionID = &input.Subscription.ID
	}

	if _, err := s.usageLogRepo.Create(ctx, usageLog); err != nil {
		log.Printf("[OpenAICompat] Record usage log failed: %v", err)
	}

	// Deduct balance
	if breakdown.ActualCost > 0 {
		if input.Subscription != nil && apiKey.GroupID != nil {
			s.billingCacheService.QueueUpdateSubscriptionUsage(user.ID, *apiKey.GroupID, breakdown.ActualCost)
		} else {
			s.billingCacheService.QueueDeductBalance(user.ID, breakdown.ActualCost)
		}
	}

	return nil
}

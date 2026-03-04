package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	qwenDeviceCodeURL = "https://chat.qwen.ai/api/v1/oauth2/device/code"
	qwenTokenURL      = "https://chat.qwen.ai/api/v1/oauth2/token"
	qwenClientID      = "f0304373b74a44d2b584a3fb70ca9e56"
	qwenScope         = "openid profile email model.completion"
	qwenDeviceGrant   = "urn:ietf:params:oauth:grant-type:device_code"

	qwenSessionTTL     = 30 * time.Minute
	qwenPollInterval   = 5 * time.Second
	qwenMaxPollRetries = 60
)

// QwenOAuthService handles Qwen Device Authorization Grant (RFC 8628)
type QwenOAuthService struct {
	sessionStore *qwenSessionStore
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
}

// NewQwenOAuthService creates a new QwenOAuthService
func NewQwenOAuthService(proxyRepo ProxyRepository, httpUpstream HTTPUpstream) *QwenOAuthService {
	return &QwenOAuthService{
		sessionStore: newQwenSessionStore(),
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
	}
}

// QwenDeviceFlowResult is returned after initiating device authorization
type QwenDeviceFlowResult struct {
	SessionID               string `json:"session_id"`
	UserCode                string `json:"user_code"`
	VerificationURL         string `json:"verification_url"`
	VerificationURLComplete string `json:"verification_url_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// QwenTokenInfo contains token information from Qwen OAuth
type QwenTokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	ResourceURL  string `json:"resource_url,omitempty"`
}

// qwenDeviceSession stores device flow session state
type qwenDeviceSession struct {
	DeviceCode   string
	CodeVerifier string
	ProxyURL     string
	Interval     int
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// qwenSessionStore stores device flow sessions with TTL cleanup
type qwenSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*qwenDeviceSession
	stopCh   chan struct{}
}

func newQwenSessionStore() *qwenSessionStore {
	store := &qwenSessionStore{
		sessions: make(map[string]*qwenDeviceSession),
		stopCh:   make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *qwenSessionStore) Set(sessionID string, session *qwenDeviceSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *qwenSessionStore) Get(sessionID string) (*qwenDeviceSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if time.Since(session.CreatedAt) > qwenSessionTTL {
		return nil, false
	}
	return session, true
}

func (s *qwenSessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *qwenSessionStore) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
}

func (s *qwenSessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				if time.Since(session.CreatedAt) > qwenSessionTTL {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// InitiateDeviceFlow starts the device authorization flow
func (s *QwenOAuthService) InitiateDeviceFlow(ctx context.Context, proxyID *int64) (*QwenDeviceFlowResult, error) {
	// Generate PKCE values
	codeVerifier, err := generateQwenCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("生成 code_verifier 失败: %w", err)
	}
	codeChallenge := generateQwenCodeChallenge(codeVerifier)

	sessionID, err := generateQwenSessionID()
	if err != nil {
		return nil, fmt.Errorf("生成 session_id 失败: %w", err)
	}

	// Resolve proxy URL
	var proxyURL string
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	// POST to device code endpoint
	formData := url.Values{}
	formData.Set("client_id", qwenClientID)
	formData.Set("scope", qwenScope)
	formData.Set("code_challenge", codeChallenge)
	formData.Set("code_challenge_method", "S256")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenDeviceCodeURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpUpstream.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("请求设备码失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("设备码请求失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var deviceResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if deviceResp.DeviceCode == "" {
		return nil, fmt.Errorf("设备码响应缺少 device_code")
	}

	interval := deviceResp.Interval
	if interval <= 0 {
		interval = 5
	}

	// Store session
	session := &qwenDeviceSession{
		DeviceCode:   deviceResp.DeviceCode,
		CodeVerifier: codeVerifier,
		ProxyURL:     proxyURL,
		Interval:     interval,
		ExpiresAt:    time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second),
		CreatedAt:    time.Now(),
	}
	s.sessionStore.Set(sessionID, session)

	return &QwenDeviceFlowResult{
		SessionID:               sessionID,
		UserCode:                deviceResp.UserCode,
		VerificationURL:         deviceResp.VerificationURI,
		VerificationURLComplete: deviceResp.VerificationURIComplete,
		ExpiresIn:               deviceResp.ExpiresIn,
		Interval:                interval,
	}, nil
}

// PollForToken polls the token endpoint waiting for user authorization
func (s *QwenOAuthService) PollForToken(ctx context.Context, sessionID string) (*QwenTokenInfo, error) {
	session, ok := s.sessionStore.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在或已过期")
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionStore.Delete(sessionID)
		return nil, fmt.Errorf("设备码已过期")
	}

	interval := time.Duration(session.Interval) * time.Second

	for attempt := 0; attempt < qwenMaxPollRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		if time.Now().After(session.ExpiresAt) {
			s.sessionStore.Delete(sessionID)
			return nil, fmt.Errorf("设备码已过期")
		}

		tokenInfo, pollErr := s.pollTokenOnce(ctx, session)
		if tokenInfo != nil {
			s.sessionStore.Delete(sessionID)
			return tokenInfo, nil
		}
		if pollErr != nil {
			errMsg := pollErr.Error()
			if strings.Contains(errMsg, "authorization_pending") {
				continue
			}
			if strings.Contains(errMsg, "slow_down") {
				interval += 5 * time.Second
				continue
			}
			// expired_token, access_denied, or other terminal errors
			s.sessionStore.Delete(sessionID)
			return nil, pollErr
		}
	}

	s.sessionStore.Delete(sessionID)
	return nil, fmt.Errorf("轮询超时，用户未完成授权")
}

// pollTokenOnce makes a single token poll request
func (s *QwenOAuthService) pollTokenOnce(ctx context.Context, session *qwenDeviceSession) (*QwenTokenInfo, error) {
	formData := url.Values{}
	formData.Set("grant_type", qwenDeviceGrant)
	formData.Set("client_id", qwenClientID)
	formData.Set("device_code", session.DeviceCode)
	formData.Set("code_verifier", session.CodeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpUpstream.Do(req, session.ProxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("请求 token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		ResourceURL  string `json:"resource_url"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("authorization_pending: 等待用户授权")
	}

	expiresAt := time.Now().Unix() + tokenResp.ExpiresIn - 300

	return &QwenTokenInfo{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		ResourceURL:  tokenResp.ResourceURL,
	}, nil
}

// RefreshToken refreshes a Qwen OAuth token
func (s *QwenOAuthService) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*QwenTokenInfo, error) {
	var lastErr error

	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			time.Sleep(backoff)
		}

		formData := url.Values{}
		formData.Set("grant_type", "refresh_token")
		formData.Set("refresh_token", refreshToken)
		formData.Set("client_id", qwenClientID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenTokenURL, strings.NewReader(formData.Encode()))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := s.httpUpstream.Do(req, proxyURL, 0, 0)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		var tokenResp struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int64  `json:"expires_in"`
			ResourceURL  string `json:"resource_url"`
			Error        string `json:"error"`
			ErrorDesc    string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %w", err)
			continue
		}

		if tokenResp.Error != "" {
			lastErr = fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
			if isNonRetryableQwenOAuthError(lastErr) {
				return nil, lastErr
			}
			continue
		}

		if tokenResp.AccessToken == "" {
			lastErr = fmt.Errorf("token 响应缺少 access_token")
			continue
		}

		expiresAt := time.Now().Unix() + tokenResp.ExpiresIn - 300
		fmt.Printf("[QwenOAuth] Token refreshed: expires_in=%d, expires_at=%d (%s)\n",
			tokenResp.ExpiresIn, expiresAt, time.Unix(expiresAt, 0).Format("2006-01-02 15:04:05"))

		return &QwenTokenInfo{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			TokenType:    tokenResp.TokenType,
			ExpiresIn:    tokenResp.ExpiresIn,
			ExpiresAt:    expiresAt,
			ResourceURL:  tokenResp.ResourceURL,
		}, nil
	}

	return nil, fmt.Errorf("token 刷新失败 (重试后): %w", lastErr)
}

// ValidateRefreshToken validates a refresh token by refreshing it, used for manual RT import
func (s *QwenOAuthService) ValidateRefreshToken(ctx context.Context, refreshToken string, proxyID *int64) (*QwenTokenInfo, error) {
	var proxyURL string
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	tokenInfo, err := s.RefreshToken(ctx, refreshToken, proxyURL)
	if err != nil {
		return nil, err
	}

	return tokenInfo, nil
}

// RefreshAccountToken refreshes token for an Account object
func (s *QwenOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*QwenTokenInfo, error) {
	if account.Platform != PlatformQwen || account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("非 Qwen OAuth 账户")
	}

	refreshToken := account.GetCredential("refresh_token")
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("无可用的 refresh_token")
	}

	var proxyURL string
	if account.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	return s.RefreshToken(ctx, refreshToken, proxyURL)
}

// BuildAccountCredentials builds credentials map for account storage
func (s *QwenOAuthService) BuildAccountCredentials(tokenInfo *QwenTokenInfo) map[string]any {
	creds := map[string]any{
		"access_token": tokenInfo.AccessToken,
		"expires_at":   strconv.FormatInt(tokenInfo.ExpiresAt, 10),
	}
	if tokenInfo.RefreshToken != "" {
		creds["refresh_token"] = tokenInfo.RefreshToken
	}
	if tokenInfo.TokenType != "" {
		creds["token_type"] = tokenInfo.TokenType
	}
	if tokenInfo.ResourceURL != "" {
		creds["resource_url"] = tokenInfo.ResourceURL
	}
	return creds
}

// Stop stops the session store cleanup goroutine
func (s *QwenOAuthService) Stop() {
	s.sessionStore.Stop()
}

// isNonRetryableQwenOAuthError checks if error is non-retryable
func isNonRetryableQwenOAuthError(err error) bool {
	msg := err.Error()
	nonRetryable := []string{
		"invalid_grant",
		"invalid_client",
		"unauthorized_client",
		"access_denied",
	}
	for _, needle := range nonRetryable {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// PKCE helpers
func generateQwenCodeVerifier() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "="), nil
}

func generateQwenCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return strings.TrimRight(base64.URLEncoding.EncodeToString(hash[:]), "=")
}

func generateQwenSessionID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

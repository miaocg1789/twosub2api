package service

import (
	"context"
	"crypto/rand"
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

	"github.com/google/uuid"
)

const (
	kimiDeviceCodeURL = "https://auth.kimi.com/api/oauth/device_authorization"
	kimiTokenURL      = "https://auth.kimi.com/api/oauth/token"
	kimiClientID      = "17e5f671-d194-4dfb-9706-5516cb48c098"
	kimiDeviceGrant   = "urn:ietf:params:oauth:grant-type:device_code"

	kimiSessionTTL     = 30 * time.Minute
	kimiPollInterval   = 5 * time.Second
	kimiMaxPollRetries = 60
)

// KimiOAuthService handles Kimi Device Authorization Grant (RFC 8628)
type KimiOAuthService struct {
	sessionStore *kimiSessionStore
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
}

// NewKimiOAuthService creates a new KimiOAuthService
func NewKimiOAuthService(proxyRepo ProxyRepository, httpUpstream HTTPUpstream) *KimiOAuthService {
	return &KimiOAuthService{
		sessionStore: newKimiSessionStore(),
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
	}
}

// KimiDeviceFlowResult is returned after initiating device authorization
type KimiDeviceFlowResult struct {
	SessionID               string `json:"session_id"`
	UserCode                string `json:"user_code"`
	VerificationURL         string `json:"verification_url"`
	VerificationURLComplete string `json:"verification_url_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// KimiTokenInfo contains token information from Kimi OAuth
type KimiTokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	DeviceID     string `json:"device_id"`
}

// kimiDeviceSession stores device flow session state
type kimiDeviceSession struct {
	DeviceCode string
	DeviceID   string
	ProxyURL   string
	Interval   int
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// kimiSessionStore stores device flow sessions with TTL cleanup
type kimiSessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*kimiDeviceSession
	stopCh   chan struct{}
}

func newKimiSessionStore() *kimiSessionStore {
	store := &kimiSessionStore{
		sessions: make(map[string]*kimiDeviceSession),
		stopCh:   make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *kimiSessionStore) Set(sessionID string, session *kimiDeviceSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = session
}

func (s *kimiSessionStore) Get(sessionID string) (*kimiDeviceSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if time.Since(session.CreatedAt) > kimiSessionTTL {
		return nil, false
	}
	return session, true
}

func (s *kimiSessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *kimiSessionStore) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
}

func (s *kimiSessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			for id, session := range s.sessions {
				if time.Since(session.CreatedAt) > kimiSessionTTL {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// InitiateDeviceFlow starts the device authorization flow
func (s *KimiOAuthService) InitiateDeviceFlow(ctx context.Context, proxyID *int64) (*KimiDeviceFlowResult, error) {
	sessionID, err := generateKimiSessionID()
	if err != nil {
		return nil, fmt.Errorf("生成 session_id 失败: %w", err)
	}

	// Generate device_id for this session
	deviceID := uuid.New().String()

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
	formData.Set("client_id", kimiClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kimiDeviceCodeURL, strings.NewReader(formData.Encode()))
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
	session := &kimiDeviceSession{
		DeviceCode: deviceResp.DeviceCode,
		DeviceID:   deviceID,
		ProxyURL:   proxyURL,
		Interval:   interval,
		ExpiresAt:  time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second),
		CreatedAt:  time.Now(),
	}
	s.sessionStore.Set(sessionID, session)

	return &KimiDeviceFlowResult{
		SessionID:               sessionID,
		UserCode:                deviceResp.UserCode,
		VerificationURL:         deviceResp.VerificationURI,
		VerificationURLComplete: deviceResp.VerificationURIComplete,
		ExpiresIn:               deviceResp.ExpiresIn,
		Interval:                interval,
	}, nil
}

// PollForToken polls the token endpoint waiting for user authorization
func (s *KimiOAuthService) PollForToken(ctx context.Context, sessionID string) (*KimiTokenInfo, error) {
	session, ok := s.sessionStore.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session 不存在或已过期")
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionStore.Delete(sessionID)
		return nil, fmt.Errorf("设备码已过期")
	}

	interval := time.Duration(session.Interval) * time.Second

	for attempt := 0; attempt < kimiMaxPollRetries; attempt++ {
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
func (s *KimiOAuthService) pollTokenOnce(ctx context.Context, session *kimiDeviceSession) (*KimiTokenInfo, error) {
	formData := url.Values{}
	formData.Set("grant_type", kimiDeviceGrant)
	formData.Set("client_id", kimiClientID)
	formData.Set("device_code", session.DeviceCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kimiTokenURL, strings.NewReader(formData.Encode()))
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
		Scope        string `json:"scope"`
		ExpiresIn    int64  `json:"expires_in"`
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

	return &KimiTokenInfo{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		DeviceID:     session.DeviceID,
	}, nil
}

// RefreshToken refreshes a Kimi OAuth token
func (s *KimiOAuthService) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*KimiTokenInfo, error) {
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
		formData.Set("client_id", kimiClientID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, kimiTokenURL, strings.NewReader(formData.Encode()))
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
			Scope        string `json:"scope"`
			ExpiresIn    int64  `json:"expires_in"`
			Error        string `json:"error"`
			ErrorDesc    string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %w", err)
			continue
		}

		if tokenResp.Error != "" {
			lastErr = fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
			if isNonRetryableKimiOAuthError(lastErr) {
				return nil, lastErr
			}
			continue
		}

		if tokenResp.AccessToken == "" {
			lastErr = fmt.Errorf("token 响应缺少 access_token")
			continue
		}

		expiresAt := time.Now().Unix() + tokenResp.ExpiresIn - 300
		fmt.Printf("[KimiOAuth] Token refreshed: expires_in=%d, expires_at=%d (%s)\n",
			tokenResp.ExpiresIn, expiresAt, time.Unix(expiresAt, 0).Format("2006-01-02 15:04:05"))

		// Generate new device_id for refreshed token
		deviceID := uuid.New().String()

		return &KimiTokenInfo{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			TokenType:    tokenResp.TokenType,
			Scope:        tokenResp.Scope,
			ExpiresIn:    tokenResp.ExpiresIn,
			ExpiresAt:    expiresAt,
			DeviceID:     deviceID,
		}, nil
	}

	return nil, fmt.Errorf("token 刷新失败 (重试后): %w", lastErr)
}

// ValidateRefreshToken validates a refresh token by refreshing it, used for manual RT import
func (s *KimiOAuthService) ValidateRefreshToken(ctx context.Context, refreshToken string, proxyID *int64) (*KimiTokenInfo, error) {
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
func (s *KimiOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*KimiTokenInfo, error) {
	if account.Platform != PlatformKimi || account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("非 Kimi OAuth 账户")
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
func (s *KimiOAuthService) BuildAccountCredentials(tokenInfo *KimiTokenInfo) map[string]any {
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
	if tokenInfo.Scope != "" {
		creds["scope"] = tokenInfo.Scope
	}
	if tokenInfo.DeviceID != "" {
		creds["device_id"] = tokenInfo.DeviceID
	}
	return creds
}

// Stop stops the session store cleanup goroutine
func (s *KimiOAuthService) Stop() {
	s.sessionStore.Stop()
}

// isNonRetryableKimiOAuthError checks if error is non-retryable
func isNonRetryableKimiOAuthError(err error) bool {
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

func generateKimiSessionID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

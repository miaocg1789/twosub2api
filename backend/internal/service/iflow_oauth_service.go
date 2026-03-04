package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	iflowPlatformAPIURL = "https://platform.iflow.cn/api/openapi/apikey"
	iflowUserInfoURL    = "https://iflow.cn/api/oauth/getUserInfo"
)

// IFlowOAuthService handles iFlow Cookie-based authentication
type IFlowOAuthService struct {
	proxyRepo    ProxyRepository
	httpUpstream HTTPUpstream
}

// NewIFlowOAuthService creates a new IFlowOAuthService
func NewIFlowOAuthService(proxyRepo ProxyRepository, httpUpstream HTTPUpstream) *IFlowOAuthService {
	return &IFlowOAuthService{
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
	}
}

// IFlowTokenInfo contains token information from iFlow
type IFlowTokenInfo struct {
	APIKey    string `json:"api_key"`
	Cookie    string `json:"cookie"`
	ExpiresAt int64  `json:"expires_at"`
	Email     string `json:"email,omitempty"`
}

// AuthenticateWithCookie fetches API key info using BXAuth cookie
func (s *IFlowOAuthService) AuthenticateWithCookie(ctx context.Context, cookie string, proxyID *int64) (*IFlowTokenInfo, error) {
	// Resolve proxy URL
	var proxyURL string
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	// GET request to fetch API key info
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iflowPlatformAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Cookie", "BXAuth="+cookie)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.httpUpstream.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("请求 API key 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API key 请求失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data struct {
			APIKey   string `json:"apiKey"`
			ExpireAt string `json:"expireAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Data.APIKey == "" {
		return nil, fmt.Errorf("响应中缺少 API key")
	}

	// Parse expiry time
	var expiresAt int64
	if apiResp.Data.ExpireAt != "" {
		// Try parsing as RFC3339 timestamp
		if t, err := time.Parse(time.RFC3339, apiResp.Data.ExpireAt); err == nil {
			expiresAt = t.Unix()
		} else if t, err := time.Parse("2006-01-02", apiResp.Data.ExpireAt); err == nil {
			expiresAt = t.Unix()
		} else {
			// Set a default expiry of 30 days if parsing fails
			expiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
		}
	} else {
		// Default expiry: 30 days
		expiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
	}

	// Optionally fetch user email
	email, _ := s.fetchUserEmail(ctx, cookie, proxyURL)

	return &IFlowTokenInfo{
		APIKey:    apiResp.Data.APIKey,
		Cookie:    cookie,
		ExpiresAt: expiresAt,
		Email:     email,
	}, nil
}

// fetchUserEmail fetches user email from iFlow (best effort, ignores errors)
func (s *IFlowOAuthService) fetchUserEmail(ctx context.Context, cookie string, proxyURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iflowUserInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "BXAuth="+cookie)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.httpUpstream.Do(req, proxyURL, 0, 0)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var userResp struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &userResp); err != nil {
		return "", err
	}

	return userResp.Data.Email, nil
}

// RefreshAPIKey refreshes the API key using the stored cookie
func (s *IFlowOAuthService) RefreshAPIKey(ctx context.Context, cookie string, proxyURL string) (*IFlowTokenInfo, error) {
	// POST request to refresh API key
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, iflowPlatformAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Cookie", "BXAuth="+cookie)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.httpUpstream.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("请求 API key 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API key 刷新失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data struct {
			APIKey   string `json:"apiKey"`
			ExpireAt string `json:"expireAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if apiResp.Data.APIKey == "" {
		return nil, fmt.Errorf("响应中缺少 API key")
	}

	// Parse expiry time
	var expiresAt int64
	if apiResp.Data.ExpireAt != "" {
		if t, err := time.Parse(time.RFC3339, apiResp.Data.ExpireAt); err == nil {
			expiresAt = t.Unix()
		} else if t, err := time.Parse("2006-01-02", apiResp.Data.ExpireAt); err == nil {
			expiresAt = t.Unix()
		} else {
			expiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
		}
	} else {
		expiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
	}

	email, _ := s.fetchUserEmail(ctx, cookie, proxyURL)

	fmt.Printf("[IFlowOAuth] API key refreshed: expires_at=%d (%s)\n",
		expiresAt, time.Unix(expiresAt, 0).Format("2006-01-02 15:04:05"))

	return &IFlowTokenInfo{
		APIKey:    apiResp.Data.APIKey,
		Cookie:    cookie,
		ExpiresAt: expiresAt,
		Email:     email,
	}, nil
}

// RefreshAccountToken refreshes token for an Account object
func (s *IFlowOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*IFlowTokenInfo, error) {
	if account.Platform != PlatformIFlow || account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("非 iFlow OAuth 账户")
	}

	cookie := account.GetCredential("cookie")
	if strings.TrimSpace(cookie) == "" {
		return nil, fmt.Errorf("无可用的 cookie")
	}

	var proxyURL string
	if account.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	return s.RefreshAPIKey(ctx, cookie, proxyURL)
}

// BuildAccountCredentials builds credentials map for account storage
func (s *IFlowOAuthService) BuildAccountCredentials(tokenInfo *IFlowTokenInfo) map[string]any {
	creds := map[string]any{
		"api_key":    tokenInfo.APIKey,
		"cookie":     tokenInfo.Cookie,
		"expires_at": strconv.FormatInt(tokenInfo.ExpiresAt, 10),
	}
	if tokenInfo.Email != "" {
		creds["email"] = tokenInfo.Email
	}
	return creds
}

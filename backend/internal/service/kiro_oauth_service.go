package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
)

// KiroOAuthService handles Kiro token refresh operations
type KiroOAuthService struct {
	httpUpstream HTTPUpstream
	proxyRepo    ProxyRepository
}

// NewKiroOAuthService creates a new KiroOAuthService
func NewKiroOAuthService(proxyRepo ProxyRepository, httpUpstream HTTPUpstream) *KiroOAuthService {
	return &KiroOAuthService{
		proxyRepo:    proxyRepo,
		httpUpstream: httpUpstream,
	}
}

// KiroTokenInfo contains token information from Kiro OAuth
type KiroTokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	AuthType     string `json:"auth_type"`
}

// httpUpstreamAdapter adapts HTTPUpstream to kiro.HTTPClient
type httpUpstreamAdapter struct {
	upstream HTTPUpstream
}

func (a *httpUpstreamAdapter) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return a.upstream.Do(req, proxyURL, accountID, accountConcurrency)
}

// RefreshToken refreshes a Kiro token (Social or IdC)
func (s *KiroOAuthService) RefreshToken(ctx context.Context, refreshToken, authType, clientID, clientSecret, region string, proxyID *int64) (*KiroTokenInfo, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh_token is required")
	}

	var proxyURL string
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	client := &httpUpstreamAdapter{upstream: s.httpUpstream}

	switch authType {
	case kiro.AuthTypeSocial:
		tokenInfo, err := kiro.RefreshSocialToken(client, refreshToken, region, proxyURL)
		if err != nil {
			return nil, fmt.Errorf("social token refresh failed: %w", err)
		}
		return &KiroTokenInfo{
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			ExpiresIn:    tokenInfo.ExpiresIn,
			ExpiresAt:    tokenInfo.ExpiresAt,
			AuthType:     kiro.AuthTypeSocial,
		}, nil

	case kiro.AuthTypeIdC:
		if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
			return nil, fmt.Errorf("client_id and client_secret are required for IdC auth")
		}
		tokenInfo, err := kiro.RefreshIdCToken(client, refreshToken, clientID, clientSecret, region, proxyURL)
		if err != nil {
			return nil, fmt.Errorf("idc token refresh failed: %w", err)
		}
		return &KiroTokenInfo{
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			ExpiresIn:    tokenInfo.ExpiresIn,
			ExpiresAt:    tokenInfo.ExpiresAt,
			AuthType:     kiro.AuthTypeIdC,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported auth type: %s (expected %q or %q)", authType, kiro.AuthTypeSocial, kiro.AuthTypeIdC)
	}
}

// RefreshAccountToken refreshes token for an existing account
func (s *KiroOAuthService) RefreshAccountToken(ctx context.Context, account *Account) (*KiroTokenInfo, error) {
	if account.Platform != PlatformKiro || account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("not a Kiro OAuth account")
	}

	refreshToken := account.GetCredential("refresh_token")
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("no refresh_token available")
	}

	authType := account.GetCredential("auth_type")
	if authType == "" {
		authType = kiro.AuthTypeSocial
	}

	clientID := account.GetCredential("client_id")
	clientSecret := account.GetCredential("client_secret")

	// Region priority: auth_region > region > "us-east-1" (matching Aether/Kiro-Go)
	region := account.GetCredential("auth_region")
	if region == "" {
		region = account.GetCredential("region")
	}

	var proxyURL string
	if account.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	client := &httpUpstreamAdapter{upstream: s.httpUpstream}

	switch authType {
	case kiro.AuthTypeIdC:
		tokenInfo, err := kiro.RefreshIdCToken(client, refreshToken, clientID, clientSecret, region, proxyURL)
		if err != nil {
			return nil, err
		}
		return &KiroTokenInfo{
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			ExpiresIn:    tokenInfo.ExpiresIn,
			ExpiresAt:    tokenInfo.ExpiresAt,
			AuthType:     kiro.AuthTypeIdC,
		}, nil
	default:
		tokenInfo, err := kiro.RefreshSocialToken(client, refreshToken, region, proxyURL)
		if err != nil {
			return nil, err
		}
		return &KiroTokenInfo{
			AccessToken:  tokenInfo.AccessToken,
			RefreshToken: tokenInfo.RefreshToken,
			ExpiresIn:    tokenInfo.ExpiresIn,
			ExpiresAt:    tokenInfo.ExpiresAt,
			AuthType:     kiro.AuthTypeSocial,
		}, nil
	}
}

// BuildAccountCredentials builds credentials map for account storage
func (s *KiroOAuthService) BuildAccountCredentials(tokenInfo *KiroTokenInfo, authType, clientID, clientSecret string) map[string]any {
	creds := map[string]any{
		"access_token":  tokenInfo.AccessToken,
		"refresh_token": tokenInfo.RefreshToken,
		"expires_in":    tokenInfo.ExpiresIn,
		"expires_at":    tokenInfo.ExpiresAt,
		"auth_type":     authType,
	}
	if clientID != "" {
		creds["client_id"] = clientID
	}
	if clientSecret != "" {
		creds["client_secret"] = clientSecret
	}
	return creds
}

// NormalizeKiroCredentials converts Kiro-Go camelCase credential fields to sub2api snake_case format.
// This allows importing accounts exported from Kiro-Go without manual field renaming.
// Safe to call on already-normalized credentials (no-op if snake_case fields exist).
func NormalizeKiroCredentials(creds map[string]any) map[string]any {
	if creds == nil {
		return creds
	}

	// Field mapping: Kiro-Go camelCase → sub2api snake_case
	fieldMap := map[string]string{
		"auth_method":   "auth_type",
		"refreshToken":  "refresh_token",
		"accessToken":   "access_token",
		"clientId":      "client_id",
		"clientSecret":  "client_secret",
		"machineId":     "machine_id",
		"startUrl":      "start_url",
		"expiresAt":     "expires_at",
		"expiresIn":     "expires_in",
		"authType":      "auth_type",
		"authMethod":    "auth_type",
		"authRegion":    "auth_region",
		"apiRegion":     "api_region",
		"profileArn":    "profile_arn",
		"providerType":  "provider_type",
		"kiroVersion":   "kiro_version",
		"systemVersion": "system_version",
		"nodeVersion":   "node_version",
	}

	for camel, snake := range fieldMap {
		if val, ok := creds[camel]; ok {
			if _, exists := creds[snake]; !exists {
				creds[snake] = val
			}
			delete(creds, camel)
		}
	}

	// Normalize auth_type value: handle aliases (matching Aether's behavior)
	if authType, ok := creds["auth_type"].(string); ok {
		normalized := strings.ToLower(strings.TrimSpace(authType))
		switch normalized {
		case "idc", "idc_custom", "builderid", "builder-id", "builder_id", "iam", "enterprise":
			creds["auth_type"] = "IdC"
		case "social", "":
			creds["auth_type"] = "Social"
		}
	}

	// Auto-detect auth_type from provider field or presence of client_id/client_secret
	if _, hasAuthType := creds["auth_type"]; !hasAuthType {
		provider, _ := creds["provider"].(string)
		provider = strings.ToLower(strings.TrimSpace(provider))
		clientID, _ := creds["client_id"].(string)
		clientSecret, _ := creds["client_secret"].(string)
		if provider == "enterprise" || provider == "idc" || provider == "builderid" || provider == "builder-id" || provider == "builder_id" || (clientID != "" && clientSecret != "") {
			creds["auth_type"] = "IdC"
		} else {
			creds["auth_type"] = "Social"
		}
	}

	// Auto-detect OIDC region from client_id for IdC accounts
	if authType, _ := creds["auth_type"].(string); authType == "IdC" {
		clientID, _ := creds["client_id"].(string)
		if clientID != "" {
			detectedRegion := kiro.DetectRegionFromClientID(clientID)
			if detectedRegion != "" {
				// Set auth_region which has highest priority in region resolution
				creds["auth_region"] = detectedRegion
			}
		}
	}

	return creds
}

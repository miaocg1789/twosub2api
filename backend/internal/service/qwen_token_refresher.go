package service

import (
	"context"
	"fmt"
	"time"
)

// QwenTokenRefresher implements TokenRefresher for Qwen OAuth accounts
type QwenTokenRefresher struct {
	qwenOAuthService *QwenOAuthService
}

// NewQwenTokenRefresher creates a Qwen token refresher
func NewQwenTokenRefresher(qwenOAuthService *QwenOAuthService) *QwenTokenRefresher {
	return &QwenTokenRefresher{
		qwenOAuthService: qwenOAuthService,
	}
}

// CanRefresh checks if this refresher handles the given account
func (r *QwenTokenRefresher) CanRefresh(account *Account) bool {
	return account.Platform == PlatformQwen && account.Type == AccountTypeOAuth
}

// NeedsRefresh checks if the account token needs refreshing
func (r *QwenTokenRefresher) NeedsRefresh(account *Account, refreshWindow time.Duration) bool {
	if !r.CanRefresh(account) {
		return false
	}
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		return false
	}
	timeUntilExpiry := time.Until(*expiresAt)
	needsRefresh := timeUntilExpiry < refreshWindow
	if needsRefresh {
		fmt.Printf("[QwenTokenRefresher] Account %d needs refresh: expires_at=%s, time_until_expiry=%v, window=%v\n",
			account.ID, expiresAt.Format("2006-01-02 15:04:05"), timeUntilExpiry, refreshWindow)
	}
	return needsRefresh
}

// Refresh performs the token refresh
func (r *QwenTokenRefresher) Refresh(ctx context.Context, account *Account) (map[string]any, error) {
	tokenInfo, err := r.qwenOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return nil, err
	}

	newCredentials := r.qwenOAuthService.BuildAccountCredentials(tokenInfo)
	// Preserve old credentials fields that are not token-related
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}

	return newCredentials, nil
}

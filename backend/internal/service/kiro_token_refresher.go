package service

import (
	"context"
	"fmt"
	"time"
)

// KiroTokenRefresher implements TokenRefresher for Kiro OAuth accounts
type KiroTokenRefresher struct {
	kiroOAuthService *KiroOAuthService
}

// NewKiroTokenRefresher creates a Kiro token refresher
func NewKiroTokenRefresher(kiroOAuthService *KiroOAuthService) *KiroTokenRefresher {
	return &KiroTokenRefresher{
		kiroOAuthService: kiroOAuthService,
	}
}

// CanRefresh checks if this refresher handles the given account
func (r *KiroTokenRefresher) CanRefresh(account *Account) bool {
	return account.Platform == PlatformKiro && account.Type == AccountTypeOAuth
}

// NeedsRefresh checks if the account token needs refreshing
func (r *KiroTokenRefresher) NeedsRefresh(account *Account, refreshWindow time.Duration) bool {
	if !r.CanRefresh(account) {
		return false
	}
	expiresAt := account.GetCredentialAsTime("expires_at")
	if expiresAt == nil {
		// No expires_at means token was never refreshed (e.g. freshly imported account)
		return true
	}
	timeUntilExpiry := time.Until(*expiresAt)
	needsRefresh := timeUntilExpiry < refreshWindow
	if needsRefresh {
		fmt.Printf("[KiroTokenRefresher] Account %d needs refresh: expires_at=%s, time_until_expiry=%v, window=%v\n",
			account.ID, expiresAt.Format("2006-01-02 15:04:05"), timeUntilExpiry, refreshWindow)
	}
	return needsRefresh
}

// Refresh performs the token refresh
func (r *KiroTokenRefresher) Refresh(ctx context.Context, account *Account) (map[string]any, error) {
	tokenInfo, err := r.kiroOAuthService.RefreshAccountToken(ctx, account)
	if err != nil {
		return nil, err
	}

	authType := account.GetCredential("auth_type")
	clientID := account.GetCredential("client_id")
	clientSecret := account.GetCredential("client_secret")

	newCredentials := r.kiroOAuthService.BuildAccountCredentials(tokenInfo, authType, clientID, clientSecret)
	// Preserve old credentials fields that are not token-related
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}

	return newCredentials, nil
}

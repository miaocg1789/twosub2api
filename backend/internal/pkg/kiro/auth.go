package kiro

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// knownRegions lists AWS regions for auto-detection from client ID.
var knownRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"eu-north-1", "eu-west-1", "eu-west-2", "eu-west-3",
	"eu-central-1", "eu-central-2", "eu-south-1", "eu-south-2",
	"ap-southeast-1", "ap-southeast-2", "ap-southeast-3",
	"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
	"ap-south-1", "ap-south-2", "ap-east-1",
	"sa-east-1", "ca-central-1", "me-south-1", "me-central-1",
	"af-south-1", "il-central-1",
}

// DetectRegionFromClientID attempts to extract the AWS region from a
// base64-encoded IdC client ID. Returns empty string if detection fails.
func DetectRegionFromClientID(clientID string) string {
	if clientID == "" {
		return ""
	}
	// Try multiple base64 variants: URL-safe (with - and _) and standard (with + and /)
	decodings := []*base64.Encoding{
		base64.RawURLEncoding, // URL-safe, no padding (most common for client IDs)
		base64.RawStdEncoding, // standard, no padding
		base64.URLEncoding,    // URL-safe, with padding
		base64.StdEncoding,    // standard, with padding
	}
	for _, enc := range decodings {
		decoded, err := enc.DecodeString(clientID)
		if err != nil {
			continue
		}
		s := string(decoded)
		for _, r := range knownRegions {
			if strings.HasSuffix(s, r) {
				return r
			}
		}
	}
	return ""
}

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error)
}

// TokenInfo contains refreshed token information
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	ExpiresAt    int64 // unix timestamp
}

// RefreshSocialToken refreshes a Social auth token
func RefreshSocialToken(httpClient HTTPClient, refreshToken string, region string, proxyURL string) (*TokenInfo, error) {
	if region == "" {
		region = "us-east-1"
	}
	socialURL := fmt.Sprintf("https://prod.%s.auth.desktop.kiro.dev/refreshToken", region)
	host := fmt.Sprintf("prod.%s.auth.desktop.kiro.dev", region)

	body, err := json.Marshal(map[string]string{
		"refreshToken": refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal social refresh request: %w", err)
	}

	req, err := http.NewRequest("POST", socialURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create social refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("User-Agent", fmt.Sprintf("KiroIDE-%s", KiroVersion))
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Connection", "close")

	resp, err := httpClient.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("social refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read social refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("social refresh failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse social refresh response: %w", err)
	}

	return &TokenInfo{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		ExpiresAt:    time.Now().Unix() + int64(result.ExpiresIn),
	}, nil
}

// RefreshIdCToken refreshes an IdC auth token.
// It auto-detects the OIDC region from the client ID when the provided region
// fails with invalid_grant, trying the detected region as fallback.
func RefreshIdCToken(httpClient HTTPClient, refreshToken, clientID, clientSecret, region, proxyURL string) (*TokenInfo, error) {
	// Auto-detect region from client ID
	detectedRegion := DetectRegionFromClientID(clientID)

	// Build ordered list of regions to try
	if region == "" && detectedRegion != "" {
		region = detectedRegion
	} else if region == "" {
		region = "us-east-1"
	}

	tokenInfo, err := refreshIdCTokenWithRegion(httpClient, refreshToken, clientID, clientSecret, region, proxyURL)
	if err != nil && detectedRegion != "" && detectedRegion != region &&
		strings.Contains(err.Error(), "invalid_grant") {
		// Retry with the detected region
		fmt.Printf("[Kiro] IdC refresh failed with region %s, retrying with detected region %s\n", region, detectedRegion)
		return refreshIdCTokenWithRegion(httpClient, refreshToken, clientID, clientSecret, detectedRegion, proxyURL)
	}
	return tokenInfo, err
}

func refreshIdCTokenWithRegion(httpClient HTTPClient, refreshToken, clientID, clientSecret, region, proxyURL string) (*TokenInfo, error) {
	oidcURL := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", region)

	body, err := json.Marshal(map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"grantType":    "refresh_token",
		"refreshToken": refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal idc refresh request: %w", err)
	}

	req, err := http.NewRequest("POST", oidcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create idc refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", fmt.Sprintf("oidc.%s.amazonaws.com", region))
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/3.738.0 ua/2.1 os/other lang/js md/browser#unknown_unknown api/sso-oidc#3.738.0 m/E KiroIDE")
	req.Header.Set("User-Agent", "node")
	req.Header.Set("Accept", "*/*")

	resp, err := httpClient.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("idc refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read idc refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("idc refresh failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse idc refresh response: %w", err)
	}

	newRefreshToken := result.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	return &TokenInfo{
		AccessToken:  result.AccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    result.ExpiresIn,
		ExpiresAt:    time.Now().Unix() + int64(result.ExpiresIn),
	}, nil
}

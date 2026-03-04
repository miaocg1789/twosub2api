package kiro

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

// setKiroHeaders sets standard Kiro headers on an HTTP request (internal use)
func setKiroHeaders(req *http.Request, accessToken, machineId string) {
	SetKiroHeaders(req, accessToken, machineId)
}

// SetKiroHeaders sets standard Kiro headers on an HTTP request
func SetKiroHeaders(req *http.Request, accessToken, machineId string) {
	var userAgent, amzUserAgent string
	if machineId != "" {
		userAgent = fmt.Sprintf("aws-sdk-js/1.0.27 ua/2.1 os/linux lang/js md/nodejs#20.16.0 api/codewhispererstreaming#1.0.27 m/E KiroIDE-%s-%s", KiroVersion, machineId)
		amzUserAgent = fmt.Sprintf("aws-sdk-js/1.0.27 KiroIDE %s %s", KiroVersion, machineId)
	} else {
		userAgent = fmt.Sprintf("aws-sdk-js/1.0.27 ua/2.1 os/linux lang/js md/nodejs#20.16.0 api/codewhispererstreaming#1.0.27 m/E KiroIDE-%s", KiroVersion)
		amzUserAgent = fmt.Sprintf("aws-sdk-js/1.0.27 KiroIDE-%s", KiroVersion)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("x-amz-user-agent", amzUserAgent)
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	req.Header.Set("x-amzn-kiro-agent-mode", "vibe")
	req.Header.Set("Amz-Sdk-Invocation-Id", uuid.New().String())
	req.Header.Set("Amz-Sdk-Request", "attempt=1; max=3")
	req.Header.Set("Connection", "close")

	// Set Host header from URL
	if req.URL != nil {
		if parsedURL, err := url.Parse(req.URL.String()); err == nil {
			req.Header.Set("Host", parsedURL.Host)
		}
	}
}

// GetUsageLimits calls the CodeWhisperer REST API to get subscription usage limits
func GetUsageLimits(httpClient HTTPClient, accessToken, machineId, proxyURL string) (*UsageLimitsResponse, error) {
	url := CodeWhispererRestAPIBase + "/getUsageLimits?origin=AI_EDITOR&resourceType=AGENTIC_REQUEST&isEmailRequired=true"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create usage limits request: %w", err)
	}

	setKiroHeaders(req, accessToken, machineId)

	resp, err := httpClient.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("usage limits request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read usage limits response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usage limits HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result UsageLimitsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse usage limits response: %w", err)
	}
	return &result, nil
}

// ListAvailableModels calls the CodeWhisperer REST API to get available models
func ListAvailableModels(httpClient HTTPClient, accessToken, machineId, proxyURL string) ([]ModelInfo, error) {
	url := CodeWhispererRestAPIBase + "/ListAvailableModels?origin=AI_EDITOR&maxResults=50"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create list models request: %w", err)
	}

	setKiroHeaders(req, accessToken, machineId)

	resp, err := httpClient.Do(req, proxyURL, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("list models request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read list models response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse list models response: %w", err)
	}
	return result.Models, nil
}

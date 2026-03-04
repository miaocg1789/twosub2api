package service

import (
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
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"

	"github.com/gin-gonic/gin"
)

// GeminiImageVideoService handles Gemini Imagen and Veo API requests,
// converting between OpenAI-compatible format and Gemini native format.
type GeminiImageVideoService struct {
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
	tokenProvider       *GeminiTokenProvider
}

// NewGeminiImageVideoService creates a new GeminiImageVideoService.
func NewGeminiImageVideoService(
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
	tokenProvider *GeminiTokenProvider,
) *GeminiImageVideoService {
	return &GeminiImageVideoService{
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
		tokenProvider:       tokenProvider,
	}
}

// SelectAccountWithLoadAwareness selects a Gemini account for image/video generation.
func (s *GeminiImageVideoService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	accounts, err := s.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, errors.New("no available Gemini accounts")
	}

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
		return nil, errors.New("no available Gemini accounts supporting model " + requestedModel)
	}

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
		Account:     best,
		Acquired:    true,
		ReleaseFunc: func() {},
	}, nil
}

func (s *GeminiImageVideoService) listSchedulableAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, PlatformGemini, false)
		return accounts, err
	}
	if groupID != nil {
		return s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, PlatformGemini)
	}
	return s.accountRepo.ListSchedulableByPlatform(ctx, PlatformGemini)
}

// sizeToAspectRatio converts OpenAI image size to Gemini aspect ratio.
func sizeToAspectRatio(size string) string {
	switch size {
	case "1024x1024", "512x512", "256x256":
		return "1:1"
	case "1024x1792", "1024x1536":
		return "9:16"
	case "1792x1024", "1536x1024":
		return "16:9"
	default:
		return "1:1"
	}
}

// ForwardImageGeneration converts an OpenAI image generation request to Gemini Imagen format,
// forwards it, and converts the response back to OpenAI format.
func (s *GeminiImageVideoService) ForwardImageGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAICompatForwardResult, error) {
	// Parse OpenAI-format request
	var reqBody struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		N              int    `json:"n"`
		Size           string `json:"size"`
		ResponseFormat string `json:"response_format"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	if reqBody.N <= 0 {
		reqBody.N = 1
	}
	if reqBody.Size == "" {
		reqBody.Size = "1024x1024"
	}
	if reqBody.ResponseFormat == "" {
		reqBody.ResponseFormat = "b64_json"
	}

	// Build Gemini Imagen request
	aspectRatio := sizeToAspectRatio(reqBody.Size)
	geminiReq := map[string]any{
		"instances": []map[string]any{
			{"prompt": reqBody.Prompt},
		},
		"parameters": map[string]any{
			"sampleCount": reqBody.N,
			"aspectRatio": aspectRatio,
		},
	}
	geminiBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	// Build upstream URL and auth
	model := reqBody.Model
	req, err := s.buildGeminiAuthRequest(ctx, account, "POST",
		fmt.Sprintf("/v1beta/models/%s:predict", model), geminiBody)
	if err != nil {
		return nil, err
	}

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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		if shouldFailoverGemini(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:   resp.StatusCode,
				ResponseBody: respBody,
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return nil, fmt.Errorf("upstream error %d", resp.StatusCode)
	}

	// Parse Gemini Imagen response
	var geminiResp struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MimeType           string `json:"mimeType"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		// Try returning raw response if parsing fails
		c.Data(resp.StatusCode, "application/json", respBody)
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}

	// Convert to OpenAI format
	openaiData := make([]map[string]any, 0, len(geminiResp.Predictions))
	for _, pred := range geminiResp.Predictions {
		item := map[string]any{}
		if reqBody.ResponseFormat == "url" {
			// For URL format, encode as data URI (Imagen doesn't return URLs)
			mimeType := pred.MimeType
			if mimeType == "" {
				mimeType = "image/png"
			}
			item["url"] = fmt.Sprintf("data:%s;base64,%s", mimeType, pred.BytesBase64Encoded)
		} else {
			item["b64_json"] = pred.BytesBase64Encoded
		}
		item["revised_prompt"] = reqBody.Prompt
		openaiData = append(openaiData, item)
	}

	openaiResp := map[string]any{
		"created": time.Now().Unix(),
		"data":    openaiData,
	}
	openaiRespBytes, _ := json.Marshal(openaiResp)
	c.Data(http.StatusOK, "application/json", openaiRespBytes)

	result := &OpenAICompatForwardResult{
		RequestID: resp.Header.Get("x-request-id"),
		Model:     model,
		ImageSize: classifyImageSize(reqBody.Size),
	}
	result.Usage.CompletionTokens = len(geminiResp.Predictions)

	log.Printf("[GeminiImagen] Image generation completed: model=%s account=%s images=%d",
		model, account.Name, len(geminiResp.Predictions))

	return result, nil
}

// ForwardVideoGeneration handles Gemini Veo video generation requests using LRO (Long-Running Operation).
func (s *GeminiImageVideoService) ForwardVideoGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) error {
	// Parse custom video request
	var reqBody struct {
		Model       string `json:"model"`
		Prompt      string `json:"prompt"`
		Duration    int    `json:"duration"`
		AspectRatio string `json:"aspect_ratio"`
		N           int    `json:"n"`
	}
	if err := json.Unmarshal(body, &reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "Failed to parse request body"}})
		return nil
	}
	if reqBody.Duration <= 0 {
		reqBody.Duration = 5
	}
	if reqBody.AspectRatio == "" {
		reqBody.AspectRatio = "16:9"
	}
	if reqBody.N <= 0 {
		reqBody.N = 1
	}

	// Build Gemini Veo request
	geminiReq := map[string]any{
		"instances": []map[string]any{
			{"prompt": reqBody.Prompt},
		},
		"parameters": map[string]any{
			"videoDuration": fmt.Sprintf("%ds", reqBody.Duration),
			"aspectRatio":   reqBody.AspectRatio,
			"sampleCount":   reqBody.N,
		},
	}
	geminiBody, _ := json.Marshal(geminiReq)

	// Build upstream URL
	model := reqBody.Model
	req, err := s.buildGeminiAuthRequest(ctx, account, "POST",
		fmt.Sprintf("/v1beta/models/%s:generateVideos", model), geminiBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return nil
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// Execute request
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		if shouldFailoverGemini(resp.StatusCode) {
			return &UpstreamFailoverError{
				StatusCode:   resp.StatusCode,
				ResponseBody: respBody,
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return nil
	}

	// Parse LRO response to get operation name
	var lroResp struct {
		Name string `json:"name"`
		Done bool   `json:"done"`
	}
	if err := json.Unmarshal(respBody, &lroResp); err != nil {
		c.Data(resp.StatusCode, "application/json", respBody)
		return nil
	}

	// Poll for completion (max 120s, interval 5s)
	operationName := lroResp.Name
	if lroResp.Done {
		// Already done, extract video URLs
		videos := s.extractVideoURLs(respBody)
		c.JSON(http.StatusOK, gin.H{
			"data":  videos,
			"model": model,
		})
		return nil
	}

	// Poll loop
	pollCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			// Timeout: return operation ID for client to poll later
			c.JSON(http.StatusOK, gin.H{
				"status":       "processing",
				"operation_id": operationName,
				"model":        model,
			})
			return nil
		case <-ticker.C:
			pollReq, err := s.buildGeminiAuthRequest(ctx, account, "GET",
				fmt.Sprintf("/v1beta/%s", operationName), nil)
			if err != nil {
				continue
			}

			pollResp, err := s.httpUpstream.Do(pollReq, proxyURL, account.ID, account.Concurrency)
			if err != nil {
				continue
			}
			pollBody, _ := io.ReadAll(io.LimitReader(pollResp.Body, 10<<20))
			pollResp.Body.Close()

			if pollResp.StatusCode >= 400 {
				continue
			}

			var pollResult struct {
				Name string `json:"name"`
				Done bool   `json:"done"`
			}
			if err := json.Unmarshal(pollBody, &pollResult); err != nil {
				continue
			}

			if pollResult.Done {
				videos := s.extractVideoURLs(pollBody)
				c.JSON(http.StatusOK, gin.H{
					"data":  videos,
					"model": model,
				})
				return nil
			}
		}
	}
}

// PollVideoOperation checks the status of a video generation operation.
func (s *GeminiImageVideoService) PollVideoOperation(ctx context.Context, c *gin.Context, account *Account, operationName string) error {
	req, err := s.buildGeminiAuthRequest(ctx, account, "GET",
		fmt.Sprintf("/v1beta/%s", operationName), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": err.Error()}})
		return nil
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
		return nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return nil
	}

	var result struct {
		Name string `json:"name"`
		Done bool   `json:"done"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		c.Data(resp.StatusCode, "application/json", respBody)
		return nil
	}

	if result.Done {
		videos := s.extractVideoURLs(respBody)
		c.JSON(http.StatusOK, gin.H{
			"status": "completed",
			"data":   videos,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status":       "processing",
			"operation_id": operationName,
		})
	}

	return nil
}

// buildGeminiAuthRequest builds an authenticated HTTP request for the Gemini API.
func (s *GeminiImageVideoService) buildGeminiAuthRequest(ctx context.Context, account *Account, method, path string, body []byte) (*http.Request, error) {
	baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
	fullURL := strings.TrimRight(baseURL, "/") + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := account.GetCredential("api_key")
		if strings.TrimSpace(apiKey) == "" {
			return nil, errors.New("gemini api_key not configured")
		}
		req.Header.Set("x-goog-api-key", apiKey)

	case AccountTypeOAuth:
		if s.tokenProvider == nil {
			return nil, errors.New("gemini token provider not configured")
		}
		accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, fmt.Errorf("get access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

	default:
		return nil, fmt.Errorf("unsupported account type for Gemini: %s", account.Type)
	}

	return req, nil
}

// extractVideoURLs extracts video URLs from a completed Gemini Veo LRO response.
func (s *GeminiImageVideoService) extractVideoURLs(body []byte) []map[string]any {
	var resp struct {
		Response struct {
			GeneratedSamples []struct {
				Video struct {
					URI string `json:"uri"`
				} `json:"video"`
			} `json:"generatedSamples"`
		} `json:"response"`
		// Also handle flat predictions format
		Predictions []struct {
			VideoURI string `json:"videoUri"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}

	var videos []map[string]any
	for _, sample := range resp.Response.GeneratedSamples {
		if sample.Video.URI != "" {
			videos = append(videos, map[string]any{
				"url":          sample.Video.URI,
				"content_type": "video/mp4",
			})
		}
	}
	// Fallback to predictions format
	if len(videos) == 0 {
		for _, pred := range resp.Predictions {
			if pred.VideoURI != "" {
				videos = append(videos, map[string]any{
					"url":          pred.VideoURI,
					"content_type": "video/mp4",
				})
			}
		}
	}

	return videos
}

func shouldFailoverGemini(statusCode int) bool {
	switch statusCode {
	case 401, 403, 429:
		return true
	default:
		return statusCode >= 500
	}
}

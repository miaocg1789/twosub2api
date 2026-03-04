package handler

import (
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
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// OpenAICompatHandler handles OpenAI-compatible API gateway requests
// for DeepSeek, Qwen, GLM, and other OpenAI-compatible upstreams.
type OpenAICompatHandler struct {
	gatewayService      *service.OpenAICompatGatewayService
	geminiService       *service.GeminiImageVideoService
	billingCacheService *service.BillingCacheService
	concurrencyHelper   *ConcurrencyHelper
	maxAccountSwitches  int
}

// NewOpenAICompatHandler creates a new OpenAICompatHandler.
func NewOpenAICompatHandler(
	gatewayService *service.OpenAICompatGatewayService,
	geminiService *service.GeminiImageVideoService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	cfg *config.Config,
) *OpenAICompatHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &OpenAICompatHandler{
		gatewayService:      gatewayService,
		geminiService:       geminiService,
		billingCacheService: billingCacheService,
		concurrencyHelper:   NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		maxAccountSwitches:  maxAccountSwitches,
	}
}

// ChatCompletions handles POST /v1/chat/completions
func (h *OpenAICompatHandler) ChatCompletions(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	// Parse to extract model
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	reqModel, _ := reqBody["model"].(string)
	reqStream, _ := reqBody["stream"].(bool)

	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// Determine platform from model or group
	platform := h.resolvePlatform(reqModel, apiKey)

	streamStarted := false

	// Get subscription info
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	// Acquire user concurrency slot
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted)
	if err != nil {
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error",
			"Concurrency limit exceeded, please retry later", streamStarted)
		return
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// Check billing eligibility
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	// Account selection and forwarding loop with failover
	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, platform, reqModel, failedAccountIDs)
		if err != nil {
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "All accounts failed", streamStarted)
			} else {
				h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
			}
			return
		}
		account := selection.Account

		accountReleaseFunc := selection.ReleaseFunc
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

		// Forward request
		result, err := h.gatewayService.ForwardChatCompletion(c.Request.Context(), c, account, body)
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "All accounts failed", streamStarted)
					return
				}
				switchCount++
				log.Printf("[OpenAICompat] Account %d: upstream error %d, switching %d/%d", account.ID, failoverErr.StatusCode, switchCount, maxAccountSwitches)
				continue
			}
			log.Printf("[OpenAICompat] Account %d: Forward failed: %v", account.ID, err)
			return
		}

		// Async record usage
		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)

		go func(result *service.OpenAICompatForwardResult, usedAccount *service.Account, ua, ipAddr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAICompatRecordUsageInput{
				Result:       result,
				APIKey:       apiKey,
				User:         apiKey.User,
				Account:      usedAccount,
				Subscription: subscription,
				UserAgent:    ua,
				IPAddress:    ipAddr,
			}); err != nil {
				log.Printf("[OpenAICompat] Record usage failed: %v", err)
			}
		}(result, account, userAgent, clientIP)
		return
	}
}

// ImageGenerations handles POST /v1/images/generations
func (h *OpenAICompatHandler) ImageGenerations(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	reqModel, _ := reqBody["model"].(string)
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	platform := h.resolvePlatform(reqModel, apiKey)

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	// Acquire user concurrency slot
	streamStarted := false
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Concurrency limit exceeded, please retry later")
		return
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		status, code, message := billingErrorDetails(err)
		h.errorResponse(c, status, code, message)
		return
	}

	failedAccountIDs := make(map[int64]struct{})
	maxSwitches := h.maxAccountSwitches
	switchCount := 0

	isGeminiImagen := strings.HasPrefix(reqModel, "imagen-")

	for {
		var selection *service.AccountSelectionResult
		var err error
		if isGeminiImagen {
			selection, err = h.geminiService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, reqModel, failedAccountIDs)
		} else {
			selection, err = h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, platform, reqModel, failedAccountIDs)
		}
		if err != nil {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error())
			return
		}
		account := selection.Account

		accountReleaseFunc := selection.ReleaseFunc
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

		var result *service.OpenAICompatForwardResult
		if isGeminiImagen {
			result, err = h.geminiService.ForwardImageGeneration(c.Request.Context(), c, account, body)
		} else {
			result, err = h.gatewayService.ForwardImageGeneration(c.Request.Context(), c, account, body)
		}
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failedAccountIDs[account.ID] = struct{}{}
				if switchCount >= maxSwitches {
					h.errorResponse(c, http.StatusBadGateway, "upstream_error", "All accounts failed")
					return
				}
				switchCount++
				continue
			}
			log.Printf("[OpenAICompat] Image generation failed: %v", err)
			return
		}

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)

		go func(result *service.OpenAICompatForwardResult, usedAccount *service.Account, ua, ipAddr string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = h.gatewayService.RecordUsage(ctx, &service.OpenAICompatRecordUsageInput{
				Result:       result,
				APIKey:       apiKey,
				User:         apiKey.User,
				Account:      usedAccount,
				Subscription: subscription,
				UserAgent:    ua,
				IPAddress:    ipAddr,
			})
		}(result, account, userAgent, clientIP)
		return
	}
}

// VideoGenerations handles POST /v1/videos/generations
func (h *OpenAICompatHandler) VideoGenerations(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	reqModel, _ := reqBody["model"].(string)
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	if !strings.HasPrefix(reqModel, "veo-") {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Unsupported video model: "+reqModel)
		return
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	streamStarted := false
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Concurrency limit exceeded, please retry later")
		return
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		status, code, message := billingErrorDetails(err)
		h.errorResponse(c, status, code, message)
		return
	}

	failedAccountIDs := make(map[int64]struct{})
	maxSwitches := h.maxAccountSwitches
	switchCount := 0

	for {
		selection, err := h.geminiService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, reqModel, failedAccountIDs)
		if err != nil {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error())
			return
		}
		account := selection.Account

		accountReleaseFunc := selection.ReleaseFunc
		accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

		err = h.geminiService.ForwardVideoGeneration(c.Request.Context(), c, account, body)
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failedAccountIDs[account.ID] = struct{}{}
				if switchCount >= maxSwitches {
					h.errorResponse(c, http.StatusBadGateway, "upstream_error", "All accounts failed")
					return
				}
				switchCount++
				continue
			}
			log.Printf("[OpenAICompat] Video generation failed: %v", err)
			return
		}
		return
	}
}

// VideoOperationStatus handles GET /v1/videos/operations/:operationId
func (h *OpenAICompatHandler) VideoOperationStatus(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	operationID := c.Param("operationId")
	if operationID == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "operation_id is required")
		return
	}

	// Construct the full operation name for Gemini API
	operationName := operationID
	if !strings.HasPrefix(operationName, "operations/") {
		operationName = "operations/" + operationName
	}

	// Select a Gemini account (any available one, since operations are not account-specific at lookup)
	selection, err := h.geminiService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, "", nil)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Gemini accounts: "+err.Error())
		return
	}
	account := selection.Account
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}

	_ = h.geminiService.PollVideoOperation(c.Request.Context(), c, account, operationName)
}

// resolvePlatform determines the platform from the model name or group configuration.
func (h *OpenAICompatHandler) resolvePlatform(model string, apiKey *service.APIKey) string {
	// If group has a platform set, use it
	if apiKey.Group != nil && apiKey.Group.Platform != "" {
		return apiKey.Group.Platform
	}
	// Infer from model name prefix
	if strings.HasPrefix(model, "deepseek") {
		return service.PlatformDeepSeek
	}
	if strings.HasPrefix(model, "qwen") || strings.HasPrefix(model, "qwq") {
		return service.PlatformQwen
	}
	if strings.HasPrefix(model, "glm") || strings.HasPrefix(model, "chatglm") || strings.HasPrefix(model, "cogview") {
		return service.PlatformGLM
	}
	// OpenAI image models (DALL-E, GPT-Image)
	if strings.HasPrefix(model, "dall-e") || strings.HasPrefix(model, "gpt-image") {
		return service.PlatformOpenAI
	}
	// Qwen image models (Wanx, Flux, Stable Diffusion via DashScope)
	if strings.HasPrefix(model, "wanx") || strings.HasPrefix(model, "flux-") || strings.HasPrefix(model, "stable-diffusion") {
		return service.PlatformQwen
	}
	// Gemini Imagen models
	if strings.HasPrefix(model, "imagen-") {
		return service.PlatformGemini
	}
	// Gemini Veo video models
	if strings.HasPrefix(model, "veo-") {
		return service.PlatformGemini
	}
	// Default fallback
	return service.PlatformDeepSeek
}

func (h *OpenAICompatHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	if streamStarted {
		flusher, ok := c.Writer.(http.Flusher)
		if ok {
			errorEvent := fmt.Sprintf("data: {\"error\": {\"type\": \"%s\", \"message\": \"%s\"}}\n\n", errType, message)
			fmt.Fprint(c.Writer, errorEvent)
			flusher.Flush()
		}
		return
	}
	h.errorResponse(c, status, errType, message)
}

func (h *OpenAICompatHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// KimiOAuthHandler handles Kimi OAuth device flow endpoints
type KimiOAuthHandler struct {
	kimiOAuthService *service.KimiOAuthService
}

// NewKimiOAuthHandler creates a new KimiOAuthHandler
func NewKimiOAuthHandler(kimiOAuthService *service.KimiOAuthService) *KimiOAuthHandler {
	return &KimiOAuthHandler{kimiOAuthService: kimiOAuthService}
}

// KimiInitiateDeviceFlowRequest represents the request for initiating device flow
type KimiInitiateDeviceFlowRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

// InitiateDeviceFlow starts the Kimi device authorization flow
// POST /api/v1/admin/kimi/oauth/device-flow
func (h *KimiOAuthHandler) InitiateDeviceFlow(c *gin.Context) {
	var req KimiInitiateDeviceFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	result, err := h.kimiOAuthService.InitiateDeviceFlow(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.InternalError(c, "启动设备授权失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// KimiPollTokenRequest represents the request for polling token
type KimiPollTokenRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// PollForToken polls the token endpoint waiting for user authorization
// POST /api/v1/admin/kimi/oauth/poll-token
func (h *KimiOAuthHandler) PollForToken(c *gin.Context) {
	var req KimiPollTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.kimiOAuthService.PollForToken(c.Request.Context(), req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// KimiRefreshTokenRequest represents the request for refreshing token
type KimiRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	ProxyID      *int64 `json:"proxy_id"`
}

// RefreshToken validates a Kimi refresh token and returns full token info
// POST /api/v1/admin/kimi/oauth/refresh-token
func (h *KimiOAuthHandler) RefreshToken(c *gin.Context) {
	var req KimiRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.kimiOAuthService.ValidateRefreshToken(c.Request.Context(), req.RefreshToken, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

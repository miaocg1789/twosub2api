package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// QwenOAuthHandler handles Qwen OAuth device flow endpoints
type QwenOAuthHandler struct {
	qwenOAuthService *service.QwenOAuthService
}

// NewQwenOAuthHandler creates a new QwenOAuthHandler
func NewQwenOAuthHandler(qwenOAuthService *service.QwenOAuthService) *QwenOAuthHandler {
	return &QwenOAuthHandler{qwenOAuthService: qwenOAuthService}
}

// QwenInitiateDeviceFlowRequest represents the request for initiating device flow
type QwenInitiateDeviceFlowRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

// InitiateDeviceFlow starts the Qwen device authorization flow
// POST /api/v1/admin/qwen/oauth/device-flow
func (h *QwenOAuthHandler) InitiateDeviceFlow(c *gin.Context) {
	var req QwenInitiateDeviceFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	result, err := h.qwenOAuthService.InitiateDeviceFlow(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.InternalError(c, "启动设备授权失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// QwenPollTokenRequest represents the request for polling token
type QwenPollTokenRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// PollForToken polls the token endpoint waiting for user authorization
// POST /api/v1/admin/qwen/oauth/poll-token
func (h *QwenOAuthHandler) PollForToken(c *gin.Context) {
	var req QwenPollTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.qwenOAuthService.PollForToken(c.Request.Context(), req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// QwenRefreshTokenRequest represents the request for refreshing token
type QwenRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	ProxyID      *int64 `json:"proxy_id"`
}

// RefreshToken validates a Qwen refresh token and returns full token info
// POST /api/v1/admin/qwen/oauth/refresh-token
func (h *QwenOAuthHandler) RefreshToken(c *gin.Context) {
	var req QwenRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.qwenOAuthService.ValidateRefreshToken(c.Request.Context(), req.RefreshToken, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

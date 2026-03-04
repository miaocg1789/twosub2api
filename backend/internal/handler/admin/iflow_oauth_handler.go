package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// IFlowOAuthHandler handles iFlow OAuth cookie-based authentication endpoints
type IFlowOAuthHandler struct {
	iflowOAuthService *service.IFlowOAuthService
}

// NewIFlowOAuthHandler creates a new IFlowOAuthHandler
func NewIFlowOAuthHandler(iflowOAuthService *service.IFlowOAuthService) *IFlowOAuthHandler {
	return &IFlowOAuthHandler{iflowOAuthService: iflowOAuthService}
}

// IFlowCookieAuthRequest represents the request for cookie-based authentication
type IFlowCookieAuthRequest struct {
	Cookie  string `json:"cookie" binding:"required"`
	ProxyID *int64 `json:"proxy_id"`
}

// AuthenticateWithCookie authenticates using BXAuth cookie and fetches API key
// POST /api/v1/admin/iflow/oauth/cookie-auth
func (h *IFlowOAuthHandler) AuthenticateWithCookie(c *gin.Context) {
	var req IFlowCookieAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.iflowOAuthService.AuthenticateWithCookie(c.Request.Context(), req.Cookie, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

// IFlowRefreshKeyRequest represents the request for refreshing API key
type IFlowRefreshKeyRequest struct {
	Cookie  string `json:"cookie" binding:"required"`
	ProxyID *int64 `json:"proxy_id"`
}

// RefreshAPIKey refreshes the API key using stored cookie
// POST /api/v1/admin/iflow/oauth/refresh-key
func (h *IFlowOAuthHandler) RefreshAPIKey(c *gin.Context) {
	var req IFlowRefreshKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	tokenInfo, err := h.iflowOAuthService.AuthenticateWithCookie(c.Request.Context(), req.Cookie, req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type KiroOAuthHandler struct {
	kiroOAuthService *service.KiroOAuthService
}

func NewKiroOAuthHandler(kiroOAuthService *service.KiroOAuthService) *KiroOAuthHandler {
	return &KiroOAuthHandler{kiroOAuthService: kiroOAuthService}
}

type KiroRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	AuthType     string `json:"auth_type"` // "Social" or "IdC", defaults to "Social"
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Region       string `json:"region"`
	ProxyID      *int64 `json:"proxy_id"`
}

func (h *KiroOAuthHandler) RefreshToken(c *gin.Context) {
	var req KiroRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求无效: "+err.Error())
		return
	}

	authType := req.AuthType
	if authType == "" {
		authType = "Social"
	}

	tokenInfo, err := h.kiroOAuthService.RefreshToken(
		c.Request.Context(),
		req.RefreshToken,
		authType,
		req.ClientID,
		req.ClientSecret,
		req.Region,
		req.ProxyID,
	)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, tokenInfo)
}

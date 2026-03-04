package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ReferralHandler 推荐返利处理器
type ReferralHandler struct {
	referralSvc *service.ReferralService
}

// NewReferralHandler 创建推荐返利处理器
func NewReferralHandler(referralSvc *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralSvc: referralSvc}
}

// GetStats 获取推荐统计
func (h *ReferralHandler) GetStats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	stats, err := h.referralSvc.GetStats(c.Request.Context(), subject.UserID)
	if err != nil {
		response.InternalError(c, "failed to get referral stats")
		return
	}

	response.Success(c, stats)
}

// GetInvitees 获取被邀请用户列表
func (h *ReferralHandler) GetInvitees(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	invitees, total, err := h.referralSvc.GetInvitees(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.InternalError(c, "failed to get invitees")
		return
	}

	response.Success(c, gin.H{
		"items":     invitees,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetCommissions 获取返利记录
func (h *ReferralHandler) GetCommissions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	commissions, total, err := h.referralSvc.GetCommissions(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.InternalError(c, "failed to get commissions")
		return
	}

	response.Success(c, gin.H{
		"items":     commissions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

package handler

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
}

// CreateOrder 创建支付订单
// POST /api/v1/payment/create
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.paymentService.CreateOrder(c.Request.Context(), subject.UserID, req.Amount, req.PaymentMethod)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// GetOrders 获取用户订单列表
// GET /api/v1/payment/orders
func (h *PaymentHandler) GetOrders(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	orders, total, err := h.paymentService.GetUserOrders(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
	})
}

// GetOrderStatus 查询订单状态（用于轮询）
// GET /api/v1/payment/orders/:id/status
func (h *PaymentHandler) GetOrderStatus(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	status, err := h.paymentService.GetOrderStatus(c.Request.Context(), orderID, subject.UserID)
	if err != nil {
		log.Printf("get order status error: orderID=%d userID=%d err=%v", orderID, subject.UserID, err)
		response.ErrorFrom(c, err)
		return
	}

	log.Printf("get order status: orderID=%d userID=%d status=%s", orderID, subject.UserID, status)
	response.Success(c, gin.H{"status": status})
}

// GetPaymentConfig 获取支付配置
// GET /api/v1/payment/config
func (h *PaymentHandler) GetPaymentConfig(c *gin.Context) {
	config, err := h.paymentService.GetPaymentConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, config)
}

// NotifyAlipay 支付宝异步回调
// POST /api/v1/payment/notify/alipay
func (h *PaymentHandler) NotifyAlipay(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	params := make(map[string]string)
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// 判断是支付宝常规还是当面付（都走同一个回调）
	channel := "alipay"
	log.Printf("alipay notify received: trade_no=%s out_trade_no=%s trade_status=%s", params["trade_no"], params["out_trade_no"], params["trade_status"])
	if err := h.paymentService.HandleNotify(c.Request.Context(), channel, params); err != nil {
		log.Printf("alipay notify handle error: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}

	c.String(http.StatusOK, "success")
}

// NotifyWechat 微信支付异步回调
// POST /api/v1/payment/notify/wechat
func (h *PaymentHandler) NotifyWechat(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.XML(http.StatusOK, gin.H{"return_code": "FAIL", "return_msg": "read body error"})
		return
	}

	// 解析 XML 为 map
	params, err := parseWechatXML(body)
	if err != nil {
		c.XML(http.StatusOK, gin.H{"return_code": "FAIL", "return_msg": "parse xml error"})
		return
	}

	if err := h.paymentService.HandleNotify(c.Request.Context(), "wechat", params); err != nil {
		c.XML(http.StatusOK, gin.H{"return_code": "FAIL", "return_msg": err.Error()})
		return
	}

	c.XML(http.StatusOK, gin.H{"return_code": "SUCCESS", "return_msg": "OK"})
}

// NotifyEpay 易支付异步回调
// GET /api/v1/payment/notify/epay
func (h *PaymentHandler) NotifyEpay(c *gin.Context) {
	params := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	if err := h.paymentService.HandleNotify(c.Request.Context(), "epay", params); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}

	c.String(http.StatusOK, "success")
}

// PaymentReturn 支付完成跳转页面
// GET /api/v1/payment/return
func (h *PaymentHandler) PaymentReturn(c *gin.Context) {
	// 重定向到前端钱包页面
	c.Redirect(http.StatusFound, "/wallet")
}

// parseWechatXML 解析微信 XML 通知
func parseWechatXML(data []byte) (map[string]string, error) {
	type xmlEntry struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}
	type xmlMap struct {
		XMLName xml.Name   `xml:"xml"`
		Entries []xmlEntry `xml:",any"`
	}

	var m xmlMap
	if err := xml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(m.Entries))
	for _, e := range m.Entries {
		result[e.XMLName.Local] = e.Value
	}
	return result, nil
}

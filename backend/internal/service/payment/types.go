package payment

// PaymentMethod 支付方式
type PaymentMethod string

const (
	MethodAlipay      PaymentMethod = "alipay"
	MethodAlipayF2F   PaymentMethod = "alipay_f2f"
	MethodWechat      PaymentMethod = "wechat"
	MethodEpay        PaymentMethod = "epay"        // 向后兼容
	MethodEpayAlipay  PaymentMethod = "epay_alipay"  // 易支付-支付宝
	MethodEpayWechat  PaymentMethod = "epay_wechat"  // 易支付-微信
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
	OrderStatusPaid    OrderStatus = "paid"
	OrderStatusExpired OrderStatus = "expired"
	OrderStatusFailed  OrderStatus = "failed"
)

// GatewayConfig 支付网关通用配置
type GatewayConfig struct {
	NotifyURL string // 异步回调 URL
	ReturnURL string // 同步跳转 URL
}

// CreatePayRequest 创建支付请求
type CreatePayRequest struct {
	OrderNo string  // 商户订单号
	Amount  float64 // 支付金额
	Subject string  // 商品描述
}

// CreatePayResult 创建支付结果
type CreatePayResult struct {
	PaymentURL string // 跳转支付 URL（支付宝常规/易支付）
	QRCodeURL  string // 二维码 URL（当面付/微信）
	FormHTML   string // 自动提交表单 HTML（支付宝 PC）
}

// NotifyResult 回调通知验证结果
type NotifyResult struct {
	OrderNo string  // 商户订单号
	TradeNo string  // 第三方交易号
	Amount  float64 // 支付金额
	Success bool    // 是否支付成功
}

// Gateway 支付网关接口
type Gateway interface {
	// CreatePay 创建支付订单
	CreatePay(req *CreatePayRequest) (*CreatePayResult, error)
	// VerifyNotify 验证异步回调签名并解析结果
	VerifyNotify(params map[string]string) (*NotifyResult, error)
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	F2FEnabled bool
	GatewayConfig
}

// WechatConfig 微信支付配置
type WechatConfig struct {
	AppID  string
	MchID  string
	APIKey string
	GatewayConfig
}

// EpayConfig 易支付配置
type EpayConfig struct {
	APIURL string
	PID    string
	Key    string
	Type   string // 支付渠道类型: "alipay" 或 "wxpay"
	GatewayConfig
}

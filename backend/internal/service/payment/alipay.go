package payment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const alipayGatewayURL = "https://openapi.alipay.com/gateway.do"

// AlipayGateway 支付宝支付网关
type AlipayGateway struct {
	config *AlipayConfig
	f2f    bool // 是否当面付模式
}

// NewAlipayGateway 创建支付宝网关（常规 PC 支付）
func NewAlipayGateway(config *AlipayConfig) *AlipayGateway {
	return &AlipayGateway{config: config, f2f: false}
}

// NewAlipayF2FGateway 创建支付宝当面付网关
func NewAlipayF2FGateway(config *AlipayConfig) *AlipayGateway {
	return &AlipayGateway{config: config, f2f: true}
}

func (g *AlipayGateway) CreatePay(req *CreatePayRequest) (*CreatePayResult, error) {
	if g.f2f {
		return g.createF2FPay(req)
	}
	return g.createPagePay(req)
}

// createPagePay 创建支付宝 PC 页面支付
func (g *AlipayGateway) createPagePay(req *CreatePayRequest) (*CreatePayResult, error) {
	bizContent := fmt.Sprintf(`{"out_trade_no":"%s","total_amount":"%.2f","subject":"%s","product_code":"FAST_INSTANT_TRADE_PAY"}`,
		req.OrderNo, req.Amount, req.Subject)

	params := map[string]string{
		"app_id":      g.config.AppID,
		"method":      "alipay.trade.page.pay",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  g.config.NotifyURL,
		"return_url":  g.config.ReturnURL,
		"biz_content": bizContent,
	}

	sign, err := g.rsaSign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay sign: %w", err)
	}
	params["sign"] = sign

	// 构造自动提交表单
	var formBuilder strings.Builder
	formBuilder.WriteString(fmt.Sprintf(`<form id="alipaySubmit" action="%s" method="POST">`, alipayGatewayURL))
	for k, v := range params {
		formBuilder.WriteString(fmt.Sprintf(`<input type="hidden" name="%s" value="%s"/>`, k, escapeHTML(v)))
	}
	formBuilder.WriteString(`<input type="submit" value="Pay"/></form>`)
	formBuilder.WriteString(`<script>document.getElementById("alipaySubmit").submit();</script>`)

	return &CreatePayResult{
		FormHTML: formBuilder.String(),
	}, nil
}

// createF2FPay 创建支付宝当面付（生成二维码）
func (g *AlipayGateway) createF2FPay(req *CreatePayRequest) (*CreatePayResult, error) {
	bizContent := fmt.Sprintf(`{"out_trade_no":"%s","total_amount":"%.2f","subject":"%s"}`,
		req.OrderNo, req.Amount, req.Subject)

	params := map[string]string{
		"app_id":      g.config.AppID,
		"method":      "alipay.trade.precreate",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  g.config.NotifyURL,
		"biz_content": bizContent,
	}

	sign, err := g.rsaSign(params)
	if err != nil {
		return nil, fmt.Errorf("alipay f2f sign: %w", err)
	}
	params["sign"] = sign

	// 发送 HTTP 请求到支付宝网关获取 QR 码
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	resp, err := http.PostForm(alipayGatewayURL, formData)
	if err != nil {
		return nil, fmt.Errorf("alipay f2f request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read alipay f2f response: %w", err)
	}

	// 解析响应 JSON
	var alipayResp struct {
		Response struct {
			Code   string `json:"code"`
			Msg    string `json:"msg"`
			QRCode string `json:"qr_code"`
		} `json:"alipay_trade_precreate_response"`
	}
	if err := json.Unmarshal(body, &alipayResp); err != nil {
		return nil, fmt.Errorf("parse alipay f2f response: %w", err)
	}
	if alipayResp.Response.Code != "10000" {
		return nil, fmt.Errorf("alipay f2f error: code=%s msg=%s (raw: %s)", alipayResp.Response.Code, alipayResp.Response.Msg, string(body))
	}
	if alipayResp.Response.QRCode == "" {
		return nil, fmt.Errorf("alipay f2f: empty qr_code in response")
	}

	return &CreatePayResult{
		QRCodeURL: alipayResp.Response.QRCode,
	}, nil
}

func (g *AlipayGateway) VerifyNotify(params map[string]string) (*NotifyResult, error) {
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("missing sign parameter")
	}

	// 验证签名（先包含 sign_type，失败后排除 sign_type 重试）
	if err := g.rsaVerify(params, sign); err != nil {
		// 尝试排除 sign_type 后验签
		paramsCopy := make(map[string]string, len(params))
		for k, v := range params {
			if k == "sign_type" {
				continue
			}
			paramsCopy[k] = v
		}
		if err2 := g.rsaVerify(paramsCopy, sign); err2 != nil {
			return nil, fmt.Errorf("alipay verify: %w (retry without sign_type: %v)", err, err2)
		}
	}

	tradeStatus := params["trade_status"]
	success := tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED"

	amount := 0.0
	if v := params["total_amount"]; v != "" {
		fmt.Sscanf(v, "%f", &amount)
	}

	return &NotifyResult{
		OrderNo: params["out_trade_no"],
		TradeNo: params["trade_no"],
		Amount:  amount,
		Success: success,
	}, nil
}

// rsaSign RSA2(SHA256WithRSA) 签名
func (g *AlipayGateway) rsaSign(params map[string]string) (string, error) {
	content := buildSignContent(params)

	block, _ := pem.Decode([]byte(formatPrivateKey(g.config.PrivateKey)))
	if block == nil {
		return "", fmt.Errorf("failed to decode private key PEM (key length: %d)", len(g.config.PrivateKey))
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS1
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("parse private key: %w", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an RSA private key")
	}

	h := sha256.New()
	h.Write([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("rsa sign: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// rsaVerify RSA2 验签
func (g *AlipayGateway) rsaVerify(params map[string]string, sign string) error {
	content := buildSignContent(params)

	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("decode sign: %w", err)
	}

	block, _ := pem.Decode([]byte(formatPublicKey(g.config.PublicKey)))
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	rsaPub, err := parseRSAPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(content))
	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, h.Sum(nil), signBytes)
}

// parseRSAPublicKey 尝试多种格式解析 RSA 公钥
func parseRSAPublicKey(der []byte) (*rsa.PublicKey, error) {
	// 1. 尝试 PKIX (X.509 SubjectPublicKeyInfo) 格式
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rsaPub, ok := pub.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
	}

	// 2. 尝试 PKCS1 (RSA 原始格式)
	if rsaPub, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return rsaPub, nil
	}

	// 3. 尝试 X.509 证书格式（从证书中提取公钥）
	if cert, err := x509.ParseCertificate(der); err == nil {
		if rsaPub, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
	}

	return nil, fmt.Errorf("unsupported public key format (tried PKIX, PKCS1, X.509 certificate)")
}

// buildSignContent 构造待签名字符串
func buildSignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(params[k])
	}
	return buf.String()
}

func formatPrivateKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "-----") {
		return raw
	}
	// 清除所有空白字符（用户粘贴时可能带有空格、换行、制表符）
	raw = strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "").Replace(raw)
	// 按 PEM 标准每 64 字符换行
	var buf strings.Builder
	buf.WriteString("-----BEGIN PRIVATE KEY-----\n")
	for i := 0; i < len(raw); i += 64 {
		end := i + 64
		if end > len(raw) {
			end = len(raw)
		}
		buf.WriteString(raw[i:end])
		buf.WriteByte('\n')
	}
	buf.WriteString("-----END PRIVATE KEY-----")
	return buf.String()
}

func formatPublicKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "-----") {
		return raw
	}
	raw = strings.NewReplacer(" ", "", "\n", "", "\r", "", "\t", "").Replace(raw)
	var buf strings.Builder
	buf.WriteString("-----BEGIN PUBLIC KEY-----\n")
	for i := 0; i < len(raw); i += 64 {
		end := i + 64
		if end > len(raw) {
			end = len(raw)
		}
		buf.WriteString(raw[i:end])
		buf.WriteByte('\n')
	}
	buf.WriteString("-----END PUBLIC KEY-----")
	return buf.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

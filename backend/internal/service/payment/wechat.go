package payment

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const wechatUnifiedOrderURL = "https://api.mch.weixin.qq.com/pay/unifiedorder"

// WechatGateway 微信支付网关（Native 扫码支付）
type WechatGateway struct {
	config *WechatConfig
}

// NewWechatGateway 创建微信支付网关
func NewWechatGateway(config *WechatConfig) *WechatGateway {
	return &WechatGateway{config: config}
}

type wechatUnifiedOrderRequest struct {
	XMLName        xml.Name `xml:"xml"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	Body           string   `xml:"body"`
	OutTradeNo     string   `xml:"out_trade_no"`
	TotalFee       int      `xml:"total_fee"`
	SpbillCreateIP string   `xml:"spbill_create_ip"`
	NotifyURL      string   `xml:"notify_url"`
	TradeType      string   `xml:"trade_type"`
}

type wechatUnifiedOrderResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
	ResultCode string   `xml:"result_code"`
	ErrCode    string   `xml:"err_code"`
	ErrCodeDes string   `xml:"err_code_des"`
	CodeURL    string   `xml:"code_url"`
	PrepayID   string   `xml:"prepay_id"`
}

type wechatNotifyRequest struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ResultCode    string   `xml:"result_code"`
	Sign          string   `xml:"sign"`
	OutTradeNo    string   `xml:"out_trade_no"`
	TransactionID string   `xml:"transaction_id"`
	TotalFee      int      `xml:"total_fee"`
}

func (g *WechatGateway) CreatePay(req *CreatePayRequest) (*CreatePayResult, error) {
	nonceStr := generateNonceStr()
	totalFee := int(req.Amount * 100) // 转为分

	params := map[string]string{
		"appid":            g.config.AppID,
		"mch_id":           g.config.MchID,
		"nonce_str":        nonceStr,
		"body":             req.Subject,
		"out_trade_no":     req.OrderNo,
		"total_fee":        fmt.Sprintf("%d", totalFee),
		"spbill_create_ip": "127.0.0.1",
		"notify_url":       g.config.NotifyURL,
		"trade_type":       "NATIVE",
	}

	sign := g.md5Sign(params)

	xmlReq := &wechatUnifiedOrderRequest{
		AppID:          g.config.AppID,
		MchID:          g.config.MchID,
		NonceStr:       nonceStr,
		Sign:           sign,
		Body:           req.Subject,
		OutTradeNo:     req.OrderNo,
		TotalFee:       totalFee,
		SpbillCreateIP: "127.0.0.1",
		NotifyURL:      g.config.NotifyURL,
		TradeType:      "NATIVE",
	}

	xmlData, err := xml.Marshal(xmlReq)
	if err != nil {
		return nil, fmt.Errorf("marshal wechat request: %w", err)
	}

	resp, err := http.Post(wechatUnifiedOrderURL, "application/xml", strings.NewReader(string(xmlData)))
	if err != nil {
		return nil, fmt.Errorf("wechat unified order request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat response: %w", err)
	}

	var xmlResp wechatUnifiedOrderResponse
	if err := xml.Unmarshal(body, &xmlResp); err != nil {
		return nil, fmt.Errorf("unmarshal wechat response: %w", err)
	}

	if xmlResp.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat error: %s", xmlResp.ReturnMsg)
	}
	if xmlResp.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat business error: %s - %s", xmlResp.ErrCode, xmlResp.ErrCodeDes)
	}

	return &CreatePayResult{
		QRCodeURL: xmlResp.CodeURL,
	}, nil
}

func (g *WechatGateway) VerifyNotify(params map[string]string) (*NotifyResult, error) {
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("missing sign parameter")
	}

	expectedSign := g.md5Sign(params)
	if !strings.EqualFold(sign, expectedSign) {
		return nil, fmt.Errorf("wechat sign mismatch")
	}

	success := params["return_code"] == "SUCCESS" && params["result_code"] == "SUCCESS"

	totalFee := 0
	if v := params["total_fee"]; v != "" {
		fmt.Sscanf(v, "%d", &totalFee)
	}

	return &NotifyResult{
		OrderNo: params["out_trade_no"],
		TradeNo: params["transaction_id"],
		Amount:  float64(totalFee) / 100.0,
		Success: success,
	}, nil
}

// md5Sign 微信 MD5 签名
func (g *WechatGateway) md5Sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || params[k] == "" {
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
	buf.WriteString("&key=")
	buf.WriteString(g.config.APIKey)

	h := md5.Sum([]byte(buf.String()))
	return strings.ToUpper(hex.EncodeToString(h[:]))
}

func generateNonceStr() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

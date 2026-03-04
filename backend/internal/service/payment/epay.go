package payment

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// EpayGateway 通用易支付网关
type EpayGateway struct {
	config *EpayConfig
}

// NewEpayGateway 创建易支付网关
func NewEpayGateway(config *EpayConfig) *EpayGateway {
	return &EpayGateway{config: config}
}

func (g *EpayGateway) CreatePay(req *CreatePayRequest) (*CreatePayResult, error) {
	payType := g.config.Type
	if payType == "" {
		payType = "alipay" // 默认支付宝
	}
	params := map[string]string{
		"pid":          g.config.PID,
		"type":         payType,
		"out_trade_no": req.OrderNo,
		"notify_url":   g.config.NotifyURL,
		"return_url":   g.config.ReturnURL,
		"name":         req.Subject,
		"money":        fmt.Sprintf("%.2f", req.Amount),
	}

	sign := g.md5Sign(params)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	// 构造跳转 URL
	apiURL := strings.TrimRight(g.config.APIURL, "/") + "/submit.php"
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	return &CreatePayResult{
		PaymentURL: apiURL + "?" + query.Encode(),
	}, nil
}

func (g *EpayGateway) VerifyNotify(params map[string]string) (*NotifyResult, error) {
	sign := params["sign"]
	if sign == "" {
		return nil, fmt.Errorf("missing sign parameter")
	}

	expectedSign := g.md5Sign(params)
	if !strings.EqualFold(sign, expectedSign) {
		return nil, fmt.Errorf("epay sign mismatch")
	}

	success := params["trade_status"] == "TRADE_SUCCESS"

	amount := 0.0
	if v := params["money"]; v != "" {
		fmt.Sscanf(v, "%f", &amount)
	}

	return &NotifyResult{
		OrderNo: params["out_trade_no"],
		TradeNo: params["trade_no"],
		Amount:  amount,
		Success: success,
	}, nil
}

// md5Sign 易支付 MD5 签名
func (g *EpayGateway) md5Sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" || params[k] == "" {
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
	buf.WriteString(g.config.Key)

	h := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(h[:])
}

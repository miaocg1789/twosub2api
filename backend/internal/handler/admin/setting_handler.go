package admin

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SettingHandler 系统设置处理器
type SettingHandler struct {
	settingService   *service.SettingService
	emailService     *service.EmailService
	turnstileService *service.TurnstileService
	opsService       *service.OpsService
	soraS3Storage    *service.SoraS3Storage
}

// NewSettingHandler 创建系统设置处理器
func NewSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, opsService *service.OpsService, soraS3Storage *service.SoraS3Storage) *SettingHandler {
	return &SettingHandler{
		settingService:   settingService,
		emailService:     emailService,
		turnstileService: turnstileService,
		opsService:       opsService,
		soraS3Storage:    soraS3Storage,
	}
}

// GetSettings 获取所有系统设置
// GET /api/v1/admin/settings
func (h *SettingHandler) GetSettings(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Check if ops monitoring is enabled (respects config.ops.enabled)
	opsEnabled := h.opsService != nil && h.opsService.IsMonitoringEnabled(c.Request.Context())

	response.Success(c, dto.SystemSettings{
		RegistrationEnabled:                  settings.RegistrationEnabled,
		EmailVerifyEnabled:                   settings.EmailVerifyEnabled,
		PromoCodeEnabled:                     settings.PromoCodeEnabled,
		PasswordResetEnabled:                 settings.PasswordResetEnabled,
		InvitationCodeEnabled:                settings.InvitationCodeEnabled,
		TotpEnabled:                          settings.TotpEnabled,
		TotpEncryptionKeyConfigured:          h.settingService.IsTotpEncryptionKeyConfigured(),
		SMTPHost:                             settings.SMTPHost,
		SMTPPort:                             settings.SMTPPort,
		SMTPUsername:                         settings.SMTPUsername,
		SMTPPasswordConfigured:               settings.SMTPPasswordConfigured,
		SMTPFrom:                             settings.SMTPFrom,
		SMTPFromName:                         settings.SMTPFromName,
		SMTPUseTLS:                           settings.SMTPUseTLS,
		TurnstileEnabled:                     settings.TurnstileEnabled,
		TurnstileSiteKey:                     settings.TurnstileSiteKey,
		TurnstileSecretKeyConfigured:         settings.TurnstileSecretKeyConfigured,
		LinuxDoConnectEnabled:                settings.LinuxDoConnectEnabled,
		LinuxDoConnectClientID:               settings.LinuxDoConnectClientID,
		LinuxDoConnectClientSecretConfigured: settings.LinuxDoConnectClientSecretConfigured,
		LinuxDoConnectRedirectURL:            settings.LinuxDoConnectRedirectURL,
		SiteName:                             settings.SiteName,
		SiteLogo:                             settings.SiteLogo,
		SiteSubtitle:                         settings.SiteSubtitle,
		APIBaseURL:                           settings.APIBaseURL,
		ContactInfo:                          settings.ContactInfo,
		DocURL:                               settings.DocURL,
		HomeContent:                          settings.HomeContent,
		HideCcsImportButton:                  settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:          settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:              settings.PurchaseSubscriptionURL,
		SoraClientEnabled:                    settings.SoraClientEnabled,
		DefaultConcurrency:                   settings.DefaultConcurrency,
		DefaultBalance:                       settings.DefaultBalance,
		EnableModelFallback:                  settings.EnableModelFallback,
		FallbackModelAnthropic:               settings.FallbackModelAnthropic,
		FallbackModelOpenAI:                  settings.FallbackModelOpenAI,
		FallbackModelGemini:                  settings.FallbackModelGemini,
		FallbackModelAntigravity:             settings.FallbackModelAntigravity,
		EnableIdentityPatch:                  settings.EnableIdentityPatch,
		IdentityPatchPrompt:                  settings.IdentityPatchPrompt,
		OpsMonitoringEnabled:                 opsEnabled && settings.OpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:         settings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                  settings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:            settings.OpsMetricsIntervalSeconds,
		PaymentEnabled:                       settings.PaymentEnabled,
		PaymentCurrency:                      settings.PaymentCurrency,
		PaymentExchangeRate:                  settings.PaymentExchangeRate,
		PaymentPresetAmounts:                 settings.PaymentPresetAmounts,
		PaymentMinAmount:                     settings.PaymentMinAmount,
		PaymentMaxAmount:                     settings.PaymentMaxAmount,
		AlipayEnabled:                        settings.AlipayEnabled,
		AlipayAppID:                          settings.AlipayAppID,
		AlipayPrivateKeyConfigured:           settings.AlipayPrivateKeyConfigured,
		AlipayPublicKeyConfigured:            settings.AlipayPublicKeyConfigured,
		AlipayF2FEnabled:                     settings.AlipayF2FEnabled,
		WechatEnabled:                        settings.WechatEnabled,
		WechatAppID:                          settings.WechatAppID,
		WechatMchID:                          settings.WechatMchID,
		WechatAPIKeyConfigured:               settings.WechatAPIKeyConfigured,
		EpayEnabled:                          settings.EpayEnabled,
		EpayAPIURL:                           settings.EpayAPIURL,
		EpayPID:                              settings.EpayPID,
		EpayKeyConfigured:                    settings.EpayKeyConfigured,
		EpayType:                             settings.EpayType,
		ReferralEnabled:                      settings.ReferralEnabled,
		ReferralCommissionRate:               settings.ReferralCommissionRate,
	})
}

// UpdateSettingsRequest 更新设置请求
// 所有字段使用指针类型，区分"未传"和"传了零值"，避免不同设置页面互相覆盖
type UpdateSettingsRequest struct {
	// 注册设置
	RegistrationEnabled   *bool `json:"registration_enabled"`
	EmailVerifyEnabled    *bool `json:"email_verify_enabled"`
	PromoCodeEnabled      *bool `json:"promo_code_enabled"`
	PasswordResetEnabled  *bool `json:"password_reset_enabled"`
	InvitationCodeEnabled *bool `json:"invitation_code_enabled"`
	TotpEnabled           *bool `json:"totp_enabled"`

	// 邮件服务设置
	SMTPHost     *string `json:"smtp_host"`
	SMTPPort     *int    `json:"smtp_port"`
	SMTPUsername *string `json:"smtp_username"`
	SMTPPassword string  `json:"smtp_password"` // 敏感字段保持 string，空=不修改
	SMTPFrom     *string `json:"smtp_from_email"`
	SMTPFromName *string `json:"smtp_from_name"`
	SMTPUseTLS   *bool   `json:"smtp_use_tls"`

	// Cloudflare Turnstile 设置
	TurnstileEnabled   *bool   `json:"turnstile_enabled"`
	TurnstileSiteKey   *string `json:"turnstile_site_key"`
	TurnstileSecretKey string  `json:"turnstile_secret_key"` // 敏感字段

	// LinuxDo Connect OAuth 登录
	LinuxDoConnectEnabled      *bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID     *string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecret string  `json:"linuxdo_connect_client_secret"` // 敏感字段
	LinuxDoConnectRedirectURL  *string `json:"linuxdo_connect_redirect_url"`

	// OEM设置
	SiteName                    *string  `json:"site_name"`
	SiteLogo                    *string  `json:"site_logo"`
	SiteSubtitle                *string  `json:"site_subtitle"`
	APIBaseURL                  *string  `json:"api_base_url"`
	ContactInfo                 *string  `json:"contact_info"`
	DocURL                      *string  `json:"doc_url"`
	HomeContent                 *string  `json:"home_content"`
	HideCcsImportButton         *bool    `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled *bool    `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     *string  `json:"purchase_subscription_url"`
	SoraClientEnabled           *bool    `json:"sora_client_enabled"`

	// 默认配置
	DefaultConcurrency *int     `json:"default_concurrency"`
	DefaultBalance     *float64 `json:"default_balance"`

	// Model fallback configuration
	EnableModelFallback      *bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   *string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      *string `json:"fallback_model_openai"`
	FallbackModelGemini      *string `json:"fallback_model_gemini"`
	FallbackModelAntigravity *string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch *bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt *string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         *bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled *bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          *string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    *int    `json:"ops_metrics_interval_seconds"`

	// Payment settings (使用指针类型，区分"未传"和"传了零值")
	PaymentEnabled       *bool    `json:"payment_enabled"`
	PaymentCurrency      *string  `json:"payment_currency"`
	PaymentExchangeRate  *float64 `json:"payment_exchange_rate"`
	PaymentPresetAmounts *string  `json:"payment_preset_amounts"`
	PaymentMinAmount     *float64 `json:"payment_min_amount"`
	PaymentMaxAmount     *float64 `json:"payment_max_amount"`
	AlipayEnabled        *bool    `json:"payment_alipay_enabled"`
	AlipayAppID          *string  `json:"payment_alipay_app_id"`
	AlipayPrivateKey     string   `json:"payment_alipay_private_key"`
	AlipayPublicKey      string   `json:"payment_alipay_public_key"`
	AlipayF2FEnabled     *bool    `json:"payment_alipay_f2f_enabled"`
	WechatEnabled        *bool    `json:"payment_wechat_enabled"`
	WechatAppID          *string  `json:"payment_wechat_app_id"`
	WechatMchID          *string  `json:"payment_wechat_mch_id"`
	WechatAPIKey         string   `json:"payment_wechat_api_key"`
	EpayEnabled          *bool    `json:"payment_epay_enabled"`
	EpayAPIURL           *string  `json:"payment_epay_api_url"`
	EpayPID              *string  `json:"payment_epay_pid"`
	EpayKey              string   `json:"payment_epay_key"`
	EpayType             *string  `json:"payment_epay_type"`
	ReferralEnabled        *bool    `json:"referral_enabled"`
	ReferralCommissionRate *float64 `json:"referral_commission_rate"`
}

// UpdateSettings 更新系统设置
// PUT /api/v1/admin/settings
func (h *SettingHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	previousSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 验证参数
	if req.DefaultConcurrency != nil && *req.DefaultConcurrency < 1 {
		v := 1
		req.DefaultConcurrency = &v
	}
	if req.DefaultBalance != nil && *req.DefaultBalance < 0 {
		v := 0.0
		req.DefaultBalance = &v
	}
	if req.SMTPPort != nil && *req.SMTPPort <= 0 {
		v := 587
		req.SMTPPort = &v
	}

	// Turnstile 参数验证
	turnstileEnabled := previousSettings.TurnstileEnabled
	if req.TurnstileEnabled != nil {
		turnstileEnabled = *req.TurnstileEnabled
	}
	turnstileSiteKey := previousSettings.TurnstileSiteKey
	if req.TurnstileSiteKey != nil {
		turnstileSiteKey = *req.TurnstileSiteKey
	}
	if turnstileEnabled {
		if turnstileSiteKey == "" {
			response.BadRequest(c, "Turnstile Site Key is required when enabled")
			return
		}
		if req.TurnstileSecretKey == "" {
			if previousSettings.TurnstileSecretKey == "" {
				response.BadRequest(c, "Turnstile Secret Key is required when enabled")
				return
			}
			req.TurnstileSecretKey = previousSettings.TurnstileSecretKey
		}
		siteKeyChanged := previousSettings.TurnstileSiteKey != turnstileSiteKey
		secretKeyChanged := previousSettings.TurnstileSecretKey != req.TurnstileSecretKey
		if siteKeyChanged || secretKeyChanged {
			if err := h.turnstileService.ValidateSecretKey(c.Request.Context(), req.TurnstileSecretKey); err != nil {
				response.ErrorFrom(c, err)
				return
			}
		}
	}

	// TOTP 双因素认证参数验证
	if req.TotpEnabled != nil && *req.TotpEnabled && !previousSettings.TotpEnabled {
		if !h.settingService.IsTotpEncryptionKeyConfigured() {
			response.BadRequest(c, "Cannot enable TOTP: TOTP_ENCRYPTION_KEY environment variable must be configured first. Generate a key with 'openssl rand -hex 32' and set it in your environment.")
			return
		}
	}

	// LinuxDo Connect 参数验证
	linuxDoEnabled := previousSettings.LinuxDoConnectEnabled
	if req.LinuxDoConnectEnabled != nil {
		linuxDoEnabled = *req.LinuxDoConnectEnabled
	}
	if linuxDoEnabled && req.LinuxDoConnectEnabled != nil {
		clientID := ""
		if req.LinuxDoConnectClientID != nil {
			clientID = strings.TrimSpace(*req.LinuxDoConnectClientID)
			req.LinuxDoConnectClientID = &clientID
		} else {
			clientID = previousSettings.LinuxDoConnectClientID
		}
		req.LinuxDoConnectClientSecret = strings.TrimSpace(req.LinuxDoConnectClientSecret)
		redirectURL := ""
		if req.LinuxDoConnectRedirectURL != nil {
			redirectURL = strings.TrimSpace(*req.LinuxDoConnectRedirectURL)
			req.LinuxDoConnectRedirectURL = &redirectURL
		} else {
			redirectURL = previousSettings.LinuxDoConnectRedirectURL
		}

		if clientID == "" {
			response.BadRequest(c, "LinuxDo Client ID is required when enabled")
			return
		}
		if redirectURL == "" {
			response.BadRequest(c, "LinuxDo Redirect URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(redirectURL); err != nil {
			response.BadRequest(c, "LinuxDo Redirect URL must be an absolute http(s) URL")
			return
		}
		if req.LinuxDoConnectClientSecret == "" {
			if previousSettings.LinuxDoConnectClientSecret == "" {
				response.BadRequest(c, "LinuxDo Client Secret is required when enabled")
				return
			}
			req.LinuxDoConnectClientSecret = previousSettings.LinuxDoConnectClientSecret
		}
	}

	// “购买订阅”页面配置验证
	purchaseEnabled := previousSettings.PurchaseSubscriptionEnabled
	if req.PurchaseSubscriptionEnabled != nil {
		purchaseEnabled = *req.PurchaseSubscriptionEnabled
	}
	purchaseURL := previousSettings.PurchaseSubscriptionURL
	if req.PurchaseSubscriptionURL != nil {
		purchaseURL = strings.TrimSpace(*req.PurchaseSubscriptionURL)
	}

	// - 启用时要求 URL 合法且非空
	// - 禁用时允许为空；若提供了 URL 也做基本校验，避免误配置
	if purchaseEnabled {
		if purchaseURL == "" {
			response.BadRequest(c, "Purchase Subscription URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	} else if purchaseURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	}

	// Ops metrics collector interval validation (seconds).
	if req.OpsMetricsIntervalSeconds != nil {
		v := *req.OpsMetricsIntervalSeconds
		if v < 60 {
			v = 60
		}
		if v > 3600 {
			v = 3600
		}
		req.OpsMetricsIntervalSeconds = &v
	}

	// 以 previousSettings 为基础，只覆盖请求中明确传递的字段
	settings := &service.SystemSettings{}
	*settings = *previousSettings

	// 注册设置
	if req.RegistrationEnabled != nil {
		settings.RegistrationEnabled = *req.RegistrationEnabled
	}
	if req.EmailVerifyEnabled != nil {
		settings.EmailVerifyEnabled = *req.EmailVerifyEnabled
	}
	if req.PromoCodeEnabled != nil {
		settings.PromoCodeEnabled = *req.PromoCodeEnabled
	}
	if req.PasswordResetEnabled != nil {
		settings.PasswordResetEnabled = *req.PasswordResetEnabled
	}
	if req.InvitationCodeEnabled != nil {
		settings.InvitationCodeEnabled = *req.InvitationCodeEnabled
	}
	if req.TotpEnabled != nil {
		settings.TotpEnabled = *req.TotpEnabled
	}
	// 邮件服务
	if req.SMTPHost != nil {
		settings.SMTPHost = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		settings.SMTPPort = *req.SMTPPort
	}
	if req.SMTPUsername != nil {
		settings.SMTPUsername = *req.SMTPUsername
	}
	if req.SMTPPassword != "" {
		settings.SMTPPassword = req.SMTPPassword
	}
	if req.SMTPFrom != nil {
		settings.SMTPFrom = *req.SMTPFrom
	}
	if req.SMTPFromName != nil {
		settings.SMTPFromName = *req.SMTPFromName
	}
	if req.SMTPUseTLS != nil {
		settings.SMTPUseTLS = *req.SMTPUseTLS
	}
	// Turnstile
	if req.TurnstileEnabled != nil {
		settings.TurnstileEnabled = *req.TurnstileEnabled
	}
	if req.TurnstileSiteKey != nil {
		settings.TurnstileSiteKey = *req.TurnstileSiteKey
	}
	if req.TurnstileSecretKey != "" {
		settings.TurnstileSecretKey = req.TurnstileSecretKey
	}
	// LinuxDo Connect
	if req.LinuxDoConnectEnabled != nil {
		settings.LinuxDoConnectEnabled = *req.LinuxDoConnectEnabled
	}
	if req.LinuxDoConnectClientID != nil {
		settings.LinuxDoConnectClientID = *req.LinuxDoConnectClientID
	}
	if req.LinuxDoConnectClientSecret != "" {
		settings.LinuxDoConnectClientSecret = req.LinuxDoConnectClientSecret
	}
	if req.LinuxDoConnectRedirectURL != nil {
		settings.LinuxDoConnectRedirectURL = *req.LinuxDoConnectRedirectURL
	}
	// OEM
	if req.SiteName != nil {
		settings.SiteName = *req.SiteName
	}
	if req.SiteLogo != nil {
		settings.SiteLogo = *req.SiteLogo
	}
	if req.SiteSubtitle != nil {
		settings.SiteSubtitle = *req.SiteSubtitle
	}
	if req.APIBaseURL != nil {
		settings.APIBaseURL = *req.APIBaseURL
	}
	if req.ContactInfo != nil {
		settings.ContactInfo = *req.ContactInfo
	}
	if req.DocURL != nil {
		settings.DocURL = *req.DocURL
	}
	if req.HomeContent != nil {
		settings.HomeContent = *req.HomeContent
	}
	if req.HideCcsImportButton != nil {
		settings.HideCcsImportButton = *req.HideCcsImportButton
	}
	settings.PurchaseSubscriptionEnabled = purchaseEnabled
	settings.PurchaseSubscriptionURL = purchaseURL
	if req.SoraClientEnabled != nil {
		settings.SoraClientEnabled = *req.SoraClientEnabled
	}
	// 默认配置
	if req.DefaultConcurrency != nil {
		settings.DefaultConcurrency = *req.DefaultConcurrency
	}
	if req.DefaultBalance != nil {
		settings.DefaultBalance = *req.DefaultBalance
	}
	// Model fallback
	if req.EnableModelFallback != nil {
		settings.EnableModelFallback = *req.EnableModelFallback
	}
	if req.FallbackModelAnthropic != nil {
		settings.FallbackModelAnthropic = *req.FallbackModelAnthropic
	}
	if req.FallbackModelOpenAI != nil {
		settings.FallbackModelOpenAI = *req.FallbackModelOpenAI
	}
	if req.FallbackModelGemini != nil {
		settings.FallbackModelGemini = *req.FallbackModelGemini
	}
	if req.FallbackModelAntigravity != nil {
		settings.FallbackModelAntigravity = *req.FallbackModelAntigravity
	}
	// Identity patch
	if req.EnableIdentityPatch != nil {
		settings.EnableIdentityPatch = *req.EnableIdentityPatch
	}
	if req.IdentityPatchPrompt != nil {
		settings.IdentityPatchPrompt = *req.IdentityPatchPrompt
	}
	// Ops monitoring
	if req.OpsMonitoringEnabled != nil {
		settings.OpsMonitoringEnabled = *req.OpsMonitoringEnabled
	}
	if req.OpsRealtimeMonitoringEnabled != nil {
		settings.OpsRealtimeMonitoringEnabled = *req.OpsRealtimeMonitoringEnabled
	}
	if req.OpsQueryModeDefault != nil {
		settings.OpsQueryModeDefault = *req.OpsQueryModeDefault
	}
	if req.OpsMetricsIntervalSeconds != nil {
		settings.OpsMetricsIntervalSeconds = *req.OpsMetricsIntervalSeconds
	}
	// Payment
	if req.PaymentEnabled != nil {
		settings.PaymentEnabled = *req.PaymentEnabled
	}
	if req.PaymentCurrency != nil {
		settings.PaymentCurrency = *req.PaymentCurrency
	}
	if req.PaymentExchangeRate != nil {
		settings.PaymentExchangeRate = *req.PaymentExchangeRate
	}
	if req.PaymentPresetAmounts != nil {
		settings.PaymentPresetAmounts = *req.PaymentPresetAmounts
	}
	if req.PaymentMinAmount != nil {
		settings.PaymentMinAmount = *req.PaymentMinAmount
	}
	if req.PaymentMaxAmount != nil {
		settings.PaymentMaxAmount = *req.PaymentMaxAmount
	}
	if req.AlipayEnabled != nil {
		settings.AlipayEnabled = *req.AlipayEnabled
	}
	if req.AlipayAppID != nil {
		settings.AlipayAppID = *req.AlipayAppID
	}
	if req.AlipayPrivateKey != "" {
		settings.AlipayPrivateKey = req.AlipayPrivateKey
	}
	if req.AlipayPublicKey != "" {
		settings.AlipayPublicKey = req.AlipayPublicKey
	}
	if req.AlipayF2FEnabled != nil {
		settings.AlipayF2FEnabled = *req.AlipayF2FEnabled
	}
	if req.WechatEnabled != nil {
		settings.WechatEnabled = *req.WechatEnabled
	}
	if req.WechatAppID != nil {
		settings.WechatAppID = *req.WechatAppID
	}
	if req.WechatMchID != nil {
		settings.WechatMchID = *req.WechatMchID
	}
	if req.WechatAPIKey != "" {
		settings.WechatAPIKey = req.WechatAPIKey
	}
	if req.EpayEnabled != nil {
		settings.EpayEnabled = *req.EpayEnabled
	}
	if req.EpayAPIURL != nil {
		settings.EpayAPIURL = *req.EpayAPIURL
	}
	if req.EpayPID != nil {
		settings.EpayPID = *req.EpayPID
	}
	if req.EpayKey != "" {
		settings.EpayKey = req.EpayKey
	}
	if req.EpayType != nil {
		settings.EpayType = *req.EpayType
	}
	if req.ReferralEnabled != nil {
		settings.ReferralEnabled = *req.ReferralEnabled
	}
	if req.ReferralCommissionRate != nil {
		settings.ReferralCommissionRate = *req.ReferralCommissionRate
	}

	if err := h.settingService.UpdateSettings(c.Request.Context(), settings); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.auditSettingsUpdate(c, previousSettings, settings, req)

	// 重新获取设置返回
	updatedSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.SystemSettings{
		RegistrationEnabled:                  updatedSettings.RegistrationEnabled,
		EmailVerifyEnabled:                   updatedSettings.EmailVerifyEnabled,
		PromoCodeEnabled:                     updatedSettings.PromoCodeEnabled,
		PasswordResetEnabled:                 updatedSettings.PasswordResetEnabled,
		InvitationCodeEnabled:                updatedSettings.InvitationCodeEnabled,
		TotpEnabled:                          updatedSettings.TotpEnabled,
		TotpEncryptionKeyConfigured:          h.settingService.IsTotpEncryptionKeyConfigured(),
		SMTPHost:                             updatedSettings.SMTPHost,
		SMTPPort:                             updatedSettings.SMTPPort,
		SMTPUsername:                         updatedSettings.SMTPUsername,
		SMTPPasswordConfigured:               updatedSettings.SMTPPasswordConfigured,
		SMTPFrom:                             updatedSettings.SMTPFrom,
		SMTPFromName:                         updatedSettings.SMTPFromName,
		SMTPUseTLS:                           updatedSettings.SMTPUseTLS,
		TurnstileEnabled:                     updatedSettings.TurnstileEnabled,
		TurnstileSiteKey:                     updatedSettings.TurnstileSiteKey,
		TurnstileSecretKeyConfigured:         updatedSettings.TurnstileSecretKeyConfigured,
		LinuxDoConnectEnabled:                updatedSettings.LinuxDoConnectEnabled,
		LinuxDoConnectClientID:               updatedSettings.LinuxDoConnectClientID,
		LinuxDoConnectClientSecretConfigured: updatedSettings.LinuxDoConnectClientSecretConfigured,
		LinuxDoConnectRedirectURL:            updatedSettings.LinuxDoConnectRedirectURL,
		SiteName:                             updatedSettings.SiteName,
		SiteLogo:                             updatedSettings.SiteLogo,
		SiteSubtitle:                         updatedSettings.SiteSubtitle,
		APIBaseURL:                           updatedSettings.APIBaseURL,
		ContactInfo:                          updatedSettings.ContactInfo,
		DocURL:                               updatedSettings.DocURL,
		HomeContent:                          updatedSettings.HomeContent,
		HideCcsImportButton:                  updatedSettings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:          updatedSettings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:              updatedSettings.PurchaseSubscriptionURL,
		SoraClientEnabled:                    updatedSettings.SoraClientEnabled,
		DefaultConcurrency:                   updatedSettings.DefaultConcurrency,
		DefaultBalance:                       updatedSettings.DefaultBalance,
		EnableModelFallback:                  updatedSettings.EnableModelFallback,
		FallbackModelAnthropic:               updatedSettings.FallbackModelAnthropic,
		FallbackModelOpenAI:                  updatedSettings.FallbackModelOpenAI,
		FallbackModelGemini:                  updatedSettings.FallbackModelGemini,
		FallbackModelAntigravity:             updatedSettings.FallbackModelAntigravity,
		EnableIdentityPatch:                  updatedSettings.EnableIdentityPatch,
		IdentityPatchPrompt:                  updatedSettings.IdentityPatchPrompt,
		OpsMonitoringEnabled:                 updatedSettings.OpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:         updatedSettings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                  updatedSettings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:            updatedSettings.OpsMetricsIntervalSeconds,
		PaymentEnabled:                       updatedSettings.PaymentEnabled,
		PaymentCurrency:                      updatedSettings.PaymentCurrency,
		PaymentExchangeRate:                  updatedSettings.PaymentExchangeRate,
		PaymentPresetAmounts:                 updatedSettings.PaymentPresetAmounts,
		PaymentMinAmount:                     updatedSettings.PaymentMinAmount,
		PaymentMaxAmount:                     updatedSettings.PaymentMaxAmount,
		AlipayEnabled:                        updatedSettings.AlipayEnabled,
		AlipayAppID:                          updatedSettings.AlipayAppID,
		AlipayPrivateKeyConfigured:           updatedSettings.AlipayPrivateKeyConfigured,
		AlipayPublicKeyConfigured:            updatedSettings.AlipayPublicKeyConfigured,
		AlipayF2FEnabled:                     updatedSettings.AlipayF2FEnabled,
		WechatEnabled:                        updatedSettings.WechatEnabled,
		WechatAppID:                          updatedSettings.WechatAppID,
		WechatMchID:                          updatedSettings.WechatMchID,
		WechatAPIKeyConfigured:               updatedSettings.WechatAPIKeyConfigured,
		EpayEnabled:                          updatedSettings.EpayEnabled,
		EpayAPIURL:                           updatedSettings.EpayAPIURL,
		EpayPID:                              updatedSettings.EpayPID,
		EpayKeyConfigured:                    updatedSettings.EpayKeyConfigured,
		EpayType:                             updatedSettings.EpayType,
		ReferralEnabled:                      updatedSettings.ReferralEnabled,
		ReferralCommissionRate:               updatedSettings.ReferralCommissionRate,
	})
}

func (h *SettingHandler) auditSettingsUpdate(c *gin.Context, before *service.SystemSettings, after *service.SystemSettings, req UpdateSettingsRequest) {
	if before == nil || after == nil {
		return
	}

	changed := diffSettings(before, after, req)
	if len(changed) == 0 {
		return
	}

	subject, _ := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	log.Printf("AUDIT: settings updated at=%s user_id=%d role=%s changed=%v",
		time.Now().UTC().Format(time.RFC3339),
		subject.UserID,
		role,
		changed,
	)
}

func diffSettings(before *service.SystemSettings, after *service.SystemSettings, req UpdateSettingsRequest) []string {
	changed := make([]string, 0, 20)
	if before.RegistrationEnabled != after.RegistrationEnabled {
		changed = append(changed, "registration_enabled")
	}
	if before.EmailVerifyEnabled != after.EmailVerifyEnabled {
		changed = append(changed, "email_verify_enabled")
	}
	if before.PasswordResetEnabled != after.PasswordResetEnabled {
		changed = append(changed, "password_reset_enabled")
	}
	if before.TotpEnabled != after.TotpEnabled {
		changed = append(changed, "totp_enabled")
	}
	if before.SMTPHost != after.SMTPHost {
		changed = append(changed, "smtp_host")
	}
	if before.SMTPPort != after.SMTPPort {
		changed = append(changed, "smtp_port")
	}
	if before.SMTPUsername != after.SMTPUsername {
		changed = append(changed, "smtp_username")
	}
	if req.SMTPPassword != "" {
		changed = append(changed, "smtp_password")
	}
	if before.SMTPFrom != after.SMTPFrom {
		changed = append(changed, "smtp_from_email")
	}
	if before.SMTPFromName != after.SMTPFromName {
		changed = append(changed, "smtp_from_name")
	}
	if before.SMTPUseTLS != after.SMTPUseTLS {
		changed = append(changed, "smtp_use_tls")
	}
	if before.TurnstileEnabled != after.TurnstileEnabled {
		changed = append(changed, "turnstile_enabled")
	}
	if before.TurnstileSiteKey != after.TurnstileSiteKey {
		changed = append(changed, "turnstile_site_key")
	}
	if req.TurnstileSecretKey != "" {
		changed = append(changed, "turnstile_secret_key")
	}
	if before.LinuxDoConnectEnabled != after.LinuxDoConnectEnabled {
		changed = append(changed, "linuxdo_connect_enabled")
	}
	if before.LinuxDoConnectClientID != after.LinuxDoConnectClientID {
		changed = append(changed, "linuxdo_connect_client_id")
	}
	if req.LinuxDoConnectClientSecret != "" {
		changed = append(changed, "linuxdo_connect_client_secret")
	}
	if before.LinuxDoConnectRedirectURL != after.LinuxDoConnectRedirectURL {
		changed = append(changed, "linuxdo_connect_redirect_url")
	}
	if before.SiteName != after.SiteName {
		changed = append(changed, "site_name")
	}
	if before.SiteLogo != after.SiteLogo {
		changed = append(changed, "site_logo")
	}
	if before.SiteSubtitle != after.SiteSubtitle {
		changed = append(changed, "site_subtitle")
	}
	if before.APIBaseURL != after.APIBaseURL {
		changed = append(changed, "api_base_url")
	}
	if before.ContactInfo != after.ContactInfo {
		changed = append(changed, "contact_info")
	}
	if before.DocURL != after.DocURL {
		changed = append(changed, "doc_url")
	}
	if before.HomeContent != after.HomeContent {
		changed = append(changed, "home_content")
	}
	if before.HideCcsImportButton != after.HideCcsImportButton {
		changed = append(changed, "hide_ccs_import_button")
	}
	if before.DefaultConcurrency != after.DefaultConcurrency {
		changed = append(changed, "default_concurrency")
	}
	if before.DefaultBalance != after.DefaultBalance {
		changed = append(changed, "default_balance")
	}
	if before.EnableModelFallback != after.EnableModelFallback {
		changed = append(changed, "enable_model_fallback")
	}
	if before.FallbackModelAnthropic != after.FallbackModelAnthropic {
		changed = append(changed, "fallback_model_anthropic")
	}
	if before.FallbackModelOpenAI != after.FallbackModelOpenAI {
		changed = append(changed, "fallback_model_openai")
	}
	if before.FallbackModelGemini != after.FallbackModelGemini {
		changed = append(changed, "fallback_model_gemini")
	}
	if before.FallbackModelAntigravity != after.FallbackModelAntigravity {
		changed = append(changed, "fallback_model_antigravity")
	}
	if before.EnableIdentityPatch != after.EnableIdentityPatch {
		changed = append(changed, "enable_identity_patch")
	}
	if before.IdentityPatchPrompt != after.IdentityPatchPrompt {
		changed = append(changed, "identity_patch_prompt")
	}
	if before.OpsMonitoringEnabled != after.OpsMonitoringEnabled {
		changed = append(changed, "ops_monitoring_enabled")
	}
	if before.OpsRealtimeMonitoringEnabled != after.OpsRealtimeMonitoringEnabled {
		changed = append(changed, "ops_realtime_monitoring_enabled")
	}
	if before.OpsQueryModeDefault != after.OpsQueryModeDefault {
		changed = append(changed, "ops_query_mode_default")
	}
	if before.OpsMetricsIntervalSeconds != after.OpsMetricsIntervalSeconds {
		changed = append(changed, "ops_metrics_interval_seconds")
	}
	// Payment fields
	if before.PaymentEnabled != after.PaymentEnabled {
		changed = append(changed, "payment_enabled")
	}
	if before.PaymentCurrency != after.PaymentCurrency {
		changed = append(changed, "payment_currency")
	}
	if before.PaymentExchangeRate != after.PaymentExchangeRate {
		changed = append(changed, "payment_exchange_rate")
	}
	if before.PaymentPresetAmounts != after.PaymentPresetAmounts {
		changed = append(changed, "payment_preset_amounts")
	}
	if before.PaymentMinAmount != after.PaymentMinAmount {
		changed = append(changed, "payment_min_amount")
	}
	if before.PaymentMaxAmount != after.PaymentMaxAmount {
		changed = append(changed, "payment_max_amount")
	}
	if before.AlipayEnabled != after.AlipayEnabled {
		changed = append(changed, "payment_alipay_enabled")
	}
	if before.AlipayAppID != after.AlipayAppID {
		changed = append(changed, "payment_alipay_app_id")
	}
	if req.AlipayPrivateKey != "" {
		changed = append(changed, "payment_alipay_private_key")
	}
	if req.AlipayPublicKey != "" {
		changed = append(changed, "payment_alipay_public_key")
	}
	if before.AlipayF2FEnabled != after.AlipayF2FEnabled {
		changed = append(changed, "payment_alipay_f2f_enabled")
	}
	if before.WechatEnabled != after.WechatEnabled {
		changed = append(changed, "payment_wechat_enabled")
	}
	if before.WechatAppID != after.WechatAppID {
		changed = append(changed, "payment_wechat_app_id")
	}
	if before.WechatMchID != after.WechatMchID {
		changed = append(changed, "payment_wechat_mch_id")
	}
	if req.WechatAPIKey != "" {
		changed = append(changed, "payment_wechat_api_key")
	}
	if before.EpayEnabled != after.EpayEnabled {
		changed = append(changed, "payment_epay_enabled")
	}
	if before.EpayAPIURL != after.EpayAPIURL {
		changed = append(changed, "payment_epay_api_url")
	}
	if before.EpayPID != after.EpayPID {
		changed = append(changed, "payment_epay_pid")
	}
	if req.EpayKey != "" {
		changed = append(changed, "payment_epay_key")
	}
	if before.EpayType != after.EpayType {
		changed = append(changed, "payment_epay_type")
	}
	if before.ReferralEnabled != after.ReferralEnabled {
		changed = append(changed, "referral_enabled")
	}
	if before.ReferralCommissionRate != after.ReferralCommissionRate {
		changed = append(changed, "referral_commission_rate")
	}
	return changed
}

// TestSMTPRequest 测试SMTP连接请求
type TestSMTPRequest struct {
	SMTPHost     string `json:"smtp_host" binding:"required"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
}

// TestSMTPConnection 测试SMTP连接
// POST /api/v1/admin/settings/test-smtp
func (h *SettingHandler) TestSMTPConnection(c *gin.Context) {
	var req TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}

	// 如果未提供密码，从数据库获取已保存的密码
	password := req.SMTPPassword
	if password == "" {
		savedConfig, err := h.emailService.GetSMTPConfig(c.Request.Context())
		if err == nil && savedConfig != nil {
			password = savedConfig.Password
		}
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		UseTLS:   req.SMTPUseTLS,
	}

	err := h.emailService.TestSMTPConnectionWithConfig(config)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "SMTP connection successful"})
}

// SendTestEmailRequest 发送测试邮件请求
type SendTestEmailRequest struct {
	Email        string `json:"email" binding:"required,email"`
	SMTPHost     string `json:"smtp_host" binding:"required"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from_email"`
	SMTPFromName string `json:"smtp_from_name"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`
}

// SendTestEmail 发送测试邮件
// POST /api/v1/admin/settings/send-test-email
func (h *SettingHandler) SendTestEmail(c *gin.Context) {
	var req SendTestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}

	// 如果未提供密码，从数据库获取已保存的密码
	password := req.SMTPPassword
	if password == "" {
		savedConfig, err := h.emailService.GetSMTPConfig(c.Request.Context())
		if err == nil && savedConfig != nil {
			password = savedConfig.Password
		}
	}

	config := &service.SMTPConfig{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
		From:     req.SMTPFrom,
		FromName: req.SMTPFromName,
		UseTLS:   req.SMTPUseTLS,
	}

	siteName := h.settingService.GetSiteName(c.Request.Context())
	subject := "[" + siteName + "] Test Email"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
        .content { padding: 40px 30px; text-align: center; }
        .success { color: #10b981; font-size: 48px; margin-bottom: 20px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + siteName + `</h1>
        </div>
        <div class="content">
            <div class="success">✓</div>
            <h2>Email Configuration Successful!</h2>
            <p>This is a test email to verify your SMTP settings are working correctly.</p>
        </div>
        <div class="footer">
            <p>This is an automated test message.</p>
        </div>
    </div>
</body>
</html>
`

	if err := h.emailService.SendEmailWithConfig(config, req.Email, subject, body); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Test email sent successfully"})
}

// GetAdminAPIKey 获取管理员 API Key 状态
// GET /api/v1/admin/settings/admin-api-key
func (h *SettingHandler) GetAdminAPIKey(c *gin.Context) {
	maskedKey, exists, err := h.settingService.GetAdminAPIKeyStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"exists":     exists,
		"masked_key": maskedKey,
	})
}

// RegenerateAdminAPIKey 生成/重新生成管理员 API Key
// POST /api/v1/admin/settings/admin-api-key/regenerate
func (h *SettingHandler) RegenerateAdminAPIKey(c *gin.Context) {
	key, err := h.settingService.GenerateAdminAPIKey(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"key": key, // 完整 key 只在生成时返回一次
	})
}

// DeleteAdminAPIKey 删除管理员 API Key
// DELETE /api/v1/admin/settings/admin-api-key
func (h *SettingHandler) DeleteAdminAPIKey(c *gin.Context) {
	if err := h.settingService.DeleteAdminAPIKey(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Admin API key deleted"})
}

// GetStreamTimeoutSettings 获取流超时处理配置
// GET /api/v1/admin/settings/stream-timeout
func (h *SettingHandler) GetStreamTimeoutSettings(c *gin.Context) {
	settings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.StreamTimeoutSettings{
		Enabled:                settings.Enabled,
		Action:                 settings.Action,
		TempUnschedMinutes:     settings.TempUnschedMinutes,
		ThresholdCount:         settings.ThresholdCount,
		ThresholdWindowMinutes: settings.ThresholdWindowMinutes,
	})
}

func toSoraS3SettingsDTO(settings *service.SoraS3Settings) dto.SoraS3Settings {
	if settings == nil {
		return dto.SoraS3Settings{}
	}
	return dto.SoraS3Settings{
		Enabled:                   settings.Enabled,
		Endpoint:                  settings.Endpoint,
		Region:                    settings.Region,
		Bucket:                    settings.Bucket,
		AccessKeyID:               settings.AccessKeyID,
		SecretAccessKeyConfigured: settings.SecretAccessKeyConfigured,
		Prefix:                    settings.Prefix,
		ForcePathStyle:            settings.ForcePathStyle,
		CDNURL:                    settings.CDNURL,
		DefaultStorageQuotaBytes:  settings.DefaultStorageQuotaBytes,
	}
}

func toSoraS3ProfileDTO(profile service.SoraS3Profile) dto.SoraS3Profile {
	return dto.SoraS3Profile{
		ProfileID:                 profile.ProfileID,
		Name:                      profile.Name,
		IsActive:                  profile.IsActive,
		Enabled:                   profile.Enabled,
		Endpoint:                  profile.Endpoint,
		Region:                    profile.Region,
		Bucket:                    profile.Bucket,
		AccessKeyID:               profile.AccessKeyID,
		SecretAccessKeyConfigured: profile.SecretAccessKeyConfigured,
		Prefix:                    profile.Prefix,
		ForcePathStyle:            profile.ForcePathStyle,
		CDNURL:                    profile.CDNURL,
		DefaultStorageQuotaBytes:  profile.DefaultStorageQuotaBytes,
		UpdatedAt:                 profile.UpdatedAt,
	}
}

func validateSoraS3RequiredWhenEnabled(enabled bool, endpoint, bucket, accessKeyID, secretAccessKey string, hasStoredSecret bool) error {
	if !enabled {
		return nil
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("S3 Endpoint is required when enabled")
	}
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("S3 Bucket is required when enabled")
	}
	if strings.TrimSpace(accessKeyID) == "" {
		return fmt.Errorf("S3 Access Key ID is required when enabled")
	}
	if strings.TrimSpace(secretAccessKey) != "" || hasStoredSecret {
		return nil
	}
	return fmt.Errorf("S3 Secret Access Key is required when enabled")
}

func findSoraS3ProfileByID(items []service.SoraS3Profile, profileID string) *service.SoraS3Profile {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return &items[idx]
		}
	}
	return nil
}

// GetSoraS3Settings 获取 Sora S3 存储配置（兼容旧单配置接口）
// GET /api/v1/admin/settings/sora-s3
func (h *SettingHandler) GetSoraS3Settings(c *gin.Context) {
	settings, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3SettingsDTO(settings))
}

// ListSoraS3Profiles 获取 Sora S3 多配置
// GET /api/v1/admin/settings/sora-s3/profiles
func (h *SettingHandler) ListSoraS3Profiles(c *gin.Context) {
	result, err := h.settingService.ListSoraS3Profiles(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]dto.SoraS3Profile, 0, len(result.Items))
	for idx := range result.Items {
		items = append(items, toSoraS3ProfileDTO(result.Items[idx]))
	}
	response.Success(c, dto.ListSoraS3ProfilesResponse{
		ActiveProfileID: result.ActiveProfileID,
		Items:           items,
	})
}

// UpdateSoraS3SettingsRequest 更新/测试 Sora S3 配置请求（兼容旧接口）
type UpdateSoraS3SettingsRequest struct {
	ProfileID                string `json:"profile_id"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}

type CreateSoraS3ProfileRequest struct {
	ProfileID                string `json:"profile_id"`
	Name                     string `json:"name"`
	SetActive                bool   `json:"set_active"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}

type UpdateSoraS3ProfileRequest struct {
	Name                     string `json:"name"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
}

// CreateSoraS3Profile 创建 Sora S3 配置
// POST /api/v1/admin/settings/sora-s3/profiles
func (h *SettingHandler) CreateSoraS3Profile(c *gin.Context) {
	var req CreateSoraS3ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.BadRequest(c, "Name is required")
		return
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, false); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	created, err := h.settingService.CreateSoraS3Profile(c.Request.Context(), &service.SoraS3Profile{
		ProfileID:                req.ProfileID,
		Name:                     req.Name,
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	}, req.SetActive)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, toSoraS3ProfileDTO(*created))
}

// UpdateSoraS3Profile 更新 Sora S3 配置
// PUT /api/v1/admin/settings/sora-s3/profiles/:profile_id
func (h *SettingHandler) UpdateSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}

	var req UpdateSoraS3ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.BadRequest(c, "Name is required")
		return
	}

	existingList, err := h.settingService.ListSoraS3Profiles(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	existing := findSoraS3ProfileByID(existingList.Items, profileID)
	if existing == nil {
		response.ErrorFrom(c, service.ErrSoraS3ProfileNotFound)
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updated, updateErr := h.settingService.UpdateSoraS3Profile(c.Request.Context(), profileID, &service.SoraS3Profile{
		Name:                     req.Name,
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	})
	if updateErr != nil {
		response.ErrorFrom(c, updateErr)
		return
	}

	response.Success(c, toSoraS3ProfileDTO(*updated))
}

// DeleteSoraS3Profile 删除 Sora S3 配置
// DELETE /api/v1/admin/settings/sora-s3/profiles/:profile_id
func (h *SettingHandler) DeleteSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	if err := h.settingService.DeleteSoraS3Profile(c.Request.Context(), profileID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// SetActiveSoraS3Profile 切换激活 Sora S3 配置
// POST /api/v1/admin/settings/sora-s3/profiles/:profile_id/activate
func (h *SettingHandler) SetActiveSoraS3Profile(c *gin.Context) {
	profileID := strings.TrimSpace(c.Param("profile_id"))
	if profileID == "" {
		response.BadRequest(c, "Profile ID is required")
		return
	}
	active, err := h.settingService.SetActiveSoraS3Profile(c.Request.Context(), profileID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3ProfileDTO(*active))
}

// UpdateSoraS3Settings 更新 Sora S3 存储配置（兼容旧单配置接口）
// PUT /api/v1/admin/settings/sora-s3
func (h *SettingHandler) UpdateSoraS3Settings(c *gin.Context) {
	var req UpdateSoraS3SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	settings := &service.SoraS3Settings{
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	}
	if err := h.settingService.SetSoraS3Settings(c.Request.Context(), settings); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updatedSettings, err := h.settingService.GetSoraS3Settings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toSoraS3SettingsDTO(updatedSettings))
}

// TestSoraS3Connection 测试 Sora S3 连接（HeadBucket）
// POST /api/v1/admin/settings/sora-s3/test
func (h *SettingHandler) TestSoraS3Connection(c *gin.Context) {
	if h.soraS3Storage == nil {
		response.Error(c, 500, "S3 存储服务未初始化")
		return
	}

	var req UpdateSoraS3SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !req.Enabled {
		response.BadRequest(c, "S3 未启用，无法测试连接")
		return
	}

	if req.SecretAccessKey == "" {
		if req.ProfileID != "" {
			profiles, err := h.settingService.ListSoraS3Profiles(c.Request.Context())
			if err == nil {
				profile := findSoraS3ProfileByID(profiles.Items, req.ProfileID)
				if profile != nil {
					req.SecretAccessKey = profile.SecretAccessKey
				}
			}
		}
		if req.SecretAccessKey == "" {
			existing, err := h.settingService.GetSoraS3Settings(c.Request.Context())
			if err == nil {
				req.SecretAccessKey = existing.SecretAccessKey
			}
		}
	}

	testCfg := &service.SoraS3Settings{
		Enabled:         true,
		Endpoint:        req.Endpoint,
		Region:          req.Region,
		Bucket:          req.Bucket,
		AccessKeyID:     req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		Prefix:          req.Prefix,
		ForcePathStyle:  req.ForcePathStyle,
		CDNURL:          req.CDNURL,
	}
	if err := h.soraS3Storage.TestConnectionWithSettings(c.Request.Context(), testCfg); err != nil {
		response.Error(c, 400, "S3 连接测试失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "S3 连接成功"})
}

// UpdateStreamTimeoutSettingsRequest 更新流超时配置请求
type UpdateStreamTimeoutSettingsRequest struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}

// UpdateStreamTimeoutSettings 更新流超时处理配置
// PUT /api/v1/admin/settings/stream-timeout
func (h *SettingHandler) UpdateStreamTimeoutSettings(c *gin.Context) {
	var req UpdateStreamTimeoutSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	settings := &service.StreamTimeoutSettings{
		Enabled:                req.Enabled,
		Action:                 req.Action,
		TempUnschedMinutes:     req.TempUnschedMinutes,
		ThresholdCount:         req.ThresholdCount,
		ThresholdWindowMinutes: req.ThresholdWindowMinutes,
	}

	if err := h.settingService.SetStreamTimeoutSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 重新获取设置返回
	updatedSettings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.StreamTimeoutSettings{
		Enabled:                updatedSettings.Enabled,
		Action:                 updatedSettings.Action,
		TempUnschedMinutes:     updatedSettings.TempUnschedMinutes,
		ThresholdCount:         updatedSettings.ThresholdCount,
		ThresholdWindowMinutes: updatedSettings.ThresholdWindowMinutes,
	})
}

package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/kiro"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"golang.org/x/sync/errgroup"
)

type UsageLogRepository interface {
	// Create creates a usage log and returns whether it was actually inserted.
	// inserted is false when the insert was skipped due to conflict (idempotent retries).
	Create(ctx context.Context, log *UsageLog) (inserted bool, err error)
	GetByID(ctx context.Context, id int64) (*UsageLog, error)
	Delete(ctx context.Context, id int64) error

	ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error)
	ListByAPIKey(ctx context.Context, apiKeyID int64, params pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error)
	ListByAccount(ctx context.Context, accountID int64, params pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error)

	ListByUserAndTimeRange(ctx context.Context, userID int64, startTime, endTime time.Time) ([]UsageLog, *pagination.PaginationResult, error)
	ListByAPIKeyAndTimeRange(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) ([]UsageLog, *pagination.PaginationResult, error)
	ListByAccountAndTimeRange(ctx context.Context, accountID int64, startTime, endTime time.Time) ([]UsageLog, *pagination.PaginationResult, error)
	ListByModelAndTimeRange(ctx context.Context, modelName string, startTime, endTime time.Time) ([]UsageLog, *pagination.PaginationResult, error)

	GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error)
	GetAccountTodayStats(ctx context.Context, accountID int64) (*usagestats.AccountStats, error)

	// Admin dashboard stats
	GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error)
	GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.TrendDataPoint, error)
	GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.ModelStat, error)
	GetAPIKeyUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.APIKeyUsageTrendPoint, error)
	GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error)
	GetBatchUserUsageStats(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchUserUsageStats, error)
	GetBatchAPIKeyUsageStats(ctx context.Context, apiKeyIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchAPIKeyUsageStats, error)

	// User dashboard stats
	GetUserDashboardStats(ctx context.Context, userID int64) (*usagestats.UserDashboardStats, error)
	GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*usagestats.UserDashboardStats, error)
	GetUserUsageTrendByUserID(ctx context.Context, userID int64, startTime, endTime time.Time, granularity string) ([]usagestats.TrendDataPoint, error)
	GetUserModelStats(ctx context.Context, userID int64, startTime, endTime time.Time) ([]usagestats.ModelStat, error)

	// Admin usage listing/stats
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error)
	GetGlobalStats(ctx context.Context, startTime, endTime time.Time) (*usagestats.UsageStats, error)
	GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error)

	// Account stats
	GetAccountUsageStats(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.AccountUsageStatsResponse, error)

	// Aggregated stats (optimized)
	GetUserStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error)
	GetAPIKeyStatsAggregated(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error)
	GetAccountStatsAggregated(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error)
	GetModelStatsAggregated(ctx context.Context, modelName string, startTime, endTime time.Time) (*usagestats.UsageStats, error)
	GetDailyStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) ([]map[string]any, error)
}

type accountWindowStatsBatchReader interface {
	GetAccountWindowStatsBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]*usagestats.AccountStats, error)
}

// apiUsageCache 缓存从 Anthropic API 获取的使用率数据（utilization, resets_at）
type apiUsageCache struct {
	response  *ClaudeUsageResponse
	timestamp time.Time
}

// windowStatsCache 缓存从本地数据库查询的窗口统计（requests, tokens, cost）
type windowStatsCache struct {
	stats     *WindowStats
	timestamp time.Time
}

// antigravityUsageCache 缓存 Antigravity 额度数据
type antigravityUsageCache struct {
	usageInfo *UsageInfo
	timestamp time.Time
}

// kiroSubscriptionCache 缓存 Kiro 订阅额度数据
type kiroSubscriptionCache struct {
	quota     *KiroSubscriptionQuota
	timestamp time.Time
}

const (
	apiCacheTTL              = 3 * time.Minute
	windowStatsCacheTTL      = 1 * time.Minute
	kiroSubscriptionCacheTTL = 5 * time.Minute
)

// UsageCache 封装账户使用量相关的缓存
type UsageCache struct {
	apiCache              sync.Map // accountID -> *apiUsageCache
	windowStatsCache      sync.Map // accountID -> *windowStatsCache
	antigravityCache      sync.Map // accountID -> *antigravityUsageCache
	kiroSubscriptionCache sync.Map // accountID -> *kiroSubscriptionCache
}

// NewUsageCache 创建 UsageCache 实例
func NewUsageCache() *UsageCache {
	return &UsageCache{}
}

// WindowStats 窗口期统计
//
// cost: 账号口径费用（total_cost * account_rate_multiplier）
// standard_cost: 标准费用（total_cost，不含倍率）
// user_cost: 用户/API Key 口径费用（actual_cost，受分组倍率影响）
type WindowStats struct {
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	Cost         float64 `json:"cost"`
	StandardCost float64 `json:"standard_cost"`
	UserCost     float64 `json:"user_cost"`
}

// UsageProgress 使用量进度
type UsageProgress struct {
	Utilization      float64      `json:"utilization"`            // 使用率百分比 (0-100+，100表示100%)
	ResetsAt         *time.Time   `json:"resets_at"`              // 重置时间
	RemainingSeconds int          `json:"remaining_seconds"`      // 距重置剩余秒数
	WindowStats      *WindowStats `json:"window_stats,omitempty"` // 窗口期统计（从窗口开始到当前的使用量）
	UsedRequests     int64        `json:"used_requests,omitempty"`
	LimitRequests    int64        `json:"limit_requests,omitempty"`
}

// AntigravityModelQuota Antigravity 单个模型的配额信息
type AntigravityModelQuota struct {
	Utilization int    `json:"utilization"` // 使用率 0-100
	ResetTime   string `json:"reset_time"`  // 重置时间 ISO8601
}

// KiroModelUsage Kiro 单个模型的日使用量
type KiroModelUsage struct {
	Requests int64   `json:"requests"` // 日请求数
	Tokens   int64   `json:"tokens"`   // 日 Token 数
	Cost     float64 `json:"cost"`     // 日费用
}

// KiroSubscriptionQuota Kiro 订阅额度信息（从 CodeWhisperer API 获取）
type KiroSubscriptionQuota struct {
	SubscriptionType string  `json:"subscription_type"`          // FREE/PRO/PRO_PLUS/POWER
	CurrentUsage     float64 `json:"current_usage"`              // 当前使用量
	UsageLimit       float64 `json:"usage_limit"`                // 总额度
	UsagePercent     float64 `json:"usage_percent"`              // 使用百分比 (0-100)
	NextResetDate    string  `json:"next_reset_date"`            // 重置日期 YYYY-MM-DD
	TrialStatus      string  `json:"trial_status,omitempty"`     // 试用状态
	TrialCurrent     float64 `json:"trial_current,omitempty"`    // 试用已用
	TrialLimit       float64 `json:"trial_limit,omitempty"`      // 试用总量
	Email            string  `json:"email,omitempty"`            // 邮箱
}

// UsageInfo 账号使用量信息
type UsageInfo struct {
	UpdatedAt          *time.Time     `json:"updated_at,omitempty"`           // 更新时间
	FiveHour           *UsageProgress `json:"five_hour"`                      // 5小时窗口
	SevenDay           *UsageProgress `json:"seven_day,omitempty"`            // 7天窗口
	SevenDaySonnet     *UsageProgress `json:"seven_day_sonnet,omitempty"`     // 7天Sonnet窗口
	GeminiSharedDaily  *UsageProgress `json:"gemini_shared_daily,omitempty"`  // Gemini shared pool RPD (Google One / Code Assist)
	GeminiProDaily     *UsageProgress `json:"gemini_pro_daily,omitempty"`     // Gemini Pro 日配额
	GeminiFlashDaily   *UsageProgress `json:"gemini_flash_daily,omitempty"`   // Gemini Flash 日配额
	GeminiSharedMinute *UsageProgress `json:"gemini_shared_minute,omitempty"` // Gemini shared pool RPM (Google One / Code Assist)
	GeminiProMinute    *UsageProgress `json:"gemini_pro_minute,omitempty"`    // Gemini Pro RPM
	GeminiFlashMinute  *UsageProgress `json:"gemini_flash_minute,omitempty"`  // Gemini Flash RPM

	// Antigravity 多模型配额
	AntigravityQuota map[string]*AntigravityModelQuota `json:"antigravity_quota,omitempty"`

	// Kiro 多模型日使用量
	KiroQuota map[string]*KiroModelUsage `json:"kiro_quota,omitempty"`

	// Kiro 订阅额度（从 CodeWhisperer API 实时获取）
	KiroSubscription *KiroSubscriptionQuota `json:"kiro_subscription,omitempty"`
}

// ClaudeUsageResponse Anthropic API返回的usage结构
type ClaudeUsageResponse struct {
	FiveHour struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"seven_day"`
	SevenDaySonnet struct {
		Utilization float64 `json:"utilization"`
		ResetsAt    string  `json:"resets_at"`
	} `json:"seven_day_sonnet"`
}

// ClaudeUsageFetchOptions 包含获取 Claude 用量数据所需的所有选项
type ClaudeUsageFetchOptions struct {
	AccessToken          string       // OAuth access token
	ProxyURL             string       // 代理 URL（可选）
	AccountID            int64        // 账号 ID（用于 TLS 指纹选择）
	EnableTLSFingerprint bool         // 是否启用 TLS 指纹伪装
	Fingerprint          *Fingerprint // 缓存的指纹信息（User-Agent 等）
}

// ClaudeUsageFetcher fetches usage data from Anthropic OAuth API
type ClaudeUsageFetcher interface {
	FetchUsage(ctx context.Context, accessToken, proxyURL string) (*ClaudeUsageResponse, error)
	// FetchUsageWithOptions 使用完整选项获取用量数据，支持 TLS 指纹和自定义 User-Agent
	FetchUsageWithOptions(ctx context.Context, opts *ClaudeUsageFetchOptions) (*ClaudeUsageResponse, error)
}

// AccountUsageService 账号使用量查询服务
type AccountUsageService struct {
	accountRepo             AccountRepository
	usageLogRepo            UsageLogRepository
	usageFetcher            ClaudeUsageFetcher
	geminiQuotaService      *GeminiQuotaService
	antigravityQuotaFetcher *AntigravityQuotaFetcher
	kiroOAuthService        *KiroOAuthService
	cache                   *UsageCache
	identityCache           IdentityCache
}

// NewAccountUsageService 创建AccountUsageService实例
func NewAccountUsageService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageFetcher ClaudeUsageFetcher,
	geminiQuotaService *GeminiQuotaService,
	antigravityQuotaFetcher *AntigravityQuotaFetcher,
	kiroOAuthService *KiroOAuthService,
	cache *UsageCache,
	identityCache IdentityCache,
) *AccountUsageService {
	return &AccountUsageService{
		accountRepo:             accountRepo,
		usageLogRepo:            usageLogRepo,
		usageFetcher:            usageFetcher,
		geminiQuotaService:      geminiQuotaService,
		antigravityQuotaFetcher: antigravityQuotaFetcher,
		kiroOAuthService:        kiroOAuthService,
		cache:                   cache,
		identityCache:           identityCache,
	}
}

// GetUsage 获取账号使用量
// OAuth账号: 调用Anthropic API获取真实数据（需要profile scope），API响应缓存10分钟，窗口统计缓存1分钟
// Setup Token账号: 根据session_window推算5h窗口，7d数据不可用（没有profile scope）
// API Key账号: 不支持usage查询
func (s *AccountUsageService) GetUsage(ctx context.Context, accountID int64) (*UsageInfo, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get account failed: %w", err)
	}

	if account.Platform == PlatformGemini {
		usage, err := s.getGeminiUsage(ctx, account)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	// Antigravity 平台：使用 AntigravityQuotaFetcher 获取额度
	if account.Platform == PlatformAntigravity {
		usage, err := s.getAntigravityUsage(ctx, account)
		if err == nil {
			s.tryClearRecoverableAccountError(ctx, account)
		}
		return usage, err
	}

	// Kiro 平台：从本地日志统计日使用量
	if account.Platform == PlatformKiro {
		return s.getKiroUsage(ctx, account)
	}

	// 只有oauth类型账号可以通过API获取usage（有profile scope）
	if account.CanGetUsage() {
		var apiResp *ClaudeUsageResponse

		// 1. 检查 API 缓存（10 分钟）
		if cached, ok := s.cache.apiCache.Load(accountID); ok {
			if cache, ok := cached.(*apiUsageCache); ok && time.Since(cache.timestamp) < apiCacheTTL {
				apiResp = cache.response
			}
		}

		// 2. 如果没有缓存，从 API 获取
		if apiResp == nil {
			apiResp, err = s.fetchOAuthUsageRaw(ctx, account)
			if err != nil {
				return nil, err
			}
			// 缓存 API 响应
			s.cache.apiCache.Store(accountID, &apiUsageCache{
				response:  apiResp,
				timestamp: time.Now(),
			})
		}

		// 3. 构建 UsageInfo（每次都重新计算 RemainingSeconds）
		now := time.Now()
		usage := s.buildUsageInfo(apiResp, &now)

		// 4. 添加窗口统计（有独立缓存，1 分钟）
		s.addWindowStats(ctx, account, usage)

		s.tryClearRecoverableAccountError(ctx, account)
		return usage, nil
	}

	// Setup Token账号：根据session_window推算（没有profile scope，无法调用usage API）
	if account.Type == AccountTypeSetupToken {
		usage := s.estimateSetupTokenUsage(account)
		// 添加窗口统计
		s.addWindowStats(ctx, account, usage)
		return usage, nil
	}

	// API Key账号不支持usage查询
	return nil, fmt.Errorf("account type %s does not support usage query", account.Type)
}

func (s *AccountUsageService) getGeminiUsage(ctx context.Context, account *Account) (*UsageInfo, error) {
	now := time.Now()
	usage := &UsageInfo{
		UpdatedAt: &now,
	}

	if s.geminiQuotaService == nil || s.usageLogRepo == nil {
		return usage, nil
	}

	quota, ok := s.geminiQuotaService.QuotaForAccount(ctx, account)
	if !ok {
		return usage, nil
	}

	dayStart := geminiDailyWindowStart(now)
	stats, err := s.usageLogRepo.GetModelStatsWithFilters(ctx, dayStart, now, 0, 0, account.ID, 0, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get gemini usage stats failed: %w", err)
	}

	dayTotals := geminiAggregateUsage(stats)
	dailyResetAt := geminiDailyResetTime(now)

	// Daily window (RPD)
	if quota.SharedRPD > 0 {
		totalReq := dayTotals.ProRequests + dayTotals.FlashRequests
		totalTokens := dayTotals.ProTokens + dayTotals.FlashTokens
		totalCost := dayTotals.ProCost + dayTotals.FlashCost
		usage.GeminiSharedDaily = buildGeminiUsageProgress(totalReq, quota.SharedRPD, dailyResetAt, totalTokens, totalCost, now)
	} else {
		usage.GeminiProDaily = buildGeminiUsageProgress(dayTotals.ProRequests, quota.ProRPD, dailyResetAt, dayTotals.ProTokens, dayTotals.ProCost, now)
		usage.GeminiFlashDaily = buildGeminiUsageProgress(dayTotals.FlashRequests, quota.FlashRPD, dailyResetAt, dayTotals.FlashTokens, dayTotals.FlashCost, now)
	}

	// Minute window (RPM) - fixed-window approximation: current minute [truncate(now), truncate(now)+1m)
	minuteStart := now.Truncate(time.Minute)
	minuteResetAt := minuteStart.Add(time.Minute)
	minuteStats, err := s.usageLogRepo.GetModelStatsWithFilters(ctx, minuteStart, now, 0, 0, account.ID, 0, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get gemini minute usage stats failed: %w", err)
	}
	minuteTotals := geminiAggregateUsage(minuteStats)

	if quota.SharedRPM > 0 {
		totalReq := minuteTotals.ProRequests + minuteTotals.FlashRequests
		totalTokens := minuteTotals.ProTokens + minuteTotals.FlashTokens
		totalCost := minuteTotals.ProCost + minuteTotals.FlashCost
		usage.GeminiSharedMinute = buildGeminiUsageProgress(totalReq, quota.SharedRPM, minuteResetAt, totalTokens, totalCost, now)
	} else {
		usage.GeminiProMinute = buildGeminiUsageProgress(minuteTotals.ProRequests, quota.ProRPM, minuteResetAt, minuteTotals.ProTokens, minuteTotals.ProCost, now)
		usage.GeminiFlashMinute = buildGeminiUsageProgress(minuteTotals.FlashRequests, quota.FlashRPM, minuteResetAt, minuteTotals.FlashTokens, minuteTotals.FlashCost, now)
	}

	return usage, nil
}

// getAntigravityUsage 获取 Antigravity 账户额度
func (s *AccountUsageService) getAntigravityUsage(ctx context.Context, account *Account) (*UsageInfo, error) {
	if s.antigravityQuotaFetcher == nil || !s.antigravityQuotaFetcher.CanFetch(account) {
		now := time.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}

	// 1. 检查缓存（10 分钟）
	if cached, ok := s.cache.antigravityCache.Load(account.ID); ok {
		if cache, ok := cached.(*antigravityUsageCache); ok && time.Since(cache.timestamp) < apiCacheTTL {
			// 重新计算 RemainingSeconds
			usage := cache.usageInfo
			if usage.FiveHour != nil && usage.FiveHour.ResetsAt != nil {
				usage.FiveHour.RemainingSeconds = int(time.Until(*usage.FiveHour.ResetsAt).Seconds())
			}
			return usage, nil
		}
	}

	// 2. 获取代理 URL
	proxyURL := s.antigravityQuotaFetcher.GetProxyURL(ctx, account)

	// 3. 调用 API 获取额度
	result, err := s.antigravityQuotaFetcher.FetchQuota(ctx, account, proxyURL)
	if err != nil {
		return nil, fmt.Errorf("fetch antigravity quota failed: %w", err)
	}

	// 4. 缓存结果
	s.cache.antigravityCache.Store(account.ID, &antigravityUsageCache{
		usageInfo: result.UsageInfo,
		timestamp: time.Now(),
	})

	return result.UsageInfo, nil
}

// addWindowStats 为 usage 数据添加窗口期统计
// 使用独立缓存（1 分钟），与 API 缓存分离
func (s *AccountUsageService) addWindowStats(ctx context.Context, account *Account, usage *UsageInfo) {
	// 修复：即使 FiveHour 为 nil，也要尝试获取统计数据
	// 因为 SevenDay/SevenDaySonnet 可能需要
	if usage.FiveHour == nil && usage.SevenDay == nil && usage.SevenDaySonnet == nil {
		return
	}

	// 检查窗口统计缓存（1 分钟）
	var windowStats *WindowStats
	if cached, ok := s.cache.windowStatsCache.Load(account.ID); ok {
		if cache, ok := cached.(*windowStatsCache); ok && time.Since(cache.timestamp) < windowStatsCacheTTL {
			windowStats = cache.stats
		}
	}

	// 如果没有缓存，从数据库查询
	if windowStats == nil {
		// 使用统一的窗口开始时间计算逻辑（考虑窗口过期情况）
		startTime := account.GetCurrentWindowStartTime()

		stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, account.ID, startTime)
		if err != nil {
			log.Printf("Failed to get window stats for account %d: %v", account.ID, err)
			return
		}

		windowStats = &WindowStats{
			Requests:     stats.Requests,
			Tokens:       stats.Tokens,
			Cost:         stats.Cost,
			StandardCost: stats.StandardCost,
			UserCost:     stats.UserCost,
		}

		// 缓存窗口统计（1 分钟）
		s.cache.windowStatsCache.Store(account.ID, &windowStatsCache{
			stats:     windowStats,
			timestamp: time.Now(),
		})
	}

	// 为 FiveHour 添加 WindowStats（5h 窗口统计）
	if usage.FiveHour != nil {
		usage.FiveHour.WindowStats = windowStats
	}
}

// GetTodayStats 获取账号今日统计
func (s *AccountUsageService) GetTodayStats(ctx context.Context, accountID int64) (*WindowStats, error) {
	stats, err := s.usageLogRepo.GetAccountTodayStats(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get today stats failed: %w", err)
	}

	return &WindowStats{
		Requests:     stats.Requests,
		Tokens:       stats.Tokens,
		Cost:         stats.Cost,
		StandardCost: stats.StandardCost,
		UserCost:     stats.UserCost,
	}, nil
}

// GetTodayStatsBatch 批量获取账号今日统计，优先走批量 SQL，失败时回退单账号查询。
func (s *AccountUsageService) GetTodayStatsBatch(ctx context.Context, accountIDs []int64) (map[int64]*WindowStats, error) {
	uniqueIDs := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		uniqueIDs = append(uniqueIDs, accountID)
	}

	result := make(map[int64]*WindowStats, len(uniqueIDs))
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	startTime := timezone.Today()
	if batchReader, ok := s.usageLogRepo.(accountWindowStatsBatchReader); ok {
		statsByAccount, err := batchReader.GetAccountWindowStatsBatch(ctx, uniqueIDs, startTime)
		if err == nil {
			for _, accountID := range uniqueIDs {
				result[accountID] = windowStatsFromAccountStats(statsByAccount[accountID])
			}
			return result, nil
		}
	}

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	for _, accountID := range uniqueIDs {
		id := accountID
		g.Go(func() error {
			stats, err := s.usageLogRepo.GetAccountWindowStats(gctx, id, startTime)
			if err != nil {
				return nil
			}
			mu.Lock()
			result[id] = windowStatsFromAccountStats(stats)
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()

	for _, accountID := range uniqueIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = &WindowStats{}
		}
	}
	return result, nil
}

func windowStatsFromAccountStats(stats *usagestats.AccountStats) *WindowStats {
	if stats == nil {
		return &WindowStats{}
	}
	return &WindowStats{
		Requests:     stats.Requests,
		Tokens:       stats.Tokens,
		Cost:         stats.Cost,
		StandardCost: stats.StandardCost,
		UserCost:     stats.UserCost,
	}
}

func (s *AccountUsageService) GetAccountUsageStats(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.AccountUsageStatsResponse, error) {
	stats, err := s.usageLogRepo.GetAccountUsageStats(ctx, accountID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get account usage stats failed: %w", err)
	}
	return stats, nil
}

// fetchOAuthUsageRaw 从 Anthropic API 获取原始响应（不构建 UsageInfo）
// 如果账号开启了 TLS 指纹，则使用 TLS 指纹伪装
// 如果有缓存的 Fingerprint，则使用缓存的 User-Agent 等信息
func (s *AccountUsageService) fetchOAuthUsageRaw(ctx context.Context, account *Account) (*ClaudeUsageResponse, error) {
	accessToken := account.GetCredential("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("no access token available")
	}

	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 构建完整的选项
	opts := &ClaudeUsageFetchOptions{
		AccessToken:          accessToken,
		ProxyURL:             proxyURL,
		AccountID:            account.ID,
		EnableTLSFingerprint: account.IsTLSFingerprintEnabled(),
	}

	// 尝试获取缓存的 Fingerprint（包含 User-Agent 等信息）
	if s.identityCache != nil {
		if fp, err := s.identityCache.GetFingerprint(ctx, account.ID); err == nil && fp != nil {
			opts.Fingerprint = fp
		}
	}

	return s.usageFetcher.FetchUsageWithOptions(ctx, opts)
}

// parseTime 尝试多种格式解析时间
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func (s *AccountUsageService) tryClearRecoverableAccountError(ctx context.Context, account *Account) {
	if account == nil || account.Status != StatusError {
		return
	}

	msg := strings.ToLower(strings.TrimSpace(account.ErrorMessage))
	if msg == "" {
		return
	}

	if !strings.Contains(msg, "token refresh failed") &&
		!strings.Contains(msg, "invalid_client") &&
		!strings.Contains(msg, "missing_project_id") &&
		!strings.Contains(msg, "unauthenticated") {
		return
	}

	if err := s.accountRepo.ClearError(ctx, account.ID); err != nil {
		log.Printf("[usage] failed to clear recoverable account error for account %d: %v", account.ID, err)
		return
	}

	account.Status = StatusActive
	account.ErrorMessage = ""
}

// buildUsageInfo 构建UsageInfo
func (s *AccountUsageService) buildUsageInfo(resp *ClaudeUsageResponse, updatedAt *time.Time) *UsageInfo {
	info := &UsageInfo{
		UpdatedAt: updatedAt,
	}

	// 5小时窗口 - 始终创建对象（即使 ResetsAt 为空）
	info.FiveHour = &UsageProgress{
		Utilization: resp.FiveHour.Utilization,
	}
	if resp.FiveHour.ResetsAt != "" {
		if fiveHourReset, err := parseTime(resp.FiveHour.ResetsAt); err == nil {
			info.FiveHour.ResetsAt = &fiveHourReset
			info.FiveHour.RemainingSeconds = int(time.Until(fiveHourReset).Seconds())
		} else {
			log.Printf("Failed to parse FiveHour.ResetsAt: %s, error: %v", resp.FiveHour.ResetsAt, err)
		}
	}

	// 7天窗口
	if resp.SevenDay.ResetsAt != "" {
		if sevenDayReset, err := parseTime(resp.SevenDay.ResetsAt); err == nil {
			info.SevenDay = &UsageProgress{
				Utilization:      resp.SevenDay.Utilization,
				ResetsAt:         &sevenDayReset,
				RemainingSeconds: int(time.Until(sevenDayReset).Seconds()),
			}
		} else {
			log.Printf("Failed to parse SevenDay.ResetsAt: %s, error: %v", resp.SevenDay.ResetsAt, err)
			info.SevenDay = &UsageProgress{
				Utilization: resp.SevenDay.Utilization,
			}
		}
	}

	// 7天Sonnet窗口
	if resp.SevenDaySonnet.ResetsAt != "" {
		if sonnetReset, err := parseTime(resp.SevenDaySonnet.ResetsAt); err == nil {
			info.SevenDaySonnet = &UsageProgress{
				Utilization:      resp.SevenDaySonnet.Utilization,
				ResetsAt:         &sonnetReset,
				RemainingSeconds: int(time.Until(sonnetReset).Seconds()),
			}
		} else {
			log.Printf("Failed to parse SevenDaySonnet.ResetsAt: %s, error: %v", resp.SevenDaySonnet.ResetsAt, err)
			info.SevenDaySonnet = &UsageProgress{
				Utilization: resp.SevenDaySonnet.Utilization,
			}
		}
	}

	return info
}

// estimateSetupTokenUsage 根据session_window推算Setup Token账号的使用量
func (s *AccountUsageService) estimateSetupTokenUsage(account *Account) *UsageInfo {
	info := &UsageInfo{}

	// 如果有session_window信息
	if account.SessionWindowEnd != nil {
		remaining := int(time.Until(*account.SessionWindowEnd).Seconds())
		if remaining < 0 {
			remaining = 0
		}

		// 根据状态估算使用率 (百分比形式，100 = 100%)
		var utilization float64
		switch account.SessionWindowStatus {
		case "rejected":
			utilization = 100.0
		case "allowed_warning":
			utilization = 80.0
		default:
			utilization = 0.0
		}

		info.FiveHour = &UsageProgress{
			Utilization:      utilization,
			ResetsAt:         account.SessionWindowEnd,
			RemainingSeconds: remaining,
		}
	} else {
		// 没有窗口信息，返回空数据
		info.FiveHour = &UsageProgress{
			Utilization:      0,
			RemainingSeconds: 0,
		}
	}

	// Setup Token无法获取7d数据
	return info
}

func buildGeminiUsageProgress(used, limit int64, resetAt time.Time, tokens int64, cost float64, now time.Time) *UsageProgress {
	// limit <= 0 means "no local quota window" (unknown or unlimited).
	if limit <= 0 {
		return nil
	}
	utilization := (float64(used) / float64(limit)) * 100
	remainingSeconds := int(resetAt.Sub(now).Seconds())
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	resetCopy := resetAt
	return &UsageProgress{
		Utilization:      utilization,
		ResetsAt:         &resetCopy,
		RemainingSeconds: remainingSeconds,
		UsedRequests:     used,
		LimitRequests:    limit,
		WindowStats: &WindowStats{
			Requests: used,
			Tokens:   tokens,
			Cost:     cost,
		},
	}
}

// GetAccountWindowStats 获取账号在指定时间窗口内的使用统计
// 用于账号列表页面显示当前窗口费用
func (s *AccountUsageService) GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error) {
	return s.usageLogRepo.GetAccountWindowStats(ctx, accountID, startTime)
}

// getKiroUsage 获取 Kiro 账户日使用量（从本地日志统计）+ 订阅额度（从 CodeWhisperer API）
func (s *AccountUsageService) getKiroUsage(ctx context.Context, account *Account) (*UsageInfo, error) {
	now := time.Now()
	usage := &UsageInfo{
		UpdatedAt: &now,
		KiroQuota: make(map[string]*KiroModelUsage),
	}

	if s.usageLogRepo == nil {
		return usage, nil
	}

	// 查询今日使用量（UTC 零点开始）
	dayStart := now.UTC().Truncate(24 * time.Hour)
	stats, err := s.usageLogRepo.GetModelStatsWithFilters(ctx, dayStart, now, 0, 0, account.ID, 0, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get kiro usage stats failed: %w", err)
	}

	// 按规范名分组聚合（合并同一模型的不同变体）
	for _, stat := range stats {
		canonical := stat.Model
		if mapped, ok := kiro.CanonicalModelName[stat.Model]; ok {
			canonical = mapped
		}

		if existing, ok := usage.KiroQuota[canonical]; ok {
			existing.Requests += stat.Requests
			existing.Tokens += stat.TotalTokens
			existing.Cost += stat.Cost
		} else {
			usage.KiroQuota[canonical] = &KiroModelUsage{
				Requests: stat.Requests,
				Tokens:   stat.TotalTokens,
				Cost:     stat.Cost,
			}
		}
	}

	// 获取订阅额度（从 CodeWhisperer API，带缓存）
	usage.KiroSubscription = s.fetchKiroSubscription(ctx, account)

	return usage, nil
}

// fetchKiroSubscription 获取 Kiro 订阅额度信息（带 5 分钟缓存）
func (s *AccountUsageService) fetchKiroSubscription(ctx context.Context, account *Account) *KiroSubscriptionQuota {
	// 1. 检查缓存
	if cached, ok := s.cache.kiroSubscriptionCache.Load(account.ID); ok {
		if cache, ok := cached.(*kiroSubscriptionCache); ok && time.Since(cache.timestamp) < kiroSubscriptionCacheTTL {
			return cache.quota
		}
	}

	// 2. 获取 access_token
	accessToken := account.GetCredential("access_token")
	if accessToken == "" {
		// 尝试刷新 token
		if s.kiroOAuthService != nil {
			tokenInfo, err := s.kiroOAuthService.RefreshAccountToken(ctx, account)
			if err != nil {
				log.Printf("[fetchKiroSubscription] Account %d: token refresh failed: %v", account.ID, err)
				return nil
			}
			accessToken = tokenInfo.AccessToken
		} else {
			return nil
		}
	}

	machineId := account.GetCredential("machine_id")

	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 3. 调用 API
	httpClient := &httpUpstreamAdapter{upstream: s.kiroOAuthService.httpUpstream}
	usageLimits, err := kiro.GetUsageLimits(httpClient, accessToken, machineId, proxyURL)
	if err != nil {
		log.Printf("[fetchKiroSubscription] Account %d: GetUsageLimits failed: %v", account.ID, err)
		return nil
	}

	// 4. 解析响应
	quota := &KiroSubscriptionQuota{}

	// 解析用户信息
	if usageLimits.UserInfo != nil {
		quota.Email = usageLimits.UserInfo.Email
	}

	// 解析订阅类型
	if usageLimits.SubscriptionInfo != nil {
		titleOrName := usageLimits.SubscriptionInfo.SubscriptionTitle
		if titleOrName == "" {
			titleOrName = usageLimits.SubscriptionInfo.SubscriptionName
		}
		if titleOrName == "" {
			titleOrName = usageLimits.SubscriptionInfo.SubscriptionType
		}
		quota.SubscriptionType = parseKiroSubscriptionType(titleOrName)
	}

	// 解析使用量：
	// Kiro 可能返回多个 usage breakdown，且额外额度常放在 bonuses 中。
	// 仅看 usageLimit 会把 550 显示成 50（基础额度），这里改为按“基础额度 + bonus额度”计算有效额度。
	if len(usageLimits.UsageBreakdownList) > 0 {
		selected := usageLimits.UsageBreakdownList[0]
		selectedEffectiveCurrent := selected.CurrentUsage
		selectedEffectiveLimit := selected.UsageLimit
		for _, bonus := range selected.Bonuses {
			if bonus.UsageLimit > 0 {
				selectedEffectiveLimit += bonus.UsageLimit
			}
			if bonus.CurrentUsage > 0 {
				selectedEffectiveCurrent += bonus.CurrentUsage
			}
		}

		for i := 1; i < len(usageLimits.UsageBreakdownList); i++ {
			candidate := usageLimits.UsageBreakdownList[i]
			candidateEffectiveCurrent := candidate.CurrentUsage
			candidateEffectiveLimit := candidate.UsageLimit
			for _, bonus := range candidate.Bonuses {
				if bonus.UsageLimit > 0 {
					candidateEffectiveLimit += bonus.UsageLimit
				}
				if bonus.CurrentUsage > 0 {
					candidateEffectiveCurrent += bonus.CurrentUsage
				}
			}
			if candidateEffectiveLimit > selectedEffectiveLimit {
				selected = candidate
				selectedEffectiveCurrent = candidateEffectiveCurrent
				selectedEffectiveLimit = candidateEffectiveLimit
			}
		}

			baseCurrentUsage := selectedEffectiveCurrent
			baseUsageLimit := selectedEffectiveLimit

			quota.CurrentUsage = baseCurrentUsage
			quota.UsageLimit = baseUsageLimit
			if quota.UsageLimit > 0 {
				quota.UsagePercent = (quota.CurrentUsage / quota.UsageLimit) * 100
			}

		// 试用配额也取最大 usageLimit 的条目，避免被较小分项覆盖。
			for _, breakdown := range usageLimits.UsageBreakdownList {
				if breakdown.FreeTrialInfo == nil {
					continue
				}
				if breakdown.FreeTrialInfo.UsageLimit >= quota.TrialLimit {
					quota.TrialCurrent = breakdown.FreeTrialInfo.CurrentUsage
					quota.TrialLimit = breakdown.FreeTrialInfo.UsageLimit
					quota.TrialStatus = breakdown.FreeTrialInfo.FreeTrialStatus
				}
			}

			// FREE/试用账号展示兜底：
			// - 常见场景下基础额度与 trial 额度是两段独立配额（例如 50 + 500）
			// - 优先按“基础 + trial”合并显示；若不满足则回退到较大值。
			displayUsageLimit := baseUsageLimit
			displayCurrentUsage := baseCurrentUsage
			if strings.EqualFold(quota.SubscriptionType, "FREE") && baseUsageLimit > 0 && quota.TrialLimit > baseUsageLimit {
				displayUsageLimit = baseUsageLimit + quota.TrialLimit
				displayCurrentUsage = baseCurrentUsage + quota.TrialCurrent
			} else {
				if quota.TrialLimit > displayUsageLimit {
					displayUsageLimit = quota.TrialLimit
				}
				if quota.TrialCurrent > displayCurrentUsage {
					displayCurrentUsage = quota.TrialCurrent
				}
			}
			quota.UsageLimit = displayUsageLimit
			quota.CurrentUsage = displayCurrentUsage
			if quota.UsageLimit > 0 {
				quota.UsagePercent = (quota.CurrentUsage / quota.UsageLimit) * 100
			}
		}

	// 解析重置日期
	if usageLimits.NextDateReset != "" {
		if ts, err := usageLimits.NextDateReset.Int64(); err == nil && ts > 0 {
			quota.NextResetDate = time.Unix(ts, 0).Format("2006-01-02")
		} else if f, err := usageLimits.NextDateReset.Float64(); err == nil && f > 0 {
			quota.NextResetDate = time.Unix(int64(f), 0).Format("2006-01-02")
		}
	}

	// 5. 缓存结果
	s.cache.kiroSubscriptionCache.Store(account.ID, &kiroSubscriptionCache{
		quota:     quota,
		timestamp: time.Now(),
	})

	return quota
}

// parseKiroSubscriptionType 解析 Kiro 订阅类型字符串
func parseKiroSubscriptionType(raw string) string {
	upper := strings.ToUpper(raw)
	if strings.Contains(upper, "PRO_PLUS") || strings.Contains(upper, "PROPLUS") {
		return "PRO_PLUS"
	}
	if strings.Contains(upper, "POWER") {
		return "POWER"
	}
	if strings.Contains(upper, "PRO") {
		return "PRO"
	}
	return "FREE"
}

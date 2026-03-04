package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ReferralStats 推荐统计
type ReferralStats struct {
	TotalInvited    int     `json:"total_invited"`
	TotalCommission float64 `json:"total_commission"`
	CommissionRate  float64 `json:"commission_rate"`
}

// ReferralInvitee 被邀请用户
type ReferralInvitee struct {
	UserID          int64     `json:"user_id"`
	Email           string    `json:"email"` // masked
	CreatedAt       time.Time `json:"created_at"`
	TotalCommission float64   `json:"total_commission"`
}

// ReferralCommission 返利记录
type ReferralCommission struct {
	ID               int64     `json:"id"`
	OrderAmount      float64   `json:"order_amount"`
	CommissionRate   float64   `json:"commission_rate"`
	CommissionAmount float64   `json:"commission_amount"`
	CreatedAt        time.Time `json:"created_at"`
}

// ReferralService 推荐返利服务
type ReferralService struct {
	db         *sql.DB
	settingSvc *SettingService
}

// NewReferralService 创建推荐返利服务
func NewReferralService(db *sql.DB, settingSvc *SettingService) *ReferralService {
	return &ReferralService{
		db:         db,
		settingSvc: settingSvc,
	}
}

// GetStats 获取推荐统计
func (s *ReferralService) GetStats(ctx context.Context, userID int64) (*ReferralStats, error) {
	stats := &ReferralStats{}

	// 当前返利比例
	stats.CommissionRate = s.settingSvc.GetReferralCommissionRate(ctx)

	// 总邀请人数
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE referrer_id = $1`,
		userID,
	).Scan(&stats.TotalInvited)
	if err != nil {
		return nil, fmt.Errorf("count invitees: %w", err)
	}

	// 总返利金额
	var totalCommission sql.NullFloat64
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(commission_amount), 0) FROM referral_commissions WHERE referrer_id = $1`,
		userID,
	).Scan(&totalCommission)
	if err != nil {
		return nil, fmt.Errorf("sum commissions: %w", err)
	}
	if totalCommission.Valid {
		stats.TotalCommission = totalCommission.Float64
	}

	return stats, nil
}

// GetInvitees 获取被邀请用户列表（分页）
func (s *ReferralService) GetInvitees(ctx context.Context, userID int64, page, pageSize int) ([]ReferralInvitee, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE referrer_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count invitees: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.created_at, COALESCE(rc.total_commission, 0)
		 FROM users u
		 LEFT JOIN (
		   SELECT referred_user_id, SUM(commission_amount) AS total_commission
		   FROM referral_commissions
		   WHERE referrer_id = $1
		   GROUP BY referred_user_id
		 ) rc ON rc.referred_user_id = u.id
		 WHERE u.referrer_id = $1
		 ORDER BY u.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query invitees: %w", err)
	}
	defer rows.Close()

	var invitees []ReferralInvitee
	for rows.Next() {
		var inv ReferralInvitee
		if err := rows.Scan(&inv.UserID, &inv.Email, &inv.CreatedAt, &inv.TotalCommission); err != nil {
			return nil, 0, fmt.Errorf("scan invitee: %w", err)
		}
		inv.Email = maskEmail(inv.Email)
		invitees = append(invitees, inv)
	}

	return invitees, total, nil
}

// GetCommissions 获取返利记录（分页）
func (s *ReferralService) GetCommissions(ctx context.Context, userID int64, page, pageSize int) ([]ReferralCommission, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM referral_commissions WHERE referrer_id = $1`,
		userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count commissions: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, order_amount, commission_rate, commission_amount, created_at
		 FROM referral_commissions
		 WHERE referrer_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query commissions: %w", err)
	}
	defer rows.Close()

	var commissions []ReferralCommission
	for rows.Next() {
		var c ReferralCommission
		if err := rows.Scan(&c.ID, &c.OrderAmount, &c.CommissionRate, &c.CommissionAmount, &c.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan commission: %w", err)
		}
		commissions = append(commissions, c)
	}

	return commissions, total, nil
}

// maskEmail 遮蔽邮箱地址: ab***@example.com
func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 2 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:2] + "***@" + parts[1]
}

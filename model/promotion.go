package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// PromotionRule 用户等级（分组）自动晋升规则：
// 当用户累计实付充值金额（按展示币种归一）达到 MinPaidAmount 时，
// 自动将 users.group 从 FromGroup 提升为 ToGroup。支持链式晋升。
type PromotionRule struct {
	Id            int     `json:"id"`
	FromGroup     string  `json:"from_group" gorm:"type:varchar(64);index"`
	ToGroup       string  `json:"to_group" gorm:"type:varchar(64)"`
	MinPaidAmount float64 `json:"min_paid_amount"`
	Enabled       bool    `json:"enabled"`
	Remark        string  `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

// PromotionLog 记录每一次自动晋升，便于审计。
type PromotionLog struct {
	Id         int     `json:"id"`
	UserId     int     `json:"user_id" gorm:"index"`
	Username   string  `json:"username" gorm:"type:varchar(64);default:''"`
	FromGroup  string  `json:"from_group" gorm:"type:varchar(64)"`
	ToGroup    string  `json:"to_group" gorm:"type:varchar(64)"`
	PaidAmount float64 `json:"paid_amount"`
	RuleId     int     `json:"rule_id"`
	CreatedAt  int64   `json:"created_at"`
}

// promotionMaxChainDepth 链式晋升的最大级数，防止规则成环导致死循环。
const promotionMaxChainDepth = 10

// promotionPaidThresholdEpsilon 容忍浮点累计误差，避免恰好达到门槛时不晋升。
const promotionPaidThresholdEpsilon = 1e-6

// promotionDisplayCurrency 返回当前站点的金额展示币种（CNY 或 USD）。
// TOKENS/CUSTOM 展示类型按 USD 口径统计。
func promotionDisplayCurrency() string {
	if operation_setting.IsCNYDisplay() {
		return "CNY"
	}
	return "USD"
}

// normalizeTopUpMoneyToDisplay 将单笔 top_ups.money 归一到展示币种。
// epay 的 Money 是人民币，balance 跟随展示币种，其余 provider 的 Money 是美元。
// 汇率异常（<=0）时跳过换算并记录系统错误。
func normalizeTopUpMoneyToDisplay(money float64, paymentProvider string) float64 {
	rate := operation_setting.USDExchangeRate
	if promotionDisplayCurrency() == "CNY" {
		switch paymentProvider {
		case PaymentProviderEpay, PaymentProviderBalance:
			return money
		}
		if rate <= 0 {
			common.SysError(fmt.Sprintf("promotion: invalid USD exchange rate %v, skip conversion for provider %s", rate, paymentProvider))
			return money
		}
		return money * rate
	}
	// USD 展示：仅 epay（人民币）需要换算
	if paymentProvider == PaymentProviderEpay {
		if rate <= 0 {
			common.SysError(fmt.Sprintf("promotion: invalid USD exchange rate %v, skip conversion for provider %s", rate, paymentProvider))
			return money
		}
		return money / rate
	}
	return money
}

// GetUserTotalPaidAmount 返回用户累计实付充值金额（按展示币种归一）。
// 按 payment_provider 分组聚合后在 Go 侧换算，避免跨方言的 SQL CASE。
func GetUserTotalPaidAmount(userId int) (float64, error) {
	type providerSum struct {
		PaymentProvider string  `gorm:"column:payment_provider"`
		Total           float64 `gorm:"column:total"`
	}
	var sums []providerSum
	err := DB.Model(&TopUp{}).
		Select("payment_provider, SUM(money) AS total").
		Where("user_id = ? AND status = ?", userId, common.TopUpStatusSuccess).
		Group("payment_provider").
		Scan(&sums).Error
	if err != nil {
		return 0, err
	}
	total := 0.0
	for _, s := range sums {
		total += normalizeTopUpMoneyToDisplay(s.Total, s.PaymentProvider)
	}
	return total, nil
}

// EvaluateAndPromoteUser 检查用户是否满足晋升规则，满足则更新分组并记录日志。
// 支持链式晋升（default→vip→enterprise）。任何错误只记录不返回，绝不影响支付流程。
func EvaluateAndPromoteUser(userId int) {
	if userId <= 0 {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			common.SysError(fmt.Sprintf("promotion evaluation panic for user %d: %v", userId, r))
		}
	}()
	if err := evaluateAndPromoteUser(userId); err != nil {
		common.SysError(fmt.Sprintf("promotion evaluation failed for user %d: %v", userId, err))
	}
}

func evaluateAndPromoteUser(userId int) error {
	rules, err := getEnabledPromotionRules()
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}
	totalPaid, err := GetUserTotalPaidAmount(userId)
	if err != nil {
		return err
	}
	for depth := 0; depth < promotionMaxChainDepth; depth++ {
		group, err := GetUserGroup(userId, true)
		if err != nil {
			return err
		}
		// 当前分组下满足门槛的规则中取门槛最高者
		var best *PromotionRule
		for i := range rules {
			rule := &rules[i]
			if rule.FromGroup != group || rule.ToGroup == "" || rule.ToGroup == group {
				continue
			}
			if totalPaid+promotionPaidThresholdEpsilon < rule.MinPaidAmount {
				continue
			}
			if best == nil || rule.MinPaidAmount > best.MinPaidAmount {
				best = rule
			}
		}
		if best == nil {
			return nil
		}
		if err := DB.Model(&User{}).Where("id = ?", userId).Update("group", best.ToGroup).Error; err != nil {
			return err
		}
		if err := RefreshUserGroupCache(userId); err != nil {
			common.SysError(fmt.Sprintf("failed to refresh user group cache after promotion for user %d: %v", userId, err))
		}
		username, _ := GetUsernameById(userId, true)
		promotionLog := &PromotionLog{
			UserId:     userId,
			Username:   username,
			FromGroup:  best.FromGroup,
			ToGroup:    best.ToGroup,
			PaidAmount: totalPaid,
			RuleId:     best.Id,
			CreatedAt:  common.GetTimestamp(),
		}
		if err := DB.Create(promotionLog).Error; err != nil {
			return err
		}
		RecordLog(userId, LogTypeManage, fmt.Sprintf("自动晋升：%s → %s（累计实付 %.2f）", best.FromGroup, best.ToGroup, totalPaid))
	}
	common.SysError(fmt.Sprintf("promotion chain for user %d exceeded max depth %d, possible rule cycle", userId, promotionMaxChainDepth))
	return nil
}

// GetAllPromotionRules 返回全部晋升规则（管理员使用）。
func GetAllPromotionRules() (rules []*PromotionRule, err error) {
	err = DB.Order("id asc").Find(&rules).Error
	return rules, err
}

func getEnabledPromotionRules() (rules []PromotionRule, err error) {
	err = DB.Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

func AddPromotionRule(rule *PromotionRule) error {
	now := common.GetTimestamp()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return DB.Create(rule).Error
}

func UpdatePromotionRule(rule *PromotionRule) error {
	// 使用 map 更新，确保 Enabled=false 等零值也能写入
	return DB.Model(&PromotionRule{}).Where("id = ?", rule.Id).Updates(map[string]interface{}{
		"from_group":      rule.FromGroup,
		"to_group":        rule.ToGroup,
		"min_paid_amount": rule.MinPaidAmount,
		"enabled":         rule.Enabled,
		"remark":          rule.Remark,
		"updated_at":      common.GetTimestamp(),
	}).Error
}

func DeletePromotionRule(id int) error {
	return DB.Where("id = ?", id).Delete(&PromotionRule{}).Error
}

// GetPromotionLogs 分页查询晋升日志（管理员使用，按 id 倒序）。
func GetPromotionLogs(pageInfo *common.PageInfo) (logs []*PromotionLog, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&PromotionLog{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&logs).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetPaidStatsByGroup 统计 [startTs, endTs) 内成功充值的实付金额（按展示币种归一），
// 按用户当前分组与按日（本地时区）聚合。top_ups 与 users 同库，允许 join。
// 在 Go 侧按 CompleteTime 归日，避免 SQL 日期函数的方言差异。
func GetPaidStatsByGroup(startTs, endTs int64) (byGroup map[string]float64, byDay map[string]float64, err error) {
	type paidRow struct {
		Group           string  `gorm:"column:grp"`
		CompleteTime    int64   `gorm:"column:complete_time"`
		Money           float64 `gorm:"column:money"`
		PaymentProvider string  `gorm:"column:payment_provider"`
	}
	var rows []paidRow
	err = DB.Model(&TopUp{}).
		Select("users."+commonGroupCol+" AS grp, top_ups.complete_time, top_ups.money, top_ups.payment_provider").
		Joins("JOIN users ON users.id = top_ups.user_id").
		Where("top_ups.status = ? AND top_ups.complete_time >= ? AND top_ups.complete_time < ?", common.TopUpStatusSuccess, startTs, endTs).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	byGroup = make(map[string]float64)
	byDay = make(map[string]float64)
	for _, row := range rows {
		amount := normalizeTopUpMoneyToDisplay(row.Money, row.PaymentProvider)
		day := time.Unix(row.CompleteTime, 0).Local().Format("2006-01-02")
		byGroup[row.Group] += amount
		byDay[day] += amount
	}
	return byGroup, byDay, nil
}

// GetConsumeStatsByGroup 统计 [startTs, endTs) 内消耗日志（type=LogTypeConsume）的额度，
// 按用户当前分组与按日（本地时区）聚合。logs 在 LOG_DB（可能独立库），
// 不能跨库 join，因此先取日志明细再批量回查 users 表映射分组。
func GetConsumeStatsByGroup(startTs, endTs int64) (byGroup map[string]int64, byDay map[string]int64, err error) {
	type consumeRow struct {
		UserId    int   `gorm:"column:user_id"`
		CreatedAt int64 `gorm:"column:created_at"`
		Quota     int64 `gorm:"column:quota"`
	}
	var rows []consumeRow
	err = LOG_DB.Model(&Log{}).
		Select("user_id, created_at, quota").
		Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, startTs, endTs).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}

	userIds := make([]int, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.UserId]; !ok {
			seen[row.UserId] = struct{}{}
			userIds = append(userIds, row.UserId)
		}
	}
	userGroups := make(map[int]string, len(userIds))
	if len(userIds) > 0 {
		type groupRow struct {
			Id    int    `gorm:"column:id"`
			Group string `gorm:"column:grp"`
		}
		var groupRows []groupRow
		if err = DB.Model(&User{}).
			Select("id, "+commonGroupCol+" AS grp").
			Where("id IN ?", userIds).
			Scan(&groupRows).Error; err != nil {
			return nil, nil, err
		}
		for _, gr := range groupRows {
			userGroups[gr.Id] = gr.Group
		}
	}

	byGroup = make(map[string]int64)
	byDay = make(map[string]int64)
	for _, row := range rows {
		group, ok := userGroups[row.UserId]
		if !ok {
			// 用户可能已被删除，归入 unknown 避免漏计成本
			group = "unknown"
		}
		day := time.Unix(row.CreatedAt, 0).Local().Format("2006-01-02")
		byGroup[group] += row.Quota
		byDay[day] += row.Quota
	}
	return byGroup, byDay, nil
}

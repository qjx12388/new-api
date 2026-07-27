package controller

import (
	"sort"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// promotionMaxMinPaidAmount 晋升门槛金额上限，防止异常大数进入计费/比较链路
const promotionMaxMinPaidAmount = 1e9

// validatePromotionRule 校验晋升规则的必填字段与取值范围，返回错误信息（为空表示通过）
func validatePromotionRule(rule *model.PromotionRule) string {
	if rule.FromGroup == "" || rule.ToGroup == "" {
		return "晋升前后分组不能为空"
	}
	if rule.FromGroup == rule.ToGroup {
		return "晋升前后分组不能相同"
	}
	if rule.MinPaidAmount <= 0 {
		return "晋升门槛金额必须大于 0"
	}
	if rule.MinPaidAmount > promotionMaxMinPaidAmount {
		return "晋升门槛金额过大"
	}
	if !ratio_setting.ContainsGroupRatio(rule.FromGroup) {
		return "晋升前分组不存在于分组倍率配置中"
	}
	if !ratio_setting.ContainsGroupRatio(rule.ToGroup) {
		return "晋升后分组不存在于分组倍率配置中"
	}
	return ""
}

func GetPromotionRules(c *gin.Context) {
	rules, err := model.GetAllPromotionRules()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
	return
}

func AddPromotionRule(c *gin.Context) {
	rule := model.PromotionRule{}
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if msg := validatePromotionRule(&rule); msg != "" {
		common.ApiErrorMsg(c, msg)
		return
	}
	if err := model.AddPromotionRule(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &rule)
	return
}

func UpdatePromotionRule(c *gin.Context) {
	rule := model.PromotionRule{}
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	if rule.Id <= 0 {
		common.ApiErrorMsg(c, "无效的规则 ID")
		return
	}
	if msg := validatePromotionRule(&rule); msg != "" {
		common.ApiErrorMsg(c, msg)
		return
	}
	if err := model.UpdatePromotionRule(&rule); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &rule)
	return
}

func DeletePromotionRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeletePromotionRule(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
	return
}

func GetPromotionLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logs, total, err := model.GetPromotionLogs(pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

type promotionGroupStat struct {
	Group   string   `json:"group"`
	Paid    float64  `json:"paid"`
	Consume float64  `json:"consume"`
	Ratio   *float64 `json:"ratio"`
}

type promotionDayStat struct {
	Day     string  `json:"day"`
	Paid    float64 `json:"paid"`
	Consume float64 `json:"consume"`
}

// GetPromotionRevenueStats 按用户分组统计充值实付与消耗（均折算为展示币种），
// 用于评估各等级用户的营收贡献。缺省统计区间为当月 1 号 00:00（本地时间）至今。
func GetPromotionRevenueStats(c *gin.Context) {
	now := time.Now()
	startTs, err := strconv.ParseInt(c.Query("start_ts"), 10, 64)
	if err != nil || startTs <= 0 {
		startTs = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Unix()
	}
	endTs, err := strconv.ParseInt(c.Query("end_ts"), 10, 64)
	if err != nil || endTs <= 0 {
		endTs = now.Unix()
	}
	if endTs <= startTs {
		common.ApiErrorMsg(c, "invalid time range")
		return
	}

	paidByGroup, paidByDay, err := model.GetPaidStatsByGroup(startTs, endTs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	consumeByGroup, consumeByDay, err := model.GetConsumeStatsByGroup(startTs, endTs)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	currency := "USD"
	isCNY := operation_setting.IsCNYDisplay()
	if isCNY {
		currency = "CNY"
	}
	usdExchangeRate := operation_setting.USDExchangeRate
	// 消耗以 quota 存储，折算成美元金额；CNY 展示时再乘汇率
	quotaToDisplay := func(quota int64) float64 {
		amount := float64(quota) / common.QuotaPerUnit
		if isCNY && usdExchangeRate > 0 {
			amount *= usdExchangeRate
		}
		return amount
	}

	// 按分组：取充值与消耗分组的并集，paid<=0 时 ratio 为 null
	groupSet := make(map[string]struct{}, len(paidByGroup)+len(consumeByGroup))
	for group := range paidByGroup {
		groupSet[group] = struct{}{}
	}
	for group := range consumeByGroup {
		groupSet[group] = struct{}{}
	}
	groupNames := make([]string, 0, len(groupSet))
	for group := range groupSet {
		groupNames = append(groupNames, group)
	}
	sort.Strings(groupNames)
	groups := make([]promotionGroupStat, 0, len(groupNames))
	for _, group := range groupNames {
		paid := paidByGroup[group]
		consume := quotaToDisplay(consumeByGroup[group])
		var ratio *float64
		if paid > 0 {
			r := consume / paid
			ratio = &r
		}
		groups = append(groups, promotionGroupStat{
			Group:   group,
			Paid:    paid,
			Consume: consume,
			Ratio:   ratio,
		})
	}

	// 按日：补齐区间内每一天（无数据为 0），按日期升序
	days := make([]promotionDayStat, 0)
	startDay := time.Unix(startTs, 0).Local()
	startDay = time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, startDay.Location())
	for day := startDay; day.Unix() < endTs; day = day.AddDate(0, 0, 1) {
		dayStr := day.Format("2006-01-02")
		days = append(days, promotionDayStat{
			Day:     dayStr,
			Paid:    paidByDay[dayStr],
			Consume: quotaToDisplay(consumeByDay[dayStr]),
		})
	}

	common.ApiSuccess(c, gin.H{
		"currency": currency,
		"start_ts": startTs,
		"end_ts":   endTs,
		"groups":   groups,
		"days":     days,
	})
	return
}

package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupPromotionControllerTest 用内存 SQLite 替换主库与日志库，并关闭 Redis（client 为 nil 会 panic）
func setupPromotionControllerTest(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousMainType := common.MainDatabaseType()
	previousLogType := common.LogDatabaseType()
	previousRedisEnabled := common.RedisEnabled

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.PromotionRule{}, &model.PromotionLog{}, &model.Log{}))

	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	// LOG_SQL_DSN 为空时 InitLogDB 令 LOG_DB = DB（与默认 SQLite 部署一致），
	// 同时触发 initCol 初始化方言列名（commonGroupCol 等）
	require.NoError(t, model.InitLogDB())
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousMainType)
		common.SetLogDatabaseType(previousLogType)
		common.RedisEnabled = previousRedisEnabled
	})
}

func performPromotionJSONRequest(t *testing.T, handler gin.HandlerFunc, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := common.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, reader)
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)
	return recorder
}

type promotionTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func decodePromotionResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) (bool, T) {
	t.Helper()
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    T      `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Success, response.Data
}

func TestPromotionRuleValidation(t *testing.T) {
	setupPromotionControllerTest(t)

	invalidRules := []model.PromotionRule{
		{FromGroup: "", ToGroup: "vip", MinPaidAmount: 100},            // 空组
		{FromGroup: "default", ToGroup: "default", MinPaidAmount: 100}, // 同组
		{FromGroup: "default", ToGroup: "vip", MinPaidAmount: -1},      // 负金额
		{FromGroup: "default", ToGroup: "vip", MinPaidAmount: 2e9},     // 超大金额
		{FromGroup: "nonexistent", ToGroup: "vip", MinPaidAmount: 100}, // 分组不存在
	}
	for _, rule := range invalidRules {
		recorder := performPromotionJSONRequest(t, AddPromotionRule, http.MethodPost, "/api/promotion/rule", rule)
		var response promotionTestResponse
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.False(t, response.Success, "rule %+v should be rejected", rule)
		assert.NotEmpty(t, response.Message)
	}

	var ruleCount int64
	require.NoError(t, model.DB.Model(&model.PromotionRule{}).Count(&ruleCount).Error)
	assert.Zero(t, ruleCount)
}

func TestPromotionRuleCRUD(t *testing.T) {
	setupPromotionControllerTest(t)

	// 新增
	rule := model.PromotionRule{
		FromGroup:     "default",
		ToGroup:       "vip",
		MinPaidAmount: 100,
		Enabled:       true,
		Remark:        "crud test",
	}
	recorder := performPromotionJSONRequest(t, AddPromotionRule, http.MethodPost, "/api/promotion/rule", rule)
	success, created := decodePromotionResponse[model.PromotionRule](t, recorder)
	require.True(t, success, "add rule failed: %s", recorder.Body.String())
	require.Positive(t, created.Id)

	// 查询
	recorder = performPromotionJSONRequest(t, GetPromotionRules, http.MethodGet, "/api/promotion/rule", nil)
	success, rules := decodePromotionResponse[[]model.PromotionRule](t, recorder)
	require.True(t, success)
	require.Len(t, rules, 1)
	assert.Equal(t, "default", rules[0].FromGroup)
	assert.Equal(t, "vip", rules[0].ToGroup)
	assert.InDelta(t, 100.0, rules[0].MinPaidAmount, 1e-9)

	// 修改：Enabled=false 等零值也必须能写入
	created.MinPaidAmount = 200
	created.Enabled = false
	created.Remark = "updated"
	recorder = performPromotionJSONRequest(t, UpdatePromotionRule, http.MethodPut, "/api/promotion/rule", created)
	success, _ = decodePromotionResponse[model.PromotionRule](t, recorder)
	require.True(t, success, "update rule failed: %s", recorder.Body.String())
	var stored model.PromotionRule
	require.NoError(t, model.DB.First(&stored, created.Id).Error)
	assert.InDelta(t, 200.0, stored.MinPaidAmount, 1e-9)
	assert.False(t, stored.Enabled)
	assert.Equal(t, "updated", stored.Remark)

	// 删除
	recorder = httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", created.Id)}}
	c.Request = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/promotion/rule/%d", created.Id), nil)
	DeletePromotionRule(c)
	var deleteResponse promotionTestResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &deleteResponse))
	require.True(t, deleteResponse.Success)

	recorder = performPromotionJSONRequest(t, GetPromotionRules, http.MethodGet, "/api/promotion/rule", nil)
	success, rules = decodePromotionResponse[[]model.PromotionRule](t, recorder)
	require.True(t, success)
	assert.Empty(t, rules)
}

func TestGetPromotionRevenueStats(t *testing.T) {
	setupPromotionControllerTest(t)

	// CNY 展示 + 固定汇率，测试结束恢复
	gs := operation_setting.GetGeneralSetting()
	oldDisplayType := gs.QuotaDisplayType
	oldRate := operation_setting.USDExchangeRate
	gs.QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate = 7.0
	t.Cleanup(func() {
		gs.QuotaDisplayType = oldDisplayType
		operation_setting.USDExchangeRate = oldRate
	})

	// 统计区间：前天 00:00 至今天 00:00（本地时间），共两天
	now := time.Now()
	day0 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -2)
	startTs := day0.Unix()
	endTs := day0.AddDate(0, 0, 2).Unix()
	day0Noon := day0.Add(12 * time.Hour).Unix()
	day1Noon := day0.AddDate(0, 0, 1).Add(12 * time.Hour).Unix()

	// 用户（AffCode 有唯一索引，需各不相同；Password 列 not null）
	users := []model.User{
		{Username: "promo-vip", Password: "password1", AffCode: "promo-aff-1", Group: "vip"},
		{Username: "promo-default", Password: "password1", AffCode: "promo-aff-2", Group: "default"},
		{Username: "promo-svip", Password: "password1", AffCode: "promo-aff-3", Group: "svip"},
	}
	for i := range users {
		require.NoError(t, model.DB.Create(&users[i]).Error)
	}

	// 充值：epay 人民币 100（day0，vip），stripe 美元 10（day1，default）
	topUps := []model.TopUp{
		{UserId: users[0].Id, Money: 100, TradeNo: "promo-trade-1", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusSuccess, CreateTime: day0Noon, CompleteTime: day0Noon},
		{UserId: users[1].Id, Money: 10, TradeNo: "promo-trade-2", PaymentProvider: model.PaymentProviderStripe, Status: common.TopUpStatusSuccess, CreateTime: day1Noon, CompleteTime: day1Noon},
	}
	for i := range topUps {
		require.NoError(t, model.DB.Create(&topUps[i]).Error)
	}

	// 消耗日志（LOG_DB）：vip $1（day1）、default $0.5（day1）、svip $2（day0，无充值记录）
	consumeLogs := []model.Log{
		{UserId: users[0].Id, Username: users[0].Username, Type: model.LogTypeConsume, Quota: 500000, CreatedAt: day1Noon},
		{UserId: users[1].Id, Username: users[1].Username, Type: model.LogTypeConsume, Quota: 250000, CreatedAt: day1Noon},
		{UserId: users[2].Id, Username: users[2].Username, Type: model.LogTypeConsume, Quota: 1000000, CreatedAt: day0Noon},
	}
	for i := range consumeLogs {
		require.NoError(t, model.LOG_DB.Create(&consumeLogs[i]).Error)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/promotion/revenue_stats?start_ts=%d&end_ts=%d", startTs, endTs), nil)
	GetPromotionRevenueStats(c)

	type groupStat struct {
		Group   string   `json:"group"`
		Paid    float64  `json:"paid"`
		Consume float64  `json:"consume"`
		Ratio   *float64 `json:"ratio"`
	}
	type dayStat struct {
		Day     string  `json:"day"`
		Paid    float64 `json:"paid"`
		Consume float64 `json:"consume"`
	}
	type statsData struct {
		Currency string      `json:"currency"`
		StartTs  int64       `json:"start_ts"`
		EndTs    int64       `json:"end_ts"`
		Groups   []groupStat `json:"groups"`
		Days     []dayStat   `json:"days"`
	}
	success, data := decodePromotionResponse[statsData](t, recorder)
	require.True(t, success, "revenue stats failed: %s", recorder.Body.String())

	assert.Equal(t, "CNY", data.Currency)
	assert.Equal(t, startTs, data.StartTs)
	assert.Equal(t, endTs, data.EndTs)

	// 分组并集：default（充值 70/消耗 3.5）、svip（仅消耗 14）、vip（充值 100/消耗 7），按名称升序
	require.Len(t, data.Groups, 3)
	assert.Equal(t, "default", data.Groups[0].Group)
	assert.InDelta(t, 70.0, data.Groups[0].Paid, 1e-9)
	assert.InDelta(t, 3.5, data.Groups[0].Consume, 1e-9)
	require.NotNil(t, data.Groups[0].Ratio)
	assert.InDelta(t, 0.05, *data.Groups[0].Ratio, 1e-9)

	assert.Equal(t, "svip", data.Groups[1].Group)
	assert.InDelta(t, 0.0, data.Groups[1].Paid, 1e-9)
	assert.InDelta(t, 14.0, data.Groups[1].Consume, 1e-9)
	assert.Nil(t, data.Groups[1].Ratio)

	assert.Equal(t, "vip", data.Groups[2].Group)
	assert.InDelta(t, 100.0, data.Groups[2].Paid, 1e-9)
	assert.InDelta(t, 7.0, data.Groups[2].Consume, 1e-9)
	require.NotNil(t, data.Groups[2].Ratio)
	assert.InDelta(t, 0.07, *data.Groups[2].Ratio, 1e-9)

	// 按日：区间内每一天都补齐，按日期升序
	require.Len(t, data.Days, 2)
	assert.Equal(t, day0.Format("2006-01-02"), data.Days[0].Day)
	assert.InDelta(t, 100.0, data.Days[0].Paid, 1e-9)
	assert.InDelta(t, 14.0, data.Days[0].Consume, 1e-9)
	assert.Equal(t, day0.AddDate(0, 0, 1).Format("2006-01-02"), data.Days[1].Day)
	assert.InDelta(t, 70.0, data.Days[1].Paid, 1e-9)
	assert.InDelta(t, 10.5, data.Days[1].Consume, 1e-9)
}

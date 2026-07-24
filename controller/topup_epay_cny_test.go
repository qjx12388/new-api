package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
)

// 设置易支付相关的全局状态，并在测试结束时恢复
func setupEpayDisplayState(t *testing.T, displayType string, price float64, usdExchangeRate float64, displayCurrencyAmount bool) {
	t.Helper()
	gs := operation_setting.GetGeneralSetting()
	ps := operation_setting.GetPaymentSetting()
	oldDisplayType := gs.QuotaDisplayType
	oldPrice := operation_setting.Price
	oldRate := operation_setting.USDExchangeRate
	oldDisplayCurrencyAmount := ps.EpayDisplayCurrencyAmountEnabled
	gs.QuotaDisplayType = displayType
	operation_setting.Price = price
	operation_setting.USDExchangeRate = usdExchangeRate
	ps.EpayDisplayCurrencyAmountEnabled = displayCurrencyAmount
	t.Cleanup(func() {
		gs.QuotaDisplayType = oldDisplayType
		operation_setting.Price = oldPrice
		operation_setting.USDExchangeRate = oldRate
		ps.EpayDisplayCurrencyAmountEnabled = oldDisplayCurrencyAmount
	})
}

func TestGetPayMoneyByDisplayType(t *testing.T) {
	// 开关打开 + CNY 展示：面额即人民币，充 ¥100 收 ¥100，不再乘 Price
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeCNY, 7.3, 7.2, true)
	assert.InDelta(t, 100.0, getPayMoney(100, ""), 1e-9)

	// 开关关闭 + CNY 展示：保持旧行为，面额为美元，按 Price 换算收款
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeCNY, 7.3, 7.2, false)
	assert.InDelta(t, 730.0, getPayMoney(100, ""), 1e-9)

	// USD 展示：面额为美元，按 Price（本币/美元）换算收款，开关无效
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeUSD, 7.3, 7.2, true)
	assert.InDelta(t, 730.0, getPayMoney(100, ""), 1e-9)
}

func TestEpayTopupAmountToQuotaByDisplayType(t *testing.T) {
	// 开关打开 + CNY 展示：按汇率折算，充 ¥72（汇率 7.2）到账 $10 额度
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeCNY, 7.3, 7.2, true)
	assert.Equal(t, 10*500000, operation_setting.EpayTopupAmountToQuota(72))

	// 开关关闭 + CNY 展示：面额为美元，充 $10 到账 $10 额度
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeCNY, 7.3, 7.2, false)
	assert.Equal(t, 10*500000, operation_setting.EpayTopupAmountToQuota(10))

	// USD 展示：面额为美元，充 $10 到账 $10 额度
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeUSD, 7.3, 7.2, true)
	assert.Equal(t, 10*500000, operation_setting.EpayTopupAmountToQuota(10))
}

func TestEpayTopupAmountToQuotaInvalidRate(t *testing.T) {
	// 汇率非法时按 1:1 兜底入账，不能因除零崩溃
	setupEpayDisplayState(t, operation_setting.QuotaDisplayTypeCNY, 7.3, 0, true)
	assert.Equal(t, 10*500000, operation_setting.EpayTopupAmountToQuota(10))
}

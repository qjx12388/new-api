package operation_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/shopspring/decimal"
)

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠

	// EpayDisplayCurrencyAmountEnabled 为 true 且站点以 CNY 展示时，
	// 易支付充值面额（amount_options / min_topup / 自定义金额）按人民币解释并 1:1 结算；
	// 为 false 时保持旧行为：面额为美元，实收按 Price（本币/美元）换算。
	EpayDisplayCurrencyAmountEnabled bool `json:"epay_display_currency_amount_enabled"`

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}

// EpayDisplayCurrencyAmountEnabled 是否将易支付充值面额按展示货币（人民币）结算。
// 需要开关打开且站点以 CNY 展示才生效。
func EpayDisplayCurrencyAmountEnabled() bool {
	return paymentSetting.EpayDisplayCurrencyAmountEnabled && IsCNYDisplay()
}

// EpayPricePerUnit 返回易支付每单位充值金额对应的实收货币数量。
// 面额按展示货币结算时（CNY），人民币 1:1 收取，不再乘以 Price（美元单价）。
func EpayPricePerUnit() float64 {
	if EpayDisplayCurrencyAmountEnabled() {
		return 1
	}
	return Price
}

// EpayTopupAmountToQuota 将易支付充值金额换算为系统额度。
// 面额按展示货币结算时（CNY），金额按 USDExchangeRate 折算成美元额度；
// 否则金额视为美元（TOKENS 展示下单时已折算），直接乘以 QuotaPerUnit。
func EpayTopupAmountToQuota(amount int64) int {
	dAmount := decimal.NewFromInt(amount)
	if EpayDisplayCurrencyAmountEnabled() {
		if USDExchangeRate <= 0 {
			common.SysError(fmt.Sprintf("USDExchangeRate 非法（%v），CNY 充值金额 %d 无法折算，按 1:1 入账", USDExchangeRate, amount))
		} else {
			dAmount = dAmount.Div(decimal.NewFromFloat(USDExchangeRate))
		}
	}
	return int(dAmount.Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart())
}

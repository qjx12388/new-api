/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { z } from 'zod'

import type { PromotionRule, PromotionRuleFormData } from '../types'

type TFunction = (key: string) => string

// ============================================================================
// Promotion Rule Form Schema
// ============================================================================

export function getPromotionRuleFormSchema(t: TFunction) {
  return z
    .object({
      from_group: z.string().min(1, t('Please select a source group')),
      to_group: z.string().min(1, t('Please select a target group')),
      min_paid_amount: z
        .number()
        .positive(t('Threshold amount must be greater than 0')),
      enabled: z.boolean(),
      remark: z.string(),
    })
    .refine((data) => data.from_group !== data.to_group, {
      message: t('Source and target groups must be different'),
      path: ['to_group'],
    })
}

export type PromotionRuleFormValues = z.infer<
  ReturnType<typeof getPromotionRuleFormSchema>
>

export const PROMOTION_RULE_FORM_DEFAULT_VALUES: PromotionRuleFormValues = {
  from_group: '',
  to_group: '',
  min_paid_amount: 0,
  enabled: true,
  remark: '',
}

export function transformRuleToFormDefaults(
  rule: PromotionRule
): PromotionRuleFormValues {
  return {
    from_group: rule.from_group,
    to_group: rule.to_group,
    min_paid_amount: rule.min_paid_amount,
    enabled: rule.enabled,
    remark: rule.remark,
  }
}

export function transformFormValuesToPayload(
  values: PromotionRuleFormValues
): PromotionRuleFormData {
  return {
    from_group: values.from_group,
    to_group: values.to_group,
    min_paid_amount: values.min_paid_amount,
    enabled: values.enabled,
    remark: values.remark.trim(),
  }
}

// ============================================================================
// Currency Display
// ============================================================================

const CURRENCY_SYMBOLS: Record<string, string> = {
  CNY: '¥',
  USD: '$',
}

export function getCurrencySymbol(currency: string): string {
  return CURRENCY_SYMBOLS[currency.toUpperCase()] ?? `${currency} `
}

export function formatMoney(value: number, currency: string): string {
  const symbol = getCurrencySymbol(currency)
  return `${symbol}${value.toFixed(2)}`
}

export function formatRatio(ratio: number | null): string {
  if (ratio == null) return '-'
  return `${(ratio * 100).toFixed(1)}%`
}

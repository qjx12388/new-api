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

// ============================================================================
// Promotion Rule & Log Types
// ============================================================================

export interface PromotionRule {
  id: number
  from_group: string
  to_group: string
  min_paid_amount: number
  enabled: boolean
  remark: string
  created_at: number
  updated_at: number
}

export interface PromotionLog {
  id: number
  user_id: number
  username: string
  from_group: string
  to_group: string
  paid_amount: number
  rule_id: number
  created_at: number
}

// ============================================================================
// Revenue Stats Types
// ============================================================================

export interface RevenueGroupStat {
  group: string
  paid: number
  consume: number
  ratio: number | null
}

export interface RevenueDayStat {
  day: string
  paid: number
  consume: number
}

export interface RevenueStats {
  currency: string
  start_ts: number
  end_ts: number
  groups: RevenueGroupStat[]
  days: RevenueDayStat[]
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export interface GetPromotionLogsParams {
  p?: number
  page_size?: number
}

export interface GetPromotionLogsResponse {
  success: boolean
  message?: string
  data?: {
    items: PromotionLog[]
    total: number
    page: number
    page_size: number
  }
}

export interface GetRevenueStatsParams {
  start_ts?: number
  end_ts?: number
}

export interface PromotionRuleFormData {
  id?: number
  from_group: string
  to_group: string
  min_paid_amount: number
  enabled: boolean
  remark: string
}

// ============================================================================
// Dialog Types
// ============================================================================

export type PromotionsDialogType = 'create' | 'update' | 'delete'

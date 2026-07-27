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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  GetPromotionLogsParams,
  GetPromotionLogsResponse,
  GetRevenueStatsParams,
  PromotionRule,
  PromotionRuleFormData,
  RevenueStats,
} from './types'

// ============================================================================
// Promotion Rule Management
// ============================================================================

// Get all promotion rules
export async function getPromotionRules(): Promise<
  ApiResponse<PromotionRule[]>
> {
  const res = await api.get('/api/promotion/rule')
  return res.data
}

// Create a promotion rule
export async function createPromotionRule(
  data: PromotionRuleFormData
): Promise<ApiResponse<PromotionRule>> {
  const res = await api.post('/api/promotion/rule', data)
  return res.data
}

// Update a promotion rule
export async function updatePromotionRule(
  data: PromotionRuleFormData & { id: number }
): Promise<ApiResponse<PromotionRule>> {
  const res = await api.put('/api/promotion/rule', data)
  return res.data
}

// Delete a promotion rule
export async function deletePromotionRule(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/promotion/rule/${id}`)
  return res.data
}

// ============================================================================
// Promotion Logs
// ============================================================================

// Get paginated promotion logs
export async function getPromotionLogs(
  params: GetPromotionLogsParams = {}
): Promise<GetPromotionLogsResponse> {
  const { p = 1, page_size = 10 } = params
  const res = await api.get(`/api/promotion/log?p=${p}&page_size=${page_size}`)
  return res.data
}

// ============================================================================
// Revenue Stats
// ============================================================================

// Get revenue stats (defaults to current month when params are omitted)
export async function getRevenueStats(
  params: GetRevenueStatsParams = {}
): Promise<ApiResponse<RevenueStats>> {
  const queryParams = new URLSearchParams()
  if (params.start_ts != null) {
    queryParams.set('start_ts', String(params.start_ts))
  }
  if (params.end_ts != null) {
    queryParams.set('end_ts', String(params.end_ts))
  }
  const query = queryParams.toString()
  const res = await api.get(
    `/api/promotion/revenue_stats${query ? `?${query}` : ''}`
  )
  return res.data
}

// ============================================================================
// Groups
// ============================================================================

// Get all user groups
export async function getGroups(): Promise<ApiResponse<string[]>> {
  const res = await api.get('/api/group/')
  return res.data
}

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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'

import type { PromotionRule } from '../types'
import { PromotionRuleEnabledSwitch } from './promotion-rule-enabled-switch'
import { PromotionRulesRowActions } from './promotion-rules-row-actions'

export function usePromotionRulesColumns(): ColumnDef<PromotionRule>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'from_group',
      header: t('Source Group'),
      meta: { mobileTitle: true },
      cell: ({ row }) => (
        <StatusBadge
          label={row.getValue('from_group') as string}
          variant='neutral'
          copyable={false}
          className='-ml-1.5'
        />
      ),
      size: 140,
    },
    {
      accessorKey: 'to_group',
      header: t('Target Group'),
      cell: ({ row }) => (
        <StatusBadge
          label={row.getValue('to_group') as string}
          variant='success'
          copyable={false}
          className='-ml-1.5'
        />
      ),
      size: 140,
    },
    {
      accessorKey: 'min_paid_amount',
      header: t('Threshold Amount'),
      cell: ({ row }) => {
        const amount = row.getValue('min_paid_amount') as number
        return <span className='font-mono text-sm'>{amount.toFixed(2)}</span>
      },
      size: 140,
    },
    {
      accessorKey: 'enabled',
      header: t('Enabled'),
      meta: { mobileBadge: true },
      cell: ({ row }) => <PromotionRuleEnabledSwitch rule={row.original} />,
      size: 100,
    },
    {
      accessorKey: 'remark',
      header: t('Remark'),
      meta: { mobileHidden: true },
      cell: ({ row }) => {
        const remark = row.getValue('remark') as string
        if (!remark) {
          return <span className='text-muted-foreground text-sm'>-</span>
        }
        return <span className='text-sm'>{remark}</span>
      },
      size: 220,
    },
    {
      id: 'actions',
      header: () => t('Actions'),
      cell: ({ row }) => <PromotionRulesRowActions row={row} />,
      meta: { pinned: 'right' as const },
    },
  ]
}

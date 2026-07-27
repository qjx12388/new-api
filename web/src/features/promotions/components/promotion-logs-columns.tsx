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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { formatTimestampToDate } from '@/lib/format'

import type { PromotionLog } from '../types'

export function usePromotionLogsColumns(): ColumnDef<PromotionLog>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'created_at',
      header: t('Time'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <div className='min-w-[160px] font-mono text-sm'>
          {formatTimestampToDate(row.getValue('created_at'))}
        </div>
      ),
      size: 180,
    },
    {
      accessorKey: 'username',
      header: t('User'),
      meta: { mobileTitle: true },
      cell: ({ row }) => {
        const log = row.original
        return (
          <span className='font-medium'>
            {log.username || t('User {{id}}', { id: log.user_id })}
          </span>
        )
      },
      size: 160,
    },
    {
      id: 'promotion',
      header: t('Promotion'),
      meta: { mobileBadge: true },
      cell: ({ row }) => {
        const log = row.original
        return (
          <div className='flex items-center gap-1'>
            <StatusBadge
              label={log.from_group}
              variant='neutral'
              copyable={false}
            />
            <ArrowRight className='text-muted-foreground h-3.5 w-3.5' />
            <StatusBadge
              label={log.to_group}
              variant='success'
              copyable={false}
            />
          </div>
        )
      },
      size: 220,
    },
    {
      accessorKey: 'paid_amount',
      header: t('Total Paid'),
      cell: ({ row }) => {
        const amount = row.getValue('paid_amount') as number
        return <span className='font-mono text-sm'>{amount.toFixed(2)}</span>
      },
      size: 120,
    },
    {
      accessorKey: 'rule_id',
      header: t('Rule ID'),
      meta: { mobileHidden: true },
      cell: ({ row }) => (
        <TableId
          value={row.getValue('rule_id') as number}
          className='w-[60px]'
        />
      ),
      size: 100,
    },
  ]
}

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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DataTablePage, useDataTable } from '@/components/data-table'

import { getPromotionRules } from '../api'
import { usePromotionRulesColumns } from './promotion-rules-columns'
import { usePromotions } from './promotions-provider'

export function PromotionRulesTable() {
  const { t } = useTranslation()
  const columns = usePromotionRulesColumns()
  const { refreshTrigger } = usePromotions()

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['promotion-rules', refreshTrigger],
    queryFn: async () => {
      const result = await getPromotionRules()
      if (!result.success) {
        toast.error(result.message || t('Failed to load promotion rules'))
        return []
      }
      return result.data || []
    },
    placeholderData: (previousData) => previousData,
  })

  const { table } = useDataTable({
    data: data || [],
    columns,
  })

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Promotion Rules Found')}
      emptyDescription={t(
        'No promotion rules available. Create your first rule to get started.'
      )}
      skeletonKeyPrefix='promotion-rules-skeleton'
      applyHeaderSize
      toolbarProps={null}
      showPagination={false}
      fixedHeight={false}
    />
  )
}

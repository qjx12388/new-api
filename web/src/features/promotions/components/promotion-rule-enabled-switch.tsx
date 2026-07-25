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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Switch } from '@/components/ui/switch'

import { updatePromotionRule } from '../api'
import type { PromotionRule } from '../types'

export function PromotionRuleEnabledSwitch({ rule }: { rule: PromotionRule }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (enabled: boolean) => updatePromotionRule({ ...rule, enabled }),
    onSuccess: (result) => {
      if (result.success) {
        queryClient.invalidateQueries({ queryKey: ['promotion-rules'] })
      } else {
        toast.error(result.message || t('Failed to update promotion rule'))
      }
    },
    onError: () => {
      toast.error(t('Failed to update promotion rule'))
    },
  })

  return (
    <Switch
      checked={rule.enabled}
      disabled={mutation.isPending}
      onCheckedChange={(checked) => mutation.mutate(checked)}
      aria-label={t('Enabled')}
    />
  )
}

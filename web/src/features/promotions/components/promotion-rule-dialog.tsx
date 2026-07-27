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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import { createPromotionRule, getGroups, updatePromotionRule } from '../api'
import {
  getPromotionRuleFormSchema,
  PROMOTION_RULE_FORM_DEFAULT_VALUES,
  transformFormValuesToPayload,
  transformRuleToFormDefaults,
  type PromotionRuleFormValues,
} from '../lib'
import type { PromotionRule } from '../types'
import { usePromotions } from './promotions-provider'

type PromotionRuleDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  currentRow?: PromotionRule
}

export function PromotionRuleDialog({
  open,
  onOpenChange,
  currentRow,
}: PromotionRuleDialogProps) {
  const { t } = useTranslation()
  const isUpdate = !!currentRow
  const { triggerRefresh } = usePromotions()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const { data: groupsData } = useQuery({
    queryKey: ['groups'],
    queryFn: getGroups,
  })
  const groups = groupsData?.data || []

  const form = useForm<PromotionRuleFormValues>({
    resolver: zodResolver(getPromotionRuleFormSchema(t)),
    defaultValues: PROMOTION_RULE_FORM_DEFAULT_VALUES,
  })

  useEffect(() => {
    if (!open) return
    if (isUpdate && currentRow) {
      form.reset(transformRuleToFormDefaults(currentRow))
    } else {
      form.reset(PROMOTION_RULE_FORM_DEFAULT_VALUES)
    }
  }, [open, isUpdate, currentRow, form])

  const onSubmit = async (values: PromotionRuleFormValues) => {
    setIsSubmitting(true)
    try {
      const payload = transformFormValuesToPayload(values)
      const result =
        isUpdate && currentRow
          ? await updatePromotionRule({ ...payload, id: currentRow.id })
          : await createPromotionRule(payload)

      if (result.success) {
        toast.success(
          isUpdate
            ? t('Promotion rule updated successfully')
            : t('Promotion rule created successfully')
        )
        onOpenChange(false)
        triggerRefresh()
      } else {
        toast.error(result.message || t('Failed to save promotion rule'))
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const groupItems = groups.map((group) => ({ value: group, label: group }))

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v)
        if (!v) form.reset()
      }}
    >
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle>
            {isUpdate ? t('Edit Promotion Rule') : t('Create Promotion Rule')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'Promote a user to the target group once the paid amount reaches the threshold.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            id='promotion-rule-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className='flex flex-col gap-4'
          >
            <FormField
              control={form.control}
              name='from_group'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Source Group')}</FormLabel>
                  <Select
                    items={groupItems}
                    onValueChange={field.onChange}
                    value={field.value}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Select a group')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {groups.map((group) => (
                          <SelectItem key={group} value={group}>
                            {group}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='to_group'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Target Group')}</FormLabel>
                  <Select
                    items={groupItems}
                    onValueChange={field.onChange}
                    value={field.value}
                  >
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('Select a group')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {groups.map((group) => (
                          <SelectItem key={group} value={group}>
                            {group}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='min_paid_amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Threshold Amount')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type='number'
                      step='0.01'
                      min='0'
                      placeholder={t('Enter the minimum paid amount')}
                      onChange={(e) =>
                        field.onChange(Number.parseFloat(e.target.value) || 0)
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='remark'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Remark')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder={t('Optional remark')} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='enabled'
              render={({ field }) => (
                <FormItem className='flex items-center justify-between gap-2'>
                  <FormLabel>{t('Enabled')}</FormLabel>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
          </form>
        </Form>
        <DialogFooter>
          <Button
            form='promotion-rule-form'
            type='submit'
            disabled={isSubmitting}
          >
            {isSubmitting ? t('Saving...') : t('Save changes')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

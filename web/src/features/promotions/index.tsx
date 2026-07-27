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
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'

import { PromotionLogsTable } from './components/promotion-logs-table'
import { PromotionRulesTable } from './components/promotion-rules-table'
import { PromotionsDialogs } from './components/promotions-dialogs'
import { PromotionsPrimaryButtons } from './components/promotions-primary-buttons'
import { PromotionsProvider } from './components/promotions-provider'
import { RevenueCharts } from './components/revenue-charts'

export function Promotions() {
  const { t } = useTranslation()
  return (
    <PromotionsProvider>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Promotions')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <PromotionsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex flex-col gap-4 sm:gap-5'>
            <RevenueCharts />

            <section className='flex flex-col gap-2.5 sm:gap-3'>
              <h3 className='text-sm font-semibold sm:text-base'>
                {t('Promotion Rules')}
              </h3>
              <PromotionRulesTable />
            </section>

            <section className='flex flex-col gap-2.5 sm:gap-3'>
              <h3 className='text-sm font-semibold sm:text-base'>
                {t('Promotion Logs')}
              </h3>
              <PromotionLogsTable />
            </section>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <PromotionsDialogs />
    </PromotionsProvider>
  )
}

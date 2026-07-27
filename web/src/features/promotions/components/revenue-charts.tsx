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
import { VChart } from '@visactor/react-vchart'
import { BarChart3, TrendingUp } from 'lucide-react'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { IconBadge } from '@/components/ui/icon-badge'
import { useTheme } from '@/context/theme-provider'
import { VCHART_OPTION } from '@/lib/vchart'

import { getRevenueStats } from '../api'
import { formatMoney, formatRatio } from '../lib'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type VChartSpec = Record<string, any>

function useVChartTheme() {
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)

      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }

      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }

    updateTheme()
  }, [resolvedTheme])

  return { resolvedTheme, themeReady }
}

interface ChartCardProps {
  icon: ReactNode
  title: string
  spec: VChartSpec
  chartKey: string
  themeReady: boolean
  resolvedTheme: string | undefined
}

function ChartCard(props: ChartCardProps) {
  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
        <IconBadge tone='chart-4' size='sm'>
          {props.icon}
        </IconBadge>
        <div className='text-sm font-semibold'>{props.title}</div>
      </div>
      <div className='h-[280px] p-1.5 sm:h-80 sm:p-2'>
        {props.themeReady && (
          <VChart
            key={props.chartKey}
            spec={{
              ...props.spec,
              theme: props.resolvedTheme === 'dark' ? 'dark' : 'light',
              background: 'transparent',
            }}
            option={VCHART_OPTION}
          />
        )}
      </div>
    </div>
  )
}

export function RevenueCharts() {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useVChartTheme()

  const { data, isLoading } = useQuery({
    queryKey: ['promotion-revenue-stats'],
    queryFn: async () => {
      const result = await getRevenueStats()
      if (!result.success) {
        toast.error(result.message || t('Failed to load revenue stats'))
        return undefined
      }
      return result.data
    },
  })

  const currency = data?.currency ?? 'CNY'
  const paidLabel = t('Paid Revenue')
  const consumeLabel = t('Consumption')

  const groupSpec = useMemo(() => {
    const values = (data?.groups ?? []).flatMap((stat) => [
      {
        Group: stat.group,
        Metric: paidLabel,
        Amount: stat.paid,
        ratio: stat.ratio,
      },
      {
        Group: stat.group,
        Metric: consumeLabel,
        Amount: stat.consume,
        ratio: stat.ratio,
      },
    ])

    return {
      type: 'bar',
      data: [{ id: 'groupData', values }],
      xField: 'Group',
      yField: 'Amount',
      seriesField: 'Metric',
      legends: { visible: true },
      bar: {
        state: {
          hover: { stroke: '#000', lineWidth: 1 },
        },
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: Record<string, unknown>) => datum?.Metric,
              value: (datum: Record<string, unknown>) =>
                formatMoney(Number(datum?.Amount) || 0, currency),
            },
            {
              key: () => t('Consumption Ratio'),
              value: (datum: Record<string, unknown>) =>
                formatRatio((datum?.ratio as number | null) ?? null),
            },
          ],
        },
      },
      title: {
        visible: true,
        text: t('Group Revenue Comparison'),
        subtext: isLoading ? t('Loading...') : undefined,
      },
      animation: true,
    }
  }, [data, isLoading, currency, paidLabel, consumeLabel, t])

  const daySpec = useMemo(() => {
    const values = (data?.days ?? []).flatMap((stat) => [
      { Day: stat.day, Metric: paidLabel, Amount: stat.paid },
      { Day: stat.day, Metric: consumeLabel, Amount: stat.consume },
    ])

    return {
      type: 'line',
      data: [{ id: 'dayData', values }],
      xField: 'Day',
      yField: 'Amount',
      seriesField: 'Metric',
      legends: { visible: true, selectMode: 'single' },
      point: { visible: false },
      line: {
        style: {
          lineWidth: 2,
          curveType: 'monotone',
        },
      },
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: Record<string, unknown>) => datum?.Metric,
              value: (datum: Record<string, unknown>) =>
                formatMoney(Number(datum?.Amount) || 0, currency),
            },
          ],
        },
      },
      title: {
        visible: true,
        text: t('Daily Revenue Trend'),
        subtext: isLoading ? t('Loading...') : undefined,
      },
      animation: true,
    }
  }, [data, isLoading, currency, paidLabel, consumeLabel, t])

  const chartKey = [
    resolvedTheme,
    isLoading ? 'loading' : 'ready',
    data?.groups?.length ?? 0,
    data?.days?.length ?? 0,
  ].join('-')

  return (
    <div className='grid gap-3 lg:grid-cols-2'>
      <ChartCard
        icon={<BarChart3 />}
        title={t('Group Revenue Comparison')}
        spec={groupSpec}
        chartKey={`group-${chartKey}`}
        themeReady={themeReady}
        resolvedTheme={resolvedTheme}
      />
      <ChartCard
        icon={<TrendingUp />}
        title={t('Daily Revenue Trend')}
        spec={daySpec}
        chartKey={`day-${chartKey}`}
        themeReady={themeReady}
        resolvedTheme={resolvedTheme}
      />
    </div>
  )
}

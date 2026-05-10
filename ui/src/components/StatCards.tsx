'use client'

import { DashboardStats } from '@/actions/statistics'
import { AnimatedNumber } from './AnimatedNumber'
import { Shield, Search, Bell, TrendingUp, AlertTriangle, Clock } from 'lucide-react'

interface StatCardsProps {
  stats: DashboardStats
}

export function StatCards({ stats }: StatCardsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label="Total Findings"
        value={stats.totalFindings}
        icon={Shield}
        color="text-red-600"
        bg="bg-red-50"
      />
      <StatCard
        label="Repos Scanned"
        value={stats.totalReposScanned}
        icon={Search}
        color="text-primary-600"
        bg="bg-primary-50"
      />
      <StatCard
        label="Files Scanned"
        value={stats.totalFilesScanned}
        icon={Search}
        color="text-primary-600"
        bg="bg-primary-50"
      />
      <StatCard
        label="Notification Rate"
        value={stats.notificationRate}
        suffix="%"
        decimals={1}
        icon={Bell}
        color="text-green-600"
        bg="bg-green-50"
      />
      <StatCard
        label="Avg Confidence"
        value={stats.avgConfidence}
        suffix="%"
        decimals={1}
        icon={TrendingUp}
        color="text-orange-600"
        bg="bg-orange-50"
      />
      <StatCard
        label="Total Errors"
        value={stats.totalErrors}
        icon={AlertTriangle}
        color="text-yellow-600"
        bg="bg-yellow-50"
      />
      <StatCard
        label="Avg Scan Duration"
        value={stats.avgScanDurationMs / 1000}
        suffix="s"
        decimals={1}
        icon={Clock}
        color="text-blue-600"
        bg="bg-blue-50"
      />
    </div>
  )
}

interface StatCardProps {
  label: string
  value: number
  suffix?: string
  decimals?: number
  icon: React.ElementType
  color: string
  bg: string
}

function StatCard({ label, value, suffix, decimals = 0, icon: Icon, color, bg }: StatCardProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-500">{label}</p>
          <p className="mt-1 text-2xl font-bold text-gray-900">
            <AnimatedNumber
              value={value}
              format={(n) =>
                `${decimals > 0 ? n.toFixed(decimals) : Math.round(n).toLocaleString()}${suffix || ''}`
              }
            />
          </p>
        </div>
        <div className={`rounded-lg p-2 ${bg}`}>
          <Icon size={20} className={color} />
        </div>
      </div>
    </div>
  )
}

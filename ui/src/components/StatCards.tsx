'use client'

import { DashboardStats } from '@/actions/statistics'
import { Shield, Search, Bell, TrendingUp, AlertTriangle, Clock } from 'lucide-react'

interface StatCardsProps {
  stats: DashboardStats
}

export function StatCards({ stats }: StatCardsProps) {
  const cards = [
    {
      label: 'Total Findings',
      value: stats.totalFindings.toLocaleString(),
      icon: Shield,
      color: 'text-red-600',
      bg: 'bg-red-50',
    },
    {
      label: 'Repos Scanned',
      value: stats.totalReposScanned.toLocaleString(),
      icon: Search,
      color: 'text-primary-600',
      bg: 'bg-primary-50',
    },
    {
      label: 'Files Scanned',
      value: stats.totalFilesScanned.toLocaleString(),
      icon: Search,
      color: 'text-primary-600',
      bg: 'bg-primary-50',
    },
    {
      label: 'Notification Rate',
      value: `${stats.notificationRate.toFixed(1)}%`,
      icon: Bell,
      color: 'text-green-600',
      bg: 'bg-green-50',
    },
    {
      label: 'Avg Confidence',
      value: `${stats.avgConfidence.toFixed(1)}%`,
      icon: TrendingUp,
      color: 'text-orange-600',
      bg: 'bg-orange-50',
    },
    {
      label: 'Total Scans',
      value: stats.totalScans.toLocaleString(),
      icon: Clock,
      color: 'text-purple-600',
      bg: 'bg-purple-50',
    },
    {
      label: 'Total Errors',
      value: stats.totalErrors.toLocaleString(),
      icon: AlertTriangle,
      color: 'text-yellow-600',
      bg: 'bg-yellow-50',
    },
    {
      label: 'Avg Scan Duration',
      value: `${(stats.avgScanDurationMs / 1000).toFixed(1)}s`,
      icon: Clock,
      color: 'text-blue-600',
      bg: 'bg-blue-50',
    },
  ]

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <div
            key={card.label}
            className="rounded-lg border border-gray-200 bg-white p-4"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">{card.label}</p>
                <p className="mt-1 text-2xl font-bold text-gray-900">{card.value}</p>
              </div>
              <div className={`rounded-lg p-2 ${card.bg}`}>
                <Icon size={20} className={card.color} />
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}

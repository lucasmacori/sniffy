'use client'

import { useQuery } from '@tanstack/react-query'
import { getDashboardStats } from '@/actions/statistics'
import { getFindingsOverTime, getSecretTypeDistribution, getTopRepositories, getConfidenceDistribution, getSourceDistribution, getNotificationStatus } from '@/actions/findings'
import { getScanPerformance } from '@/actions/statistics'
import { StatCards } from '@/components/StatCards'
import {
  FindingsOverTimeChart,
  SecretTypeChart,
  TopReposChart,
  ConfidenceChart,
  SourceChart,
  NotificationChart,
  ScanPerformanceChart,
} from '@/components/Charts'
import { AutoRefresh } from '@/components/AutoRefresh'
import { BarChart3, RefreshCw } from 'lucide-react'
import { useCallback } from 'react'

export default function StatsPage() {
  const { data: stats, isLoading: statsLoading, refetch: refetchStats, isFetching } = useQuery({
    queryKey: ['dashboardStats'],
    queryFn: getDashboardStats,
  })

  const { data: findingsOverTime, refetch: refetchTime } = useQuery({
    queryKey: ['findingsOverTime'],
    queryFn: () => getFindingsOverTime(30),
  })

  const { data: secretTypes, refetch: refetchTypes } = useQuery({
    queryKey: ['secretTypes'],
    queryFn: getSecretTypeDistribution,
  })

  const { data: topRepos, refetch: refetchRepos } = useQuery({
    queryKey: ['topRepos'],
    queryFn: () => getTopRepositories(10),
  })

  const { data: confidenceDist, refetch: refetchConfidence } = useQuery({
    queryKey: ['confidenceDist'],
    queryFn: getConfidenceDistribution,
  })

  const { data: sourceDist, refetch: refetchSource } = useQuery({
    queryKey: ['sourceDist'],
    queryFn: getSourceDistribution,
  })

  const { data: notificationStatus, refetch: refetchNotif } = useQuery({
    queryKey: ['notificationStatus'],
    queryFn: getNotificationStatus,
  })

  const { data: scanPerformance, refetch: refetchPerf } = useQuery({
    queryKey: ['scanPerformance'],
    queryFn: () => getScanPerformance(30),
  })

  const handleRefresh = useCallback(() => {
    refetchStats()
    refetchTime()
    refetchTypes()
    refetchRepos()
    refetchConfidence()
    refetchSource()
    refetchNotif()
    refetchPerf()
  }, [refetchStats, refetchTime, refetchTypes, refetchRepos, refetchConfidence, refetchSource, refetchNotif, refetchPerf])

  if (statsLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="text-gray-500">Loading statistics...</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 size={24} className="text-primary-600" />
          <h1 className="text-2xl font-bold text-gray-900">Statistics</h1>
        </div>
        <div className="flex items-center gap-4">
          <AutoRefresh onRefresh={handleRefresh} isRefreshing={isFetching} />
          <button
            onClick={handleRefresh}
            className="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            type="button"
          >
            <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} />
            Refresh
          </button>
        </div>
      </div>

      <StatCards stats={stats!} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Findings Over Time</h2>
          <FindingsOverTimeChart data={findingsOverTime || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Secret Type Breakdown</h2>
          <SecretTypeChart data={secretTypes || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Top Leaky Repositories</h2>
          <TopReposChart data={topRepos || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Confidence Distribution</h2>
          <ConfidenceChart data={confidenceDist || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Source Breakdown</h2>
          <SourceChart data={sourceDist || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Notification Status</h2>
          <NotificationChart data={notificationStatus || []} />
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4 lg:col-span-2">
          <h2 className="mb-4 text-lg font-semibold text-gray-900">Scan Performance</h2>
          <ScanPerformanceChart data={scanPerformance || []} />
        </div>
      </div>
    </div>
  )
}

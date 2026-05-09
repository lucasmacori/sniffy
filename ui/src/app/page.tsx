'use client'

import { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getFindings, getSecretTypes, getSources } from '@/actions/findings'
import { FindingTable } from '@/components/FindingTable'
import { FilterBar } from '@/components/FilterBar'
import { AutoRefresh } from '@/components/AutoRefresh'
import { Shield, RefreshCw } from 'lucide-react'

const defaultFilters = {
  secretType: '',
  source: '',
  notified: '',
  minConfidence: '',
  maxConfidence: '',
  repository: '',
  dateFrom: '',
  dateTo: '',
  sortBy: 'created_at',
  sortOrder: 'desc' as 'asc' | 'desc',
}

export default function FindingsPage() {
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState(defaultFilters)

  const { data: secretTypes } = useQuery({
    queryKey: ['secretTypes'],
    queryFn: getSecretTypes,
  })

  const { data: sources } = useQuery({
    queryKey: ['sources'],
    queryFn: getSources,
  })

  const { data, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['findings', page, filters],
    queryFn: () =>
      getFindings({
        secretType: filters.secretType || undefined,
        source: filters.source || undefined,
        notified: filters.notified ? filters.notified === 'true' : undefined,
        minConfidence: filters.minConfidence ? parseFloat(filters.minConfidence) : undefined,
        maxConfidence: filters.maxConfidence ? parseFloat(filters.maxConfidence) : undefined,
        repository: filters.repository || undefined,
        dateFrom: filters.dateFrom || undefined,
        dateTo: filters.dateTo || undefined,
        sortBy: filters.sortBy,
        sortOrder: (filters.sortOrder as 'asc' | 'desc') || 'desc',
        page,
        perPage: 25,
      }),
  })

  const handleRefresh = useCallback(() => {
    refetch()
  }, [refetch])

  const handleFilterChange = (newFilters: typeof filters) => {
    setFilters(newFilters)
    setPage(1)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Shield size={24} className="text-primary-600" />
          <h1 className="text-2xl font-bold text-gray-900">Credential Findings</h1>
        </div>
        <div className="flex items-center gap-4">
          <AutoRefresh onRefresh={handleRefresh} isRefreshing={isFetching} />
          <button
            onClick={() => refetch()}
            className="flex items-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
            type="button"
          >
            <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} />
            Refresh
          </button>
        </div>
      </div>

      <FilterBar
        secretTypes={secretTypes || []}
        sources={sources || []}
        filters={filters}
        onFilterChange={handleFilterChange}
      />

      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <div className="text-gray-500">Loading findings...</div>
        </div>
      ) : (
        <FindingTable
          findings={data?.findings || []}
          total={data?.total || 0}
          page={page}
          perPage={25}
          onPageChange={setPage}
        />
      )}
    </div>
  )
}

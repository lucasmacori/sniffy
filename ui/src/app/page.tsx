'use client'

import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getFindings, getSecretTypes, getSources, markNotified } from '@/actions/findings'
import { FindingTable } from '@/components/FindingTable'
import { FilterBar } from '@/components/FilterBar'
import { AutoRefresh } from '@/components/AutoRefresh'
import { Shield, RefreshCw, AlertTriangle } from 'lucide-react'

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
  const [perPage, setPerPage] = useState(25)
  const [filters, setFilters] = useState(defaultFilters)
  const queryClient = useQueryClient()

  const markNotifiedMutation = useMutation({
    mutationFn: markNotified,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['findings'] })
    },
  })

  const { data: secretTypes, refetch: refetchSecretTypes } = useQuery({
    queryKey: ['secretTypes'],
    queryFn: getSecretTypes,
  })

  const { data: sources, refetch: refetchSources } = useQuery({
    queryKey: ['sources'],
    queryFn: getSources,
  })

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ['findings', page, perPage, filters],
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
        perPage,
      }),
  })

  const handleRefresh = useCallback(() => {
    refetch()
    refetchSecretTypes()
    refetchSources()
  }, [refetch, refetchSecretTypes, refetchSources])

  const handleFilterChange = (newFilters: typeof filters) => {
    setFilters(newFilters)
    setPage(1)
  }

  const handlePerPageChange = (newPerPage: number) => {
    setPerPage(newPerPage)
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
            className="flex items-center gap-2 rounded-md cursor-pointer border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
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

      {error ? (
        <div className="flex h-64 items-center justify-center rounded-lg border border-red-200 bg-red-50">
          <div className="flex items-center gap-2 text-red-700">
            <AlertTriangle size={20} />
            <span>Failed to load findings</span>
          </div>
        </div>
      ) : isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <div className="text-gray-500">Loading findings...</div>
        </div>
      ) : (
        <FindingTable
          findings={data?.findings || []}
          total={data?.total || 0}
          page={page}
          perPage={perPage}
          onPageChange={setPage}
          onPerPageChange={handlePerPageChange}
          onMarkNotified={(id) => markNotifiedMutation.mutate(id)}
        />
      )}
    </div>
  )
}

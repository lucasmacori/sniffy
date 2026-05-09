'use client'

import { Finding } from '@/actions/findings'
import { FindingTable } from './FindingTable'
import { useQuery } from '@tanstack/react-query'

interface FilterBarProps {
  secretTypes: string[]
  sources: string[]
  filters: {
    secretType: string
    source: string
    notified: string
    minConfidence: string
    maxConfidence: string
    repository: string
    dateFrom: string
    dateTo: string
    sortBy: string
    sortOrder: 'asc' | 'desc'
  }
  onFilterChange: (filters: FilterBarProps['filters']) => void
}

export function FilterBar({ secretTypes, sources, filters, onFilterChange }: FilterBarProps) {
  const updateFilter = (key: keyof FilterBarProps['filters'], value: string) => {
    onFilterChange({ ...filters, [key]: value } as FilterBarProps['filters'])
  }

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Secret Type
          </label>
          <select
            value={filters.secretType}
            onChange={(e) => updateFilter('secretType', e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          >
            <option value="">All types</option>
            {secretTypes.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Source
          </label>
          <select
            value={filters.source}
            onChange={(e) => updateFilter('source', e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          >
            <option value="">All sources</option>
            {sources.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Notification Status
          </label>
          <select
            value={filters.notified}
            onChange={(e) => updateFilter('notified', e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          >
            <option value="">All</option>
            <option value="true">Notified</option>
            <option value="false">Pending</option>
          </select>
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Repository
          </label>
          <input
            type="text"
            value={filters.repository}
            onChange={(e) => updateFilter('repository', e.target.value)}
            placeholder="Search repository..."
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Min Confidence
          </label>
          <input
            type="number"
            min="0"
            max="100"
            value={filters.minConfidence}
            onChange={(e) => updateFilter('minConfidence', e.target.value)}
            placeholder="0"
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            Max Confidence
          </label>
          <input
            type="number"
            min="0"
            max="100"
            value={filters.maxConfidence}
            onChange={(e) => updateFilter('maxConfidence', e.target.value)}
            placeholder="100"
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            From Date
          </label>
          <input
            type="date"
            value={filters.dateFrom}
            onChange={(e) => updateFilter('dateFrom', e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>

        <div>
          <label className="mb-1 block text-xs font-semibold text-gray-500 uppercase">
            To Date
          </label>
          <input
            type="date"
            value={filters.dateTo}
            onChange={(e) => updateFilter('dateTo', e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>
      </div>

      <div className="mt-4 flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label className="text-xs font-semibold text-gray-500 uppercase">Sort by</label>
          <select
            value={filters.sortBy}
            onChange={(e) => updateFilter('sortBy', e.target.value)}
            className="rounded-md border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="created_at">Date</option>
            <option value="confidence">Confidence</option>
            <option value="repository">Repository</option>
            <option value="secret_type">Type</option>
          </select>
          <select
            value={filters.sortOrder}
            onChange={(e) => updateFilter('sortOrder', e.target.value as 'asc' | 'desc')}
            className="rounded-md border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
        </div>
      </div>
    </div>
  )
}

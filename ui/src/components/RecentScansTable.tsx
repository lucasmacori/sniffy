'use client'

import { useState } from 'react'
import { ScanStat } from '@/actions/statistics'
import { Clock, Server, FileText, GitCommit, AlertTriangle, CheckCircle, XCircle } from 'lucide-react'

interface RecentScansTableProps {
  scans: ScanStat[]
}

export function RecentScansTable({ scans }: RecentScansTableProps) {
  const [page, setPage] = useState(1)
  const perPage = 10
  const totalPages = Math.ceil(scans.length / perPage)

  const paginatedScans = scans.slice((page - 1) * perPage, page * perPage)

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
    return `${(ms / 60000).toFixed(1)}m`
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-lg border border-gray-200">
        <table className="w-full text-left text-sm">
          <thead className="bg-gray-50 text-gray-700">
            <tr>
              <th className="px-4 py-3 font-semibold">Worker</th>
              <th className="px-4 py-3 font-semibold">Date</th>
              <th className="px-4 py-3 font-semibold">Repos</th>
              <th className="px-4 py-3 font-semibold">Files</th>
              <th className="px-4 py-3 font-semibold">Commits</th>
              <th className="px-4 py-3 font-semibold">Findings</th>
              <th className="px-4 py-3 font-semibold">Notified</th>
              <th className="px-4 py-3 font-semibold">Errors</th>
              <th className="px-4 py-3 font-semibold">Duration</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {paginatedScans.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-4 py-8 text-center text-gray-500">
                  No scan records yet
                </td>
              </tr>
            ) : (
              paginatedScans.map((scan) => (
                <tr key={scan.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-gray-700">
                    {scan.worker_id}
                  </td>
                  <td className="px-4 py-3 text-gray-600 text-xs">
                    {new Date(scan.scan_date).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-gray-700">
                      <Server size={12} />
                      {scan.repositories_scanned}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-gray-700">
                      <FileText size={12} />
                      {scan.files_scanned}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-gray-700">
                      <GitCommit size={12} />
                      {scan.commits_scanned}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-primary-700 font-medium">
                      <AlertTriangle size={12} />
                      {scan.findings_detected}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-green-600">
                      <CheckCircle size={12} />
                      {scan.findings_notified}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center gap-1 ${scan.errors_encountered > 0 ? 'text-red-600' : 'text-gray-500'}`}>
                      <XCircle size={12} />
                      {scan.errors_encountered}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex items-center gap-1 text-gray-500 text-xs">
                      <Clock size={12} />
                      {formatDuration(scan.scan_duration_ms)}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-gray-500">
            Showing {paginatedScans.length} of {scans.length} scans
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(page - 1)}
              disabled={page <= 1}
              className="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              type="button"
            >
              Previous
            </button>
            <span className="text-sm text-gray-600">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={page >= totalPages}
              className="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              type="button"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

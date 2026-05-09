'use client'

import React, { useState } from 'react'
import { Finding } from '@/actions/findings'
import { SecretReveal } from './SecretReveal'
import { ExternalLink, Bell, BellOff } from 'lucide-react'

interface FindingTableProps {
  findings: Finding[]
  total: number
  page: number
  perPage: number
  onPageChange: (page: number) => void
}

export function FindingTable({
  findings,
  total,
  page,
  perPage,
  onPageChange,
}: FindingTableProps) {
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const totalPages = Math.ceil(total / perPage)

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 75) return 'bg-red-100 text-red-800'
    if (confidence >= 50) return 'bg-orange-100 text-orange-800'
    if (confidence >= 25) return 'bg-yellow-100 text-yellow-800'
    return 'bg-gray-100 text-gray-800'
  }

  const getConfidenceLabel = (confidence: number) => {
    if (confidence >= 75) return 'High'
    if (confidence >= 50) return 'Medium'
    if (confidence >= 25) return 'Low'
    return 'Very Low'
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-lg border border-gray-200">
        <table className="w-full text-left text-sm">
          <thead className="bg-gray-50 text-gray-700">
            <tr>
              <th className="px-4 py-3 font-semibold">Repository</th>
              <th className="px-4 py-3 font-semibold">File</th>
              <th className="px-4 py-3 font-semibold">Line</th>
              <th className="px-4 py-3 font-semibold">Type</th>
              <th className="px-4 py-3 font-semibold">Confidence</th>
              <th className="px-4 py-3 font-semibold">Source</th>
              <th className="px-4 py-3 font-semibold">Discovered</th>
              <th className="px-4 py-3 font-semibold">Status</th>
              <th className="px-4 py-3 font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {findings.length === 0 ? (
              <tr>
                <td colSpan={9} className="px-4 py-8 text-center text-gray-500">
                  No findings match your filters
                </td>
              </tr>
            ) : (
              findings.map((finding) => (
                <React.Fragment key={finding.id}>
                  <tr
                    className="cursor-pointer hover:bg-gray-50 transition-colors"
                    onClick={() =>
                      setExpandedId(expandedId === finding.id ? null : finding.id)
                    }
                  >
                    <td className="px-4 py-3 font-medium text-gray-900">
                      {finding.repository}
                    </td>
                    <td className="px-4 py-3 text-gray-600 font-mono text-xs">
                      {finding.file_path}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{finding.line_number}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center rounded-full bg-primary-100 px-2.5 py-0.5 text-xs font-medium text-primary-800">
                        {finding.secret_type}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getConfidenceColor(
                          finding.confidence
                        )}`}
                      >
                        {finding.confidence.toFixed(1)}% ({getConfidenceLabel(finding.confidence)})
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-600 capitalize">{finding.source}</td>
                    <td className="px-4 py-3 text-gray-500 text-xs">
                      {new Date(finding.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3">
                      {finding.notified ? (
                        <span className="inline-flex items-center gap-1 text-green-600 text-xs">
                          <Bell size={14} /> Notified
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 text-gray-400 text-xs">
                          <BellOff size={14} /> Pending
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        {finding.html_url && (
                          <a
                            href={`${finding.html_url}/blob/HEAD/${finding.file_path}#L${finding.line_number}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary-600 hover:text-primary-800"
                            onClick={(e) => e.stopPropagation()}
                            title="View on GitHub"
                          >
                            <ExternalLink size={16} />
                          </a>
                        )}
                      </div>
                    </td>
                  </tr>
                  {expandedId === finding.id && (
                    <tr>
                      <td colSpan={9} className="bg-gray-50 px-4 py-4">
                        <div className="space-y-3">
                          <div>
                            <span className="text-xs font-semibold text-gray-500 uppercase">
                              Secret Value
                            </span>
                            <div className="mt-1">
                              <SecretReveal secret={finding.secret_value} />
                            </div>
                          </div>
                          {finding.commit_hash && (
                            <div className="grid grid-cols-3 gap-4 text-sm">
                              <div>
                                <span className="text-xs font-semibold text-gray-500 uppercase">
                                  Commit
                                </span>
                                <p className="font-mono text-gray-700">{finding.commit_hash}</p>
                              </div>
                              <div>
                                <span className="text-xs font-semibold text-gray-500 uppercase">
                                  Author
                                </span>
                                <p className="text-gray-700">{finding.commit_author}</p>
                              </div>
                              <div>
                                <span className="text-xs font-semibold text-gray-500 uppercase">
                                  Email
                                </span>
                                <p className="text-gray-700">{finding.commit_email}</p>
                              </div>
                            </div>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">
          Showing {(page - 1) * perPage + 1} to{' '}
          {Math.min(page * perPage, total)} of {total} findings
        </p>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onPageChange(page - 1)}
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
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            type="button"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  )
}

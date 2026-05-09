import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FindingTable } from '@/components/FindingTable'
import type { Finding } from '@/actions/findings'

const mockFindings: Finding[] = [
  {
    id: 1,
    repository: 'owner/repo1',
    commit_hash: 'abc123',
    commit_author: 'Alice',
    commit_email: 'alice@example.com',
    file_path: 'config.env',
    line_number: 5,
    secret_type: 'AWS Access Key',
    secret_value: 'AKIAIOSFODNN7EXAMPLE',
    confidence: 85,
    source: 'worktree',
    html_url: 'https://github.com/owner/repo1',
    notified: 1,
    created_at: '2024-01-15T10:30:00Z',
  },
  {
    id: 2,
    repository: 'owner/repo2',
    commit_hash: null,
    commit_author: null,
    commit_email: null,
    file_path: 'app.js',
    line_number: 42,
    secret_type: 'GitHub Token',
    secret_value: 'ghp_xxxxxxxxxxxxxxxxxxxx',
    confidence: 60,
    source: 'commit',
    html_url: 'https://github.com/owner/repo2',
    notified: 0,
    created_at: '2024-01-14T08:00:00Z',
  },
]

describe('FindingTable', () => {
  it('renders findings correctly', () => {
    render(
      <FindingTable
        findings={mockFindings}
        total={2}
        page={1}
        perPage={25}
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByText('owner/repo1')).toBeInTheDocument()
    expect(screen.getByText('owner/repo2')).toBeInTheDocument()
    expect(screen.getByText('config.env')).toBeInTheDocument()
    expect(screen.getByText('AWS Access Key')).toBeInTheDocument()
    expect(screen.getByText('GitHub Token')).toBeInTheDocument()
  })

  it('shows empty state when no findings', () => {
    render(
      <FindingTable
        findings={[]}
        total={0}
        page={1}
        perPage={25}
        onPageChange={vi.fn()}
      />
    )

    expect(screen.getByText('No findings match your filters')).toBeInTheDocument()
  })

  it('expands row on click', () => {
    render(
      <FindingTable
        findings={mockFindings}
        total={2}
        page={1}
        perPage={25}
        onPageChange={vi.fn()}
      />
    )

    const row = screen.getByText('owner/repo1').closest('tr')
    fireEvent.click(row!)

    // Secret should be masked initially even in expanded view
    expect(screen.getByText(/AKIAIOSF•••/)).toBeInTheDocument()
  })

  it('calls onPageChange when pagination button clicked', () => {
    const onPageChange = vi.fn()
    const findings = Array.from({ length: 30 }, (_, i) => ({
      ...mockFindings[0],
      id: i + 1,
      repository: `owner/repo${i + 1}`,
    }))

    render(
      <FindingTable
        findings={findings}
        total={30}
        page={1}
        perPage={25}
        onPageChange={onPageChange}
      />
    )

    const nextButton = screen.getByRole('button', { name: /next/i })
    fireEvent.click(nextButton)
    expect(onPageChange).toHaveBeenCalledWith(2)
  })
})

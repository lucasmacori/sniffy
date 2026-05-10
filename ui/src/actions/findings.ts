'use server'

import { getDb, getRwDb } from '@/lib/db'

export interface Finding {
  id: number
  repository: string
  commit_hash: string | null
  commit_author: string | null
  commit_email: string | null
  file_path: string
  line_number: number
  secret_type: string
  secret_value: string
  confidence: number
  source: string
  html_url: string | null
  notified: number
  created_at: string
}

export interface FindingsFilter {
  secretType?: string
  minConfidence?: number
  maxConfidence?: number
  repository?: string
  notified?: boolean
  source?: string
  dateFrom?: string
  dateTo?: string
  page?: number
  perPage?: number
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

export async function getFindings(filter: FindingsFilter = {}): Promise<{
  findings: Finding[]
  total: number
}> {
  const db = getDb()

  const conditions: string[] = []
  const params: (string | number)[] = []

  if (filter.secretType) {
    conditions.push('secret_type = ?')
    params.push(filter.secretType)
  }
  if (filter.minConfidence !== undefined) {
    conditions.push('confidence >= ?')
    params.push(filter.minConfidence)
  }
  if (filter.maxConfidence !== undefined) {
    conditions.push('confidence <= ?')
    params.push(filter.maxConfidence)
  }
  if (filter.repository) {
    conditions.push('repository LIKE ?')
    params.push(`%${filter.repository}%`)
  }
  if (filter.notified !== undefined) {
    conditions.push('notified = ?')
    params.push(filter.notified ? 1 : 0)
  }
  if (filter.source) {
    conditions.push('source = ?')
    params.push(filter.source)
  }
  if (filter.dateFrom) {
    conditions.push('created_at >= ?')
    params.push(filter.dateFrom)
  }
  if (filter.dateTo) {
    conditions.push('created_at <= ?')
    params.push(filter.dateTo)
  }

  const whereClause = conditions.length > 0 ? `WHERE ${conditions.join(' AND ')}` : ''

  // Get total count
  const countStmt = db.prepare(`SELECT COUNT(*) as total FROM findings ${whereClause}`)
  const { total } = countStmt.get(...params) as { total: number }

  // Get paginated results
  const page = filter.page || 1
  const perPage = filter.perPage || 25
  const offset = (page - 1) * perPage

  // Validate sortBy against allowlist to prevent SQL injection
  const ALLOWED_SORT_COLUMNS = ['created_at', 'confidence', 'repository', 'secret_type', 'file_path', 'line_number', 'id']
  const safeSortBy = ALLOWED_SORT_COLUMNS.includes(filter.sortBy || '') ? (filter.sortBy as string) : 'created_at'
  const safeSortOrder = filter.sortOrder === 'asc' ? 'ASC' : 'DESC'
  const orderClause = `ORDER BY ${safeSortBy} ${safeSortOrder}`

  const query = `SELECT * FROM findings ${whereClause} ${orderClause} LIMIT ? OFFSET ?`
  const stmt = db.prepare(query)
  const findings = stmt.all(...params, perPage, offset) as Finding[]

  return { findings, total }
}

export async function getFindingById(id: number): Promise<Finding | null> {
  const db = getDb()
  const stmt = db.prepare('SELECT * FROM findings WHERE id = ?')
  const result = stmt.get(id) as Finding | undefined
  return result || null
}

export async function getSecretTypes(): Promise<string[]> {
  const db = getDb()
  const stmt = db.prepare('SELECT DISTINCT secret_type FROM findings ORDER BY secret_type')
  const rows = stmt.all() as { secret_type: string }[]
  return rows.map(r => r.secret_type)
}

export async function getSources(): Promise<string[]> {
  const db = getDb()
  const stmt = db.prepare('SELECT DISTINCT source FROM findings ORDER BY source')
  const rows = stmt.all() as { source: string }[]
  return rows.map(r => r.source)
}

export async function markNotified(id: number): Promise<void> {
  const rwDb = getRwDb()
  const stmt = rwDb.prepare('UPDATE findings SET notified = 1 WHERE id = ?')
  stmt.run(id)
  rwDb.close()
}

export async function getTopRepositories(limit: number = 10): Promise<{ repository: string; count: number }[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT repository, COUNT(*) as count
    FROM findings
    GROUP BY repository
    ORDER BY count DESC
    LIMIT ?
  `)
  return stmt.all(limit) as { repository: string; count: number }[]
}

export async function getFindingsOverTime(days: number = 30): Promise<{ date: string; count: number }[]> {
  const db = getDb()
  // Use parameterized query to prevent SQL injection
  const stmt = db.prepare(`
    SELECT DATE(created_at) as date, COUNT(*) as count
    FROM findings
    WHERE created_at >= DATE('now', '-' || ? || ' days')
    GROUP BY DATE(created_at)
    ORDER BY date
  `)
  return stmt.all(days) as { date: string; count: number }[]
}

export async function getSecretTypeDistribution(): Promise<{ type: string; count: number }[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT secret_type as type, COUNT(*) as count
    FROM findings
    GROUP BY secret_type
    ORDER BY count DESC
  `)
  return stmt.all() as { type: string; count: number }[]
}

export async function getConfidenceDistribution(): Promise<{ range: string; count: number }[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT
      CASE
        WHEN confidence < 25 THEN '0-25'
        WHEN confidence < 50 THEN '25-50'
        WHEN confidence < 75 THEN '50-75'
        ELSE '75-100'
      END as range,
      COUNT(*) as count
    FROM findings
    GROUP BY range
    ORDER BY range
  `)
  return stmt.all() as { range: string; count: number }[]
}

export async function getSourceDistribution(): Promise<{ source: string; count: number }[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT source, COUNT(*) as count
    FROM findings
    GROUP BY source
    ORDER BY count DESC
  `)
  return stmt.all() as { source: string; count: number }[]
}

export async function getNotificationStatus(): Promise<{ status: string; count: number }[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT
      CASE WHEN notified = 1 THEN 'Notified' ELSE 'Pending' END as status,
      COUNT(*) as count
    FROM findings
    GROUP BY notified
  `)
  return stmt.all() as { status: string; count: number }[]
}

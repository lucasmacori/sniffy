'use server'

import { getDb } from '@/lib/db'

export interface DashboardStats {
  totalFindings: number
  totalReposScanned: number
  totalFilesScanned: number
  totalCommitsScanned: number
  avgConfidence: number
  notificationRate: number
  totalScans: number
  totalErrors: number
  avgScanDurationMs: number
}

export async function getDashboardStats(): Promise<DashboardStats> {
  const db = getDb()

  const findingsStmt = db.prepare('SELECT COUNT(*) as count FROM findings')
  const { count: totalFindings } = findingsStmt.get() as { count: number }

  const notifiedStmt = db.prepare('SELECT COUNT(*) as count FROM findings WHERE notified = 1')
  const { count: notifiedCount } = notifiedStmt.get() as { count: number }

  const confidenceStmt = db.prepare('SELECT AVG(confidence) as avg FROM findings')
  const { avg: avgConfidence } = confidenceStmt.get() as { avg: number }

  const statsStmt = db.prepare(`
    SELECT
      SUM(repositories_scanned) as totalRepos,
      SUM(files_scanned) as totalFiles,
      SUM(commits_scanned) as totalCommits,
      COUNT(*) as totalScans,
      SUM(errors_encountered) as totalErrors,
      AVG(scan_duration_ms) as avgDuration
    FROM statistics
  `)
  const stats = statsStmt.get() as {
    totalRepos: number
    totalFiles: number
    totalCommits: number
    totalScans: number
    totalErrors: number
    avgDuration: number
  }

  return {
    totalFindings,
    totalReposScanned: stats.totalRepos || 0,
    totalFilesScanned: stats.totalFiles || 0,
    totalCommitsScanned: stats.totalCommits || 0,
    avgConfidence: avgConfidence || 0,
    notificationRate: totalFindings > 0 ? (notifiedCount / totalFindings) * 100 : 0,
    totalScans: stats.totalScans || 0,
    totalErrors: stats.totalErrors || 0,
    avgScanDurationMs: stats.avgDuration || 0,
  }
}

export interface ScanStat {
  id: number
  worker_id: string
  scan_date: string
  repositories_scanned: number
  files_scanned: number
  commits_scanned: number
  findings_detected: number
  findings_notified: number
  errors_encountered: number
  scan_duration_ms: number
}

export async function getRecentScans(limit: number = 50): Promise<ScanStat[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT * FROM statistics
    ORDER BY scan_date DESC
    LIMIT ?
  `)
  return stmt.all(limit) as ScanStat[]
}

export async function getScanPerformance(days: number = 30): Promise<{
  date: string
  repos: number
  files: number
  findings: number
  duration: number
}[]> {
  const db = getDb()
  const stmt = db.prepare(`
    SELECT
      DATE(scan_date) as date,
      SUM(repositories_scanned) as repos,
      SUM(files_scanned) as files,
      SUM(findings_detected) as findings,
      AVG(scan_duration_ms) as duration
    FROM statistics
    WHERE scan_date >= DATE('now', '-${days} days')
    GROUP BY DATE(scan_date)
    ORDER BY date
  `)
  return stmt.all() as {
    date: string
    repos: number
    files: number
    findings: number
    duration: number
  }[]
}

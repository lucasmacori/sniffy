package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lucasmacori/sniffy/internal/detector"
	_ "github.com/mattn/go-sqlite3"
)

// Storage handles persistence of findings and statistics
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new SQLite storage instance
func NewStorage(dbPath string) (*Storage, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create database directory failed: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database failed: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database failed: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate database failed: %w", err)
	}

	return s, nil
}

// migrate creates the database schema
func (s *Storage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS findings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repository TEXT NOT NULL,
		commit_hash TEXT,
		commit_author TEXT,
		commit_email TEXT,
		file_path TEXT NOT NULL,
		line_number INTEGER NOT NULL,
		secret_type TEXT NOT NULL,
		secret_value TEXT NOT NULL,
		confidence REAL NOT NULL,
		source TEXT NOT NULL,
		html_url TEXT,
		notified INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(repository, commit_hash, file_path, line_number)
	);

	CREATE TABLE IF NOT EXISTS statistics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		worker_id TEXT NOT NULL,
		scan_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		repositories_scanned INTEGER DEFAULT 0,
		files_scanned INTEGER DEFAULT 0,
		commits_scanned INTEGER DEFAULT 0,
		findings_detected INTEGER DEFAULT 0,
		findings_notified INTEGER DEFAULT 0,
		errors_encountered INTEGER DEFAULT 0,
		scan_duration_ms INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS scan_checkpoints (
		track_name TEXT PRIMARY KEY,
		last_value TIMESTAMP,
		last_page INTEGER DEFAULT 1,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_findings_repo ON findings(repository);
	CREATE INDEX IF NOT EXISTS idx_findings_notified ON findings(notified);
	CREATE INDEX IF NOT EXISTS idx_statistics_worker ON statistics(worker_id);
	CREATE INDEX IF NOT EXISTS idx_statistics_date ON statistics(scan_date);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("execute schema failed: %w", err)
	}

	return nil
}

// SaveFinding saves a finding to the database if it doesn't already exist
func (s *Storage) SaveFinding(finding detector.Finding) (bool, error) {
	var commitHash interface{}
	if finding.CommitHash != "" {
		commitHash = finding.CommitHash
	} else {
		commitHash = nil
	}

	result, err := s.db.Exec(`
		INSERT OR IGNORE INTO findings 
		(repository, commit_hash, commit_author, commit_email, file_path, line_number, secret_type, secret_value, confidence, source, html_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, finding.Repository, commitHash, finding.CommitAuthor, finding.CommitEmail,
		finding.FilePath, finding.LineNumber, finding.SecretType, finding.SecretValue,
		finding.Confidence, finding.Source, finding.HTMLURL)

	if err != nil {
		return false, fmt.Errorf("insert finding failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get rows affected failed: %w", err)
	}

	return rowsAffected > 0, nil
}

// MarkNotified marks a finding as notified
func (s *Storage) MarkNotified(finding detector.Finding) error {
	var commitHash interface{}
	if finding.CommitHash != "" {
		commitHash = finding.CommitHash
	} else {
		commitHash = nil
	}

	_, err := s.db.Exec(`
		UPDATE findings SET notified = 1
		WHERE repository = ? AND commit_hash IS ? AND file_path = ? AND line_number = ?
	`, finding.Repository, commitHash, finding.FilePath, finding.LineNumber)

	if err != nil {
		return fmt.Errorf("mark notified failed: %w", err)
	}

	return nil
}

// IsDuplicate checks if a finding already exists
func (s *Storage) IsDuplicate(finding detector.Finding) (bool, error) {
	var commitHash interface{}
	if finding.CommitHash != "" {
		commitHash = finding.CommitHash
	} else {
		commitHash = nil
	}

	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM findings
		WHERE repository = ? AND commit_hash IS ? AND file_path = ? AND line_number = ?
	`, finding.Repository, commitHash, finding.FilePath, finding.LineNumber).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("check duplicate failed: %w", err)
	}

	return count > 0, nil
}

// GetCheckpoint retrieves the last checkpoint for a track
func (s *Storage) GetCheckpoint(track string) (time.Time, int, error) {
	var lastValue sql.NullTime
	var lastPage int

	err := s.db.QueryRow(`
		SELECT last_value, last_page FROM scan_checkpoints WHERE track_name = ?
	`, track).Scan(&lastValue, &lastPage)

	if err == sql.ErrNoRows {
		return time.Time{}, 1, nil
	}
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("get checkpoint failed: %w", err)
	}

	if !lastValue.Valid {
		return time.Time{}, lastPage, nil
	}

	return lastValue.Time, lastPage, nil
}

// SaveCheckpoint saves the checkpoint for a track
func (s *Storage) SaveCheckpoint(track string, lastValue time.Time, page int) error {
	_, err := s.db.Exec(`
		INSERT INTO scan_checkpoints (track_name, last_value, last_page, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(track_name) DO UPDATE SET
			last_value = excluded.last_value,
			last_page = excluded.last_page,
			updated_at = excluded.updated_at
	`, track, lastValue, page, time.Now())

	if err != nil {
		return fmt.Errorf("save checkpoint failed: %w", err)
	}

	return nil
}

// GetStatistics returns aggregate statistics
func (s *Storage) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalFindings int
	err := s.db.QueryRow("SELECT COUNT(*) FROM findings").Scan(&totalFindings)
	if err != nil {
		return nil, fmt.Errorf("get total findings failed: %w", err)
	}
	stats["total_findings"] = totalFindings

	var notifiedFindings int
	err = s.db.QueryRow("SELECT COUNT(*) FROM findings WHERE notified = 1").Scan(&notifiedFindings)
	if err != nil {
		return nil, fmt.Errorf("get notified findings failed: %w", err)
	}
	stats["notified_findings"] = notifiedFindings

	var totalScans int
	err = s.db.QueryRow("SELECT COUNT(*) FROM statistics").Scan(&totalScans)
	if err != nil {
		return nil, fmt.Errorf("get total scans failed: %w", err)
	}
	stats["total_scans"] = totalScans

	return stats, nil
}

// SaveScanStats saves statistics for a scan cycle
func (s *Storage) SaveScanStats(workerID string, repos, files, commits, findings, notified, errors int, duration time.Duration) error {
	_, err := s.db.Exec(`
		INSERT INTO statistics 
		(worker_id, repositories_scanned, files_scanned, commits_scanned, findings_detected, findings_notified, errors_encountered, scan_duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, workerID, repos, files, commits, findings, notified, errors, int(duration.Milliseconds()))

	if err != nil {
		return fmt.Errorf("insert statistics failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

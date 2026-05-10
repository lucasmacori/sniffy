package storage

import (
	"os"
	"testing"
	"time"

	"github.com/lucasmacori/sniffy/internal/detector"
)

func setupTestStorage(t *testing.T) *Storage {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	s, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	t.Cleanup(func() {
		s.Close()
	})

	return s
}

func TestNewStorage(t *testing.T) {
	t.Run("creates database and schema", func(t *testing.T) {
		s := setupTestStorage(t)
		if s.db == nil {
			t.Error("db is nil")
		}
	})

	t.Run("creates directory if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/nested/deep/test.db"

		s, err := NewStorage(dbPath)
		if err != nil {
			t.Fatalf("NewStorage() error = %v", err)
		}
		defer s.Close()

		if _, err := os.Stat(tmpDir + "/nested/deep"); os.IsNotExist(err) {
			t.Error("expected directory to be created")
		}
	})
}

func TestStorage_SaveFinding(t *testing.T) {
	s := setupTestStorage(t)

	finding := detector.Finding{
		Repository:  "owner/repo",
		CommitHash:  "abc123",
		FilePath:    "config.env",
		LineNumber:  5,
		SecretType:  "AWS Key",
		SecretValue: "AKIAIOSFODNN7EXAMPLE",
		Confidence:  85.0,
		Source:      "worktree",
		HTMLURL:     "https://github.com/owner/repo",
	}

	t.Run("save new finding", func(t *testing.T) {
		saved, err := s.SaveFinding(finding)
		if err != nil {
			t.Fatalf("SaveFinding() error = %v", err)
		}
		if !saved {
			t.Error("SaveFinding() = false; want true")
		}
	})

	t.Run("duplicate finding", func(t *testing.T) {
		saved, err := s.SaveFinding(finding)
		if err != nil {
			t.Fatalf("SaveFinding() error = %v", err)
		}
		if saved {
			t.Error("SaveFinding() = true; want false for duplicate")
		}
	})

	t.Run("finding without commit hash", func(t *testing.T) {
		f := detector.Finding{
			Repository:  "owner/repo2",
			FilePath:    "test.txt",
			LineNumber:  1,
			SecretType:  "Token",
			SecretValue: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			Confidence:  90.0,
			Source:      "worktree",
		}

		saved, err := s.SaveFinding(f)
		if err != nil {
			t.Fatalf("SaveFinding() error = %v", err)
		}
		if !saved {
			t.Error("SaveFinding() = false; want true")
		}
	})
}

func TestStorage_IsDuplicate(t *testing.T) {
	s := setupTestStorage(t)

	finding := detector.Finding{
		Repository: "owner/repo",
		CommitHash: "abc123",
		FilePath:   "config.env",
		LineNumber: 5,
	}

	// Save first
	if _, err := s.SaveFinding(finding); err != nil {
		t.Fatalf("SaveFinding() error = %v", err)
	}

	t.Run("existing finding", func(t *testing.T) {
		isDup, err := s.IsDuplicate(finding)
		if err != nil {
			t.Fatalf("IsDuplicate() error = %v", err)
		}
		if !isDup {
			t.Error("IsDuplicate() = false; want true")
		}
	})

	t.Run("new finding", func(t *testing.T) {
		newFinding := detector.Finding{
			Repository: "owner/repo",
			CommitHash: "def456",
			FilePath:   "config.env",
			LineNumber: 5,
		}
		isDup, err := s.IsDuplicate(newFinding)
		if err != nil {
			t.Fatalf("IsDuplicate() error = %v", err)
		}
		if isDup {
			t.Error("IsDuplicate() = true; want false")
		}
	})
}

func TestStorage_MarkNotified(t *testing.T) {
	s := setupTestStorage(t)

	finding := detector.Finding{
		Repository: "owner/repo",
		FilePath:   "config.env",
		LineNumber: 5,
		SecretType: "AWS Key",
		Confidence: 85.0,
		Source:     "worktree",
	}

	if _, err := s.SaveFinding(finding); err != nil {
		t.Fatalf("SaveFinding() error = %v", err)
	}

	err := s.MarkNotified(finding)
	if err != nil {
		t.Fatalf("MarkNotified() error = %v", err)
	}

	stats, err := s.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics() error = %v", err)
	}
	if stats["notified_findings"] != 1 {
		t.Errorf("notified_findings = %v; want 1", stats["notified_findings"])
	}
}

func TestStorage_Checkpoint(t *testing.T) {
	s := setupTestStorage(t)

	t.Run("new checkpoint", func(t *testing.T) {
		lastValue, page, err := s.GetCheckpoint("fresh")
		if err != nil {
			t.Fatalf("GetCheckpoint() error = %v", err)
		}
		if !lastValue.IsZero() {
			t.Errorf("lastValue = %v; want zero", lastValue)
		}
		if page != 1 {
			t.Errorf("page = %d; want 1", page)
		}
	})

	t.Run("save and retrieve checkpoint", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		err := s.SaveCheckpoint("fresh", ts, 5)
		if err != nil {
			t.Fatalf("SaveCheckpoint() error = %v", err)
		}

		lastValue, page, err := s.GetCheckpoint("fresh")
		if err != nil {
			t.Fatalf("GetCheckpoint() error = %v", err)
		}
		if !lastValue.Equal(ts) {
			t.Errorf("lastValue = %v; want %v", lastValue, ts)
		}
		if page != 5 {
			t.Errorf("page = %d; want 5", page)
		}
	})

	t.Run("update existing checkpoint", func(t *testing.T) {
		ts := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)
		err := s.SaveCheckpoint("fresh", ts, 10)
		if err != nil {
			t.Fatalf("SaveCheckpoint() error = %v", err)
		}

		lastValue, page, err := s.GetCheckpoint("fresh")
		if err != nil {
			t.Fatalf("GetCheckpoint() error = %v", err)
		}
		if !lastValue.Equal(ts) {
			t.Errorf("lastValue = %v; want %v", lastValue, ts)
		}
		if page != 10 {
			t.Errorf("page = %d; want 10", page)
		}
	})
}

func TestStorage_GetStatistics(t *testing.T) {
	s := setupTestStorage(t)

	stats, err := s.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics() error = %v", err)
	}

	if stats["total_findings"] != 0 {
		t.Errorf("total_findings = %v; want 0", stats["total_findings"])
	}
	if stats["notified_findings"] != 0 {
		t.Errorf("notified_findings = %v; want 0", stats["notified_findings"])
	}
	if stats["total_scans"] != 0 {
		t.Errorf("total_scans = %v; want 0", stats["total_scans"])
	}
}

func TestStorage_SaveScanStats(t *testing.T) {
	s := setupTestStorage(t)

	err := s.SaveScanStats("worker-1", 10, 500, 100, 5, 3, 1, 30*time.Second)
	if err != nil {
		t.Fatalf("SaveScanStats() error = %v", err)
	}

	stats, err := s.GetStatistics()
	if err != nil {
		t.Fatalf("GetStatistics() error = %v", err)
	}
	if stats["total_scans"] != 1 {
		t.Errorf("total_scans = %v; want 1", stats["total_scans"])
	}
}

func TestStorage_Close(t *testing.T) {
	s := setupTestStorage(t)

	err := s.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

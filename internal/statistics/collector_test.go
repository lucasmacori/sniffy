package statistics

import (
	"testing"
	"time"
)

func TestCollector_StartCycle(t *testing.T) {
	c := NewCollector()
	c.StartCycle()

	stats := c.GetCurrentStats()
	if stats.RepositoriesScanned != 0 {
		t.Errorf("RepositoriesScanned = %d; want 0", stats.RepositoriesScanned)
	}
	if stats.FilesScanned != 0 {
		t.Errorf("FilesScanned = %d; want 0", stats.FilesScanned)
	}
	if stats.CommitsScanned != 0 {
		t.Errorf("CommitsScanned = %d; want 0", stats.CommitsScanned)
	}
	if stats.FindingsDetected != 0 {
		t.Errorf("FindingsDetected = %d; want 0", stats.FindingsDetected)
	}
	if stats.FindingsNotified != 0 {
		t.Errorf("FindingsNotified = %d; want 0", stats.FindingsNotified)
	}
	if stats.ErrorsEncountered != 0 {
		t.Errorf("ErrorsEncountered = %d; want 0", stats.ErrorsEncountered)
	}
}

func TestCollector_IncrementMethods(t *testing.T) {
	c := NewCollector()
	c.StartCycle()

	c.IncrementRepositories(5)
	c.IncrementFiles(100)
	c.IncrementCommits(50)
	c.IncrementFindings(3)
	c.IncrementNotified(2)
	c.IncrementErrors(1)

	stats := c.GetCurrentStats()
	if stats.RepositoriesScanned != 5 {
		t.Errorf("RepositoriesScanned = %d; want 5", stats.RepositoriesScanned)
	}
	if stats.FilesScanned != 100 {
		t.Errorf("FilesScanned = %d; want 100", stats.FilesScanned)
	}
	if stats.CommitsScanned != 50 {
		t.Errorf("CommitsScanned = %d; want 50", stats.CommitsScanned)
	}
	if stats.FindingsDetected != 3 {
		t.Errorf("FindingsDetected = %d; want 3", stats.FindingsDetected)
	}
	if stats.FindingsNotified != 2 {
		t.Errorf("FindingsNotified = %d; want 2", stats.FindingsNotified)
	}
	if stats.ErrorsEncountered != 1 {
		t.Errorf("ErrorsEncountered = %d; want 1", stats.ErrorsEncountered)
	}
}

func TestCollector_EndCycle(t *testing.T) {
	c := NewCollector()
	c.StartCycle()

	c.IncrementRepositories(5)
	c.IncrementFiles(100)
	c.IncrementFindings(3)

	duration := c.EndCycle()
	if duration < 0 {
		t.Errorf("EndCycle() duration = %v; want >= 0", duration)
	}

	total := c.GetTotalStats()
	if total.RepositoriesScanned != 5 {
		t.Errorf("Total RepositoriesScanned = %d; want 5", total.RepositoriesScanned)
	}
	if total.FilesScanned != 100 {
		t.Errorf("Total FilesScanned = %d; want 100", total.FilesScanned)
	}
	if total.FindingsDetected != 3 {
		t.Errorf("Total FindingsDetected = %d; want 3", total.FindingsDetected)
	}
	if total.TotalScans != 1 {
		t.Errorf("TotalScans = %d; want 1", total.TotalScans)
	}

	// Start a new cycle and verify current stats are reset
	c.StartCycle()
	current := c.GetCurrentStats()
	if current.RepositoriesScanned != 0 {
		t.Errorf("Current RepositoriesScanned = %d; want 0 after StartCycle", current.RepositoriesScanned)
	}
}

func TestCollector_MultipleCycles(t *testing.T) {
	c := NewCollector()

	for i := 0; i < 3; i++ {
		c.StartCycle()
		c.IncrementRepositories(2)
		c.IncrementFiles(50)
		c.EndCycle()
	}

	total := c.GetTotalStats()
	if total.RepositoriesScanned != 6 {
		t.Errorf("Total RepositoriesScanned = %d; want 6", total.RepositoriesScanned)
	}
	if total.FilesScanned != 150 {
		t.Errorf("Total FilesScanned = %d; want 150", total.FilesScanned)
	}
	if total.TotalScans != 3 {
		t.Errorf("TotalScans = %d; want 3", total.TotalScans)
	}
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	c := NewCollector()
	c.StartCycle()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				c.IncrementFiles(1)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := c.GetCurrentStats()
	if stats.FilesScanned != 1000 {
		t.Errorf("FilesScanned = %d; want 1000", stats.FilesScanned)
	}
}

func TestStats_String(t *testing.T) {
	s := Stats{
		RepositoriesScanned: 10,
		FilesScanned:        500,
		CommitsScanned:      200,
		FindingsDetected:    5,
		FindingsNotified:    3,
		ErrorsEncountered:   1,
		Duration:            30 * time.Second,
		TotalScans:          2,
	}

	got := s.String()
	want := "Repos: 10, Files: 500, Commits: 200, Findings: 5, Notified: 3, Errors: 1, Duration: 30s"
	if got != want {
		t.Errorf("String() = %q; want %q", got, want)
	}
}

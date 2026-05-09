package statistics

import (
	"fmt"
	"sync"
	"time"
)

// Collector collects and aggregates scan statistics
type Collector struct {
	mu sync.RWMutex

	// Current cycle stats
	repositoriesScanned int
	filesScanned        int
	commitsScanned      int
	findingsDetected    int
	findingsNotified    int
	errorsEncountered   int
	scanStartTime       time.Time

	// Historical stats
	totalRepositories int
	totalFiles        int
	totalCommits      int
	totalFindings     int
	totalNotified     int
	totalErrors       int
	totalScans        int
}

// NewCollector creates a new statistics collector
func NewCollector() *Collector {
	return &Collector{}
}

// StartCycle begins a new scan cycle
func (c *Collector) StartCycle() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.repositoriesScanned = 0
	c.filesScanned = 0
	c.commitsScanned = 0
	c.findingsDetected = 0
	c.findingsNotified = 0
	c.errorsEncountered = 0
	c.scanStartTime = time.Now()
}

// EndCycle ends the current scan cycle and returns the duration
func (c *Collector) EndCycle() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	duration := time.Since(c.scanStartTime)

	c.totalRepositories += c.repositoriesScanned
	c.totalFiles += c.filesScanned
	c.totalCommits += c.commitsScanned
	c.totalFindings += c.findingsDetected
	c.totalNotified += c.findingsNotified
	c.totalErrors += c.errorsEncountered
	c.totalScans++

	return duration
}

// IncrementRepositories increments the repository count
func (c *Collector) IncrementRepositories(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.repositoriesScanned += count
}

// IncrementFiles increments the file count
func (c *Collector) IncrementFiles(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.filesScanned += count
}

// IncrementCommits increments the commit count
func (c *Collector) IncrementCommits(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commitsScanned += count
}

// IncrementFindings increments the findings count
func (c *Collector) IncrementFindings(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.findingsDetected += count
}

// IncrementNotified increments the notified count
func (c *Collector) IncrementNotified(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.findingsNotified += count
}

// IncrementErrors increments the error count
func (c *Collector) IncrementErrors(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorsEncountered += count
}

// GetCurrentStats returns the current cycle statistics
func (c *Collector) GetCurrentStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		RepositoriesScanned: c.repositoriesScanned,
		FilesScanned:        c.filesScanned,
		CommitsScanned:      c.commitsScanned,
		FindingsDetected:    c.findingsDetected,
		FindingsNotified:    c.findingsNotified,
		ErrorsEncountered:   c.errorsEncountered,
		Duration:            time.Since(c.scanStartTime),
	}
}

// GetTotalStats returns the cumulative statistics
func (c *Collector) GetTotalStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Stats{
		RepositoriesScanned: c.totalRepositories,
		FilesScanned:        c.totalFiles,
		CommitsScanned:      c.totalCommits,
		FindingsDetected:    c.totalFindings,
		FindingsNotified:    c.totalNotified,
		ErrorsEncountered:   c.totalErrors,
		TotalScans:          c.totalScans,
	}
}

// Stats holds scan statistics
type Stats struct {
	RepositoriesScanned int
	FilesScanned        int
	CommitsScanned      int
	FindingsDetected    int
	FindingsNotified    int
	ErrorsEncountered   int
	Duration            time.Duration
	TotalScans          int
}

// String returns a formatted string of the statistics
func (s Stats) String() string {
	return fmt.Sprintf("Repos: %d, Files: %d, Commits: %d, Findings: %d, Notified: %d, Errors: %d, Duration: %s",
		s.RepositoriesScanned, s.FilesScanned, s.CommitsScanned,
		s.FindingsDetected, s.FindingsNotified, s.ErrorsEncountered,
		s.Duration.Round(time.Second))
}

package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lucasmacori/sniffy/internal/config"
	"github.com/lucasmacori/sniffy/internal/detector"
	"github.com/lucasmacori/sniffy/internal/git"
	"github.com/lucasmacori/sniffy/internal/notifier"
	"github.com/lucasmacori/sniffy/internal/source"
	"github.com/lucasmacori/sniffy/internal/statistics"
	"github.com/lucasmacori/sniffy/internal/storage"
)

const (
	trackFresh  = "fresh"
	trackActive = "active"
)

// Worker orchestrates the scanning process with dual-track concurrent scanning
type Worker struct {
	config    *config.Config
	src       source.Source
	cloner    *git.Cloner
	inspector *git.Inspector
	detector  *detector.CompositeDetector
	notifier  *notifier.Registry
	storage   *storage.Storage
	stats     *statistics.Collector
}

// NewWorker creates a new worker instance
func NewWorker(
	cfg *config.Config,
	src source.Source,
	cloner *git.Cloner,
	inspector *git.Inspector,
	det *detector.CompositeDetector,
	notif *notifier.Registry,
	store *storage.Storage,
	stats *statistics.Collector,
) *Worker {
	return &Worker{
		config:    cfg,
		src:       src,
		cloner:    cloner,
		inspector: inspector,
		detector:  det,
		notifier:  notif,
		storage:   store,
		stats:     stats,
	}
}

// Run starts the dual-track worker loop
func (w *Worker) Run(ctx context.Context) error {
	log.Printf("[%s] Worker started", w.config.WorkerID)
	log.Printf("[%s] Platform: %s", w.config.WorkerID, w.src.Name())
	log.Printf("[%s] Configuration: threshold=%.1f%%, max_concurrent=%d, max_repo_size=%.1fGB, disk_limit=%.1fGB",
		w.config.WorkerID, w.config.ConfidenceThreshold, w.config.MaxConcurrentClones,
		w.config.MaxRepoSizeGB, w.config.DiskLimitGB)
	log.Printf("[%s] Dual-track scanning: Fresh (new repos) + Active (recently updated, %dh window)",
		w.config.WorkerID, w.config.ActiveScanWindowHours)

	// Start both tracks concurrently
	errChan := make(chan error, 2)

	go w.runFreshTrack(ctx, errChan)
	go w.runActiveTrack(ctx, errChan)

	// Wait for either track to fail or context cancellation
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		log.Printf("[%s] Worker shutting down", w.config.WorkerID)
		return ctx.Err()
	}
}

// runFreshTrack scans for newly created repositories
func (w *Worker) runFreshTrack(ctx context.Context, errChan chan<- error) {
	log.Printf("[%s] [Fresh Track] Started", w.config.WorkerID)

	perPage := 30
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Load checkpoint from database
		lastCreatedAt, lastPage, err := w.storage.GetCheckpoint(trackFresh)
		if err != nil {
			log.Printf("[%s] [Fresh Track] Error loading checkpoint: %v", w.config.WorkerID, err)
			w.sleep(ctx, 10*time.Second)
			continue
		}

		page := lastPage
		if lastCreatedAt.IsZero() {
			// First run: start from the beginning (newest repos)
			page = 1
		}

		log.Printf("[%s] [Fresh Track] Scanning page %d, checkpoint: %s",
			w.config.WorkerID, page, lastCreatedAt.Format(time.RFC3339))

		// Discover fresh repositories
		repos, err := w.src.DiscoverFresh(ctx, page, perPage, lastCreatedAt)
		if err != nil {
			consecutiveErrors++
			log.Printf("[%s] [Fresh Track] Discovery error: %v", w.config.WorkerID, err)
			w.stats.IncrementErrors(1)

			backoff := w.calculateBackoff(consecutiveErrors)
			if w.isRateLimitError(err) {
				log.Printf("[%s] [Fresh Track] Rate limit hit. Backing off for %v...", w.config.WorkerID, backoff)
			}
			w.sleep(ctx, backoff)
			continue
		}

		// Reset consecutive errors on success
		consecutiveErrors = 0

		// Filter out repos that are too large
		repos = w.filterBySize(repos)

		if len(repos) == 0 {
			log.Printf("[%s] [Fresh Track] No new repositories found. Sleeping %ds...",
				w.config.WorkerID, w.config.FreshTrackSleepSeconds)
			w.sleep(ctx, time.Duration(w.config.FreshTrackSleepSeconds)*time.Second)

			// Reset to page 1 to catch any repos created during sleep
			if err := w.storage.SaveCheckpoint(trackFresh, lastCreatedAt, 1); err != nil {
				log.Printf("[%s] [Fresh Track] Error saving checkpoint: %v", w.config.WorkerID, err)
			}
			continue
		}

		log.Printf("[%s] [Fresh Track] Discovered %d new repositories on page %d",
			w.config.WorkerID, len(repos), page)

		// Process each repository
		maxCreatedAt := lastCreatedAt
		for _, repo := range repos {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := w.processRepository(ctx, repo); err != nil {
				log.Printf("[%s] [Fresh Track] Error processing %s: %v",
					w.config.WorkerID, repo.FullName, err)
				w.stats.IncrementErrors(1)
			}

			// Track the newest repo we've seen
			if repo.CreatedAt.After(maxCreatedAt) {
				maxCreatedAt = repo.CreatedAt
			}
		}

		// Update checkpoint
		if !maxCreatedAt.Equal(lastCreatedAt) {
			if err := w.storage.SaveCheckpoint(trackFresh, maxCreatedAt, 1); err != nil {
				log.Printf("[%s] [Fresh Track] Error saving checkpoint: %v", w.config.WorkerID, err)
			} else {
				log.Printf("[%s] [Fresh Track] Checkpoint updated to %s",
					w.config.WorkerID, maxCreatedAt.Format(time.RFC3339))
			}
		}

		// Move to next page
		nextPage := page + 1
		if err := w.storage.SaveCheckpoint(trackFresh, maxCreatedAt, nextPage); err != nil {
			log.Printf("[%s] [Fresh Track] Error saving page checkpoint: %v", w.config.WorkerID, err)
		}
	}
}

// runActiveTrack scans for recently updated repositories
func (w *Worker) runActiveTrack(ctx context.Context, errChan chan<- error) {
	log.Printf("[%s] [Active Track] Started", w.config.WorkerID)

	perPage := 30
	windowDuration := time.Duration(w.config.ActiveScanWindowHours) * time.Hour
	overlapBuffer := 30 * time.Minute // Scan 30 min into the past to catch edge cases
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cycleStart := time.Now().UTC()

		// Calculate the scan window (now - window - overlap)
		since := cycleStart.Add(-windowDuration - overlapBuffer)

		log.Printf("[%s] [Active Track] Scanning window: %s to %s",
			w.config.WorkerID,
			since.Format(time.RFC3339),
			cycleStart.Format(time.RFC3339))

		page := 1
		totalRepos := 0

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Discover active repositories
			repos, err := w.src.DiscoverActive(ctx, page, perPage, since)
			if err != nil {
				consecutiveErrors++
				log.Printf("[%s] [Active Track] Discovery error on page %d: %v",
					w.config.WorkerID, page, err)
				w.stats.IncrementErrors(1)

				backoff := w.calculateBackoff(consecutiveErrors)
				if w.isRateLimitError(err) {
					log.Printf("[%s] [Active Track] Rate limit hit. Backing off for %v...",
						w.config.WorkerID, backoff)
					w.sleep(ctx, backoff)
					break // Start a new cycle after backoff
				}
				break // Non-rate-limit error, just break and start new cycle
			}

			// Reset consecutive errors on success
			consecutiveErrors = 0

			// Filter out repos that are too large
			repos = w.filterBySize(repos)

			if len(repos) == 0 {
				break // No more repos in this window
			}

			log.Printf("[%s] [Active Track] Page %d: %d repositories",
				w.config.WorkerID, page, len(repos))

			// Process each repository
			for _, repo := range repos {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if err := w.processRepository(ctx, repo); err != nil {
					log.Printf("[%s] [Active Track] Error processing %s: %v",
						w.config.WorkerID, repo.FullName, err)
					w.stats.IncrementErrors(1)
				}
			}

			totalRepos += len(repos)
			page++
		}

		log.Printf("[%s] [Active Track] Window complete. Scanned %d repos. Starting next cycle...",
			w.config.WorkerID, totalRepos)
	}
}

// filterBySize filters repositories by size limit
func (w *Worker) filterBySize(repos []source.Repository) []source.Repository {
	var filtered []source.Repository
	for _, repo := range repos {
		sizeGB := float64(repo.SizeKB) / (1024 * 1024)
		if sizeGB <= w.config.MaxRepoSizeGB {
			filtered = append(filtered, repo)
		} else {
			log.Printf("[%s] Skipping %s (%.2f GB > %.2f GB limit)",
				w.config.WorkerID, repo.FullName, sizeGB, w.config.MaxRepoSizeGB)
		}
	}
	return filtered
}

// processRepository clones, scans, and notifies for a single repository
func (w *Worker) processRepository(ctx context.Context, repo source.Repository) error {
	log.Printf("[%s] Processing repository: %s (%.2f MB)",
		w.config.WorkerID, repo.FullName, float64(repo.SizeKB)/1024)

	w.stats.IncrementRepositories(1)

	// Clone the repository
	cloneResult, err := w.cloner.Clone(ctx, repo.CloneURL, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}
	defer w.cloner.Cleanup(cloneResult)

	log.Printf("[%s] Cloned %s to %s", w.config.WorkerID, repo.FullName, cloneResult.Path)

	// Count files for statistics
	files, err := w.inspector.GetWorktreeFiles(cloneResult.Path)
	if err != nil {
		log.Printf("[%s] Warning: could not count files in %s: %v",
			w.config.WorkerID, repo.FullName, err)
	} else {
		w.stats.IncrementFiles(len(files))
	}

	// Count commits for statistics
	commits, err := w.inspector.GetAllCommits(ctx, cloneResult.Path)
	if err != nil {
		log.Printf("[%s] Warning: could not count commits in %s: %v",
			w.config.WorkerID, repo.FullName, err)
	} else {
		w.stats.IncrementCommits(len(commits))
	}

	// Run detection
	findings, err := w.detector.Detect(ctx, cloneResult.Path, w.inspector)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	if len(findings) > 0 {
		log.Printf("[%s] Found %d potential secrets in %s",
			w.config.WorkerID, len(findings), repo.FullName)
	}

	// Update findings with repository info
	for i := range findings {
		findings[i].Repository = repo.FullName
		findings[i].HTMLURL = repo.HTMLURL
	}

	// Process findings
	newFindings := 0
	notifiedCount := 0

	for _, finding := range findings {
		// Check for duplicates
		isDup, err := w.storage.IsDuplicate(finding)
		if err != nil {
			log.Printf("[%s] Error checking duplicate: %v", w.config.WorkerID, err)
			continue
		}

		if isDup {
			continue
		}

		// Save the finding
		saved, err := w.storage.SaveFinding(finding)
		if err != nil {
			log.Printf("[%s] Error saving finding: %v", w.config.WorkerID, err)
			continue
		}

		if saved {
			newFindings++
		}

		// Notify
		if err := w.notifier.Notify(ctx, finding); err != nil {
			log.Printf("[%s] Notification error: %v", w.config.WorkerID, err)
		} else {
			if err := w.storage.MarkNotified(finding); err != nil {
				log.Printf("[%s] Error marking notified: %v", w.config.WorkerID, err)
			} else {
				notifiedCount++
			}
		}
	}

	w.stats.IncrementFindings(newFindings)
	w.stats.IncrementNotified(notifiedCount)

	if newFindings > 0 {
		log.Printf("[%s] %s: %d new findings, %d notified",
			w.config.WorkerID, repo.FullName, newFindings, notifiedCount)
	}

	return nil
}

// sleep sleeps for the given duration or until context cancellation
func (w *Worker) sleep(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
}

// isRateLimitError checks if an error is due to rate limiting
func (w *Worker) isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "secondary rate limit")
}

// calculateBackoff returns an appropriate sleep duration based on consecutive errors
func (w *Worker) calculateBackoff(consecutiveErrors int) time.Duration {
	base := 30 * time.Second
	maxBackoff := 5 * time.Minute

	// Exponential backoff: 30s, 60s, 120s, 240s, max 5min
	backoff := base * time.Duration(1<<uint(consecutiveErrors))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

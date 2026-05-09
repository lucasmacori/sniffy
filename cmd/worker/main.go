package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/lucasmacori/sniffy/internal/config"
	"github.com/lucasmacori/sniffy/internal/detector"
	"github.com/lucasmacori/sniffy/internal/git"
	"github.com/lucasmacori/sniffy/internal/github"
	"github.com/lucasmacori/sniffy/internal/notifier"
	"github.com/lucasmacori/sniffy/internal/ratelimiter"
	"github.com/lucasmacori/sniffy/internal/source"
	"github.com/lucasmacori/sniffy/internal/statistics"
	"github.com/lucasmacori/sniffy/internal/storage"
	"github.com/lucasmacori/sniffy/internal/worker"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using existing environment variables")
	}

	log.Println("Starting Sniffy - Credential Leak Detector")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Worker ID: %s", cfg.WorkerID)

	// Initialize rate limiter
	isAuth := cfg.GitHubToken != ""
	limiter := ratelimiter.NewRateLimiter(isAuth)
	log.Printf("[RateLimiter] Initialized with %s rate limit", limiter.String())

	// Initialize source platform (GitHub)
	var src source.Source
	src = github.NewClient(cfg.GitHubToken, limiter)

	// Test connection and auto-detect authentication status
	ctx := context.Background()
	if err := src.TestConnection(ctx); err != nil {
		if isAuth {
			log.Printf("[Source] Connection test failed: %v", err)
			log.Printf("[Source] Attempting fallback to unauthenticated mode...")
			limiter.FallbackToUnauthenticated()
			// Re-create client without token
			src = github.NewClient("", limiter)
			if err := src.TestConnection(ctx); err != nil {
				log.Fatalf("[Source] Unauthenticated connection also failed: %v", err)
			}
			log.Printf("[Source] Fallback successful. Using unauthenticated mode.")
		} else {
			log.Fatalf("[Source] Connection test failed: %v", err)
		}
	} else {
		if src.IsAuthenticated() {
			log.Printf("[Source] ✅ Authenticated as %s", src.Name())
		} else {
			log.Printf("[Source] ⚠️  Running in unauthenticated mode (lower rate limits)")
		}
	}

	log.Printf("[RateLimiter] Using %s rate limit", limiter.String())

	// Initialize git cloner and inspector
	cloner := git.NewCloner(cfg.MaxConcurrentClones, cfg.MaxRepoSizeGB, cfg.DiskLimitGB)
	inspector := git.NewInspector()

	// Initialize detector with all strategies
	compositeDetector := detector.NewCompositeDetector(
		cfg.ConfidenceThreshold,
		detector.NewRegexDetector(),
		detector.NewEntropyDetector(),
		detector.NewGitHistoryDetector(),
	)

	// Initialize notifiers
	var notifiers []notifier.Notifier

	for _, notifType := range cfg.NotifierTypes {
		switch notifType {
		case "email":
			emailNotifier := notifier.NewEmailNotifier(
				cfg.SMTPHost,
				cfg.SMTPPort,
				cfg.SMTPUsername,
				cfg.SMTPPassword,
				cfg.SMTPFrom,
				cfg.SMTPTo,
				cfg.EmailConfidenceThreshold,
			)
			// Wrap with rate limiter
			rateLimited := notifier.NewRateLimitedNotifier(emailNotifier, cfg.NotificationRateLimit)
			notifiers = append(notifiers, rateLimited)
			log.Printf("[Notifier] Registered email notifier (threshold: %.1f%%)", cfg.EmailConfidenceThreshold)
		default:
			log.Printf("[Notifier] Unknown notifier type: %s", notifType)
		}
	}

	if len(notifiers) == 0 {
		log.Fatal("No notifiers configured")
	}

	notifRegistry := notifier.NewRegistry(notifiers...)

	// Initialize storage
	store, err := storage.NewStorage(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	log.Printf("[Storage] Initialized: %s", cfg.DatabasePath)

	// Initialize statistics collector
	statsCollector := statistics.NewCollector()

	// Create and run worker
	w := worker.NewWorker(
		cfg,
		src,
		cloner,
		inspector,
		compositeDetector,
		notifRegistry,
		store,
		statsCollector,
	)

	// Setup graceful shutdown
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received, stopping worker...")
		cancel()
	}()

	// Run the worker
	if err := w.Run(workerCtx); err != nil {
		if err == context.Canceled {
			log.Println("Worker stopped gracefully")
		} else {
			log.Fatalf("Worker error: %v", err)
		}
	}

	// Print final statistics
	totalStats := statsCollector.GetTotalStats()
	log.Printf("Final statistics: %s", totalStats.String())
}

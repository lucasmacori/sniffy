package config

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// GitHub
	GitHubToken string

	// Worker
	WorkerID string

	// Database
	DatabasePath string

	// Scanning
	ConfidenceThreshold  float64
	MaxConcurrentClones  int
	MaxRepoSizeGB        float64
	DiskLimitGB          float64
	ActiveScanWindowHours int
	FreshTrackSleepSeconds int

	// Notification
	NotifierTypes         []string
	NotificationRateLimit float64 // per second

	// Email
	EmailConfidenceThreshold float64
	SMTPHost                 string
	SMTPPort                 int
	SMTPUsername             string
	SMTPPassword             string
	SMTPFrom                 string
	SMTPTo                   string
}

// Load reads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	cfg := &Config{
		GitHubToken: getEnv("GITHUB_TOKEN", ""),

		WorkerID: getEnv("WORKER_ID", generateWorkerID()),

		DatabasePath: getEnv("DATABASE_PATH", "./data/sniffy.db"),

		ConfidenceThreshold:    getEnvFloat("CONFIDENCE_THRESHOLD", 25.0),
		MaxConcurrentClones:    getEnvInt("MAX_CONCURRENT_CLONES", 10),
		MaxRepoSizeGB:          getEnvFloat("MAX_REPO_SIZE_GB", 1.0),
		DiskLimitGB:            getEnvFloat("DISK_LIMIT_GB", 10.0),
		ActiveScanWindowHours:  getEnvInt("ACTIVE_SCAN_WINDOW_HOURS", 1),
		FreshTrackSleepSeconds: getEnvInt("FRESH_TRACK_SLEEP_SECONDS", 30),

		NotifierTypes:         getEnvSlice("NOTIFIER_TYPES", []string{"email"}),
		NotificationRateLimit: getEnvFloat("NOTIFICATION_RATE_LIMIT", 10.0),

		EmailConfidenceThreshold: getEnvFloat("EMAIL_CONFIDENCE_THRESHOLD", 25.0),
		SMTPHost:                 getEnv("SMTP_HOST", ""),
		SMTPPort:                 getEnvInt("SMTP_PORT", 587),
		SMTPUsername:             getEnv("SMTP_USERNAME", ""),
		SMTPPassword:             getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                 getEnv("SMTP_FROM", ""),
		SMTPTo:                   getEnv("SMTP_TO", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	if c.ConfidenceThreshold < 0 || c.ConfidenceThreshold > 100 {
		return fmt.Errorf("CONFIDENCE_THRESHOLD must be between 0 and 100")
	}
	if c.MaxConcurrentClones <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_CLONES must be greater than 0")
	}
	if c.MaxRepoSizeGB <= 0 {
		return fmt.Errorf("MAX_REPO_SIZE_GB must be greater than 0")
	}
	if c.DiskLimitGB <= 0 {
		return fmt.Errorf("DISK_LIMIT_GB must be greater than 0")
	}
	if c.ActiveScanWindowHours <= 0 {
		return fmt.Errorf("ACTIVE_SCAN_WINDOW_HOURS must be greater than 0")
	}
	if c.FreshTrackSleepSeconds < 0 {
		return fmt.Errorf("FRESH_TRACK_SLEEP_SECONDS must be >= 0")
	}
	if c.NotificationRateLimit <= 0 {
		return fmt.Errorf("NOTIFICATION_RATE_LIMIT must be greater than 0")
	}
	if c.EmailConfidenceThreshold < 0 || c.EmailConfidenceThreshold > 100 {
		return fmt.Errorf("EMAIL_CONFIDENCE_THRESHOLD must be between 0 and 100")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func generateWorkerID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	suffix := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10000)
	return fmt.Sprintf("%s-%04d", hostname, suffix)
}

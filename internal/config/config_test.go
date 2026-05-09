package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name:    "defaults",
			envVars: map[string]string{},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.GitHubToken != "" {
					t.Errorf("GitHubToken = %q; want empty", cfg.GitHubToken)
				}
				if cfg.DatabasePath != "./data/sniffy.db" {
					t.Errorf("DatabasePath = %q; want ./data/sniffy.db", cfg.DatabasePath)
				}
				if cfg.ConfidenceThreshold != 25.0 {
					t.Errorf("ConfidenceThreshold = %f; want 25.0", cfg.ConfidenceThreshold)
				}
				if cfg.MaxConcurrentClones != 10 {
					t.Errorf("MaxConcurrentClones = %d; want 10", cfg.MaxConcurrentClones)
				}
				if cfg.MaxRepoSizeGB != 1.0 {
					t.Errorf("MaxRepoSizeGB = %f; want 1.0", cfg.MaxRepoSizeGB)
				}
				if cfg.DiskLimitGB != 10.0 {
					t.Errorf("DiskLimitGB = %f; want 10.0", cfg.DiskLimitGB)
				}
				if cfg.ActiveScanWindowHours != 1 {
					t.Errorf("ActiveScanWindowHours = %d; want 1", cfg.ActiveScanWindowHours)
				}
				if cfg.FreshTrackSleepSeconds != 30 {
					t.Errorf("FreshTrackSleepSeconds = %d; want 30", cfg.FreshTrackSleepSeconds)
				}
				if len(cfg.NotifierTypes) != 1 || cfg.NotifierTypes[0] != "email" {
					t.Errorf("NotifierTypes = %v; want [email]", cfg.NotifierTypes)
				}
				if cfg.NotificationRateLimit != 10.0 {
					t.Errorf("NotificationRateLimit = %f; want 10.0", cfg.NotificationRateLimit)
				}
				if cfg.EmailConfidenceThreshold != 25.0 {
					t.Errorf("EmailConfidenceThreshold = %f; want 25.0", cfg.EmailConfidenceThreshold)
				}
				if cfg.SMTPPort != 587 {
					t.Errorf("SMTPPort = %d; want 587", cfg.SMTPPort)
				}
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"GITHUB_TOKEN":                "ghp_test123",
				"WORKER_ID":                   "worker-1",
				"DATABASE_PATH":               "/tmp/test.db",
				"CONFIDENCE_THRESHOLD":        "50",
				"MAX_CONCURRENT_CLONES":       "5",
				"MAX_REPO_SIZE_GB":            "2.5",
				"DISK_LIMIT_GB":               "20",
				"ACTIVE_SCAN_WINDOW_HOURS":    "2",
				"FRESH_TRACK_SLEEP_SECONDS":   "60",
				"NOTIFIER_TYPES":              "email, slack",
				"NOTIFICATION_RATE_LIMIT":     "5.5",
				"EMAIL_CONFIDENCE_THRESHOLD":  "40",
				"SMTP_HOST":                   "smtp.example.com",
				"SMTP_PORT":                   "465",
				"SMTP_USERNAME":               "user",
				"SMTP_PASSWORD":               "pass",
				"SMTP_FROM":                   "from@example.com",
				"SMTP_TO":                     "to@example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				if cfg.GitHubToken != "ghp_test123" {
					t.Errorf("GitHubToken = %q; want ghp_test123", cfg.GitHubToken)
				}
				if cfg.WorkerID != "worker-1" {
					t.Errorf("WorkerID = %q; want worker-1", cfg.WorkerID)
				}
				if cfg.DatabasePath != "/tmp/test.db" {
					t.Errorf("DatabasePath = %q; want /tmp/test.db", cfg.DatabasePath)
				}
				if cfg.ConfidenceThreshold != 50.0 {
					t.Errorf("ConfidenceThreshold = %f; want 50.0", cfg.ConfidenceThreshold)
				}
				if cfg.MaxConcurrentClones != 5 {
					t.Errorf("MaxConcurrentClones = %d; want 5", cfg.MaxConcurrentClones)
				}
				if cfg.MaxRepoSizeGB != 2.5 {
					t.Errorf("MaxRepoSizeGB = %f; want 2.5", cfg.MaxRepoSizeGB)
				}
				if cfg.DiskLimitGB != 20.0 {
					t.Errorf("DiskLimitGB = %f; want 20.0", cfg.DiskLimitGB)
				}
				if cfg.ActiveScanWindowHours != 2 {
					t.Errorf("ActiveScanWindowHours = %d; want 2", cfg.ActiveScanWindowHours)
				}
				if cfg.FreshTrackSleepSeconds != 60 {
					t.Errorf("FreshTrackSleepSeconds = %d; want 60", cfg.FreshTrackSleepSeconds)
				}
				if len(cfg.NotifierTypes) != 2 || cfg.NotifierTypes[0] != "email" || cfg.NotifierTypes[1] != "slack" {
					t.Errorf("NotifierTypes = %v; want [email slack]", cfg.NotifierTypes)
				}
				if cfg.NotificationRateLimit != 5.5 {
					t.Errorf("NotificationRateLimit = %f; want 5.5", cfg.NotificationRateLimit)
				}
				if cfg.EmailConfidenceThreshold != 40.0 {
					t.Errorf("EmailConfidenceThreshold = %f; want 40.0", cfg.EmailConfidenceThreshold)
				}
				if cfg.SMTPHost != "smtp.example.com" {
					t.Errorf("SMTPHost = %q; want smtp.example.com", cfg.SMTPHost)
				}
				if cfg.SMTPPort != 465 {
					t.Errorf("SMTPPort = %d; want 465", cfg.SMTPPort)
				}
				if cfg.SMTPUsername != "user" {
					t.Errorf("SMTPUsername = %q; want user", cfg.SMTPUsername)
				}
				if cfg.SMTPPassword != "pass" {
					t.Errorf("SMTPPassword = %q; want pass", cfg.SMTPPassword)
				}
				if cfg.SMTPFrom != "from@example.com" {
					t.Errorf("SMTPFrom = %q; want from@example.com", cfg.SMTPFrom)
				}
				if cfg.SMTPTo != "to@example.com" {
					t.Errorf("SMTPTo = %q; want to@example.com", cfg.SMTPTo)
				}
			},
		},
		{
			name: "invalid confidence threshold too high",
			envVars: map[string]string{
				"CONFIDENCE_THRESHOLD": "150",
			},
			wantErr: true,
		},
		{
			name: "invalid confidence threshold too low",
			envVars: map[string]string{
				"CONFIDENCE_THRESHOLD": "-10",
			},
			wantErr: true,
		},
		{
			name: "invalid max concurrent clones",
			envVars: map[string]string{
				"MAX_CONCURRENT_CLONES": "0",
			},
			wantErr: true,
		},
		{
			name: "invalid max repo size",
			envVars: map[string]string{
				"MAX_REPO_SIZE_GB": "-1",
			},
			wantErr: true,
		},
		{
			name: "invalid disk limit",
			envVars: map[string]string{
				"DISK_LIMIT_GB": "0",
			},
			wantErr: true,
		},
		{
			name: "invalid active scan window",
			envVars: map[string]string{
				"ACTIVE_SCAN_WINDOW_HOURS": "-1",
			},
			wantErr: true,
		},
		{
			name: "invalid fresh track sleep",
			envVars: map[string]string{
				"FRESH_TRACK_SLEEP_SECONDS": "-5",
			},
			wantErr: true,
		},
		{
			name: "invalid notification rate limit",
			envVars: map[string]string{
				"NOTIFICATION_RATE_LIMIT": "0",
			},
			wantErr: true,
		},
		{
			name: "invalid email confidence threshold",
			envVars: map[string]string{
				"EMAIL_CONFIDENCE_THRESHOLD": "101",
			},
			wantErr: true,
		},
		{
			name: "invalid integer fallback",
			envVars: map[string]string{
				"MAX_CONCURRENT_CLONES": "not-a-number",
			},
			wantErr: false, // falls back to default, which is valid
			validate: func(t *testing.T, cfg *Config) {
				if cfg.MaxConcurrentClones != 10 {
					t.Errorf("MaxConcurrentClones = %d; want 10 (default)", cfg.MaxConcurrentClones)
				}
			},
		},
		{
			name: "invalid float fallback",
			envVars: map[string]string{
				"CONFIDENCE_THRESHOLD": "not-a-number",
			},
			wantErr: false, // falls back to default, which is valid
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ConfidenceThreshold != 25.0 {
					t.Errorf("ConfidenceThreshold = %f; want 25.0 (default)", cfg.ConfidenceThreshold)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars before each test
			keys := []string{
				"GITHUB_TOKEN", "WORKER_ID", "DATABASE_PATH",
				"CONFIDENCE_THRESHOLD", "MAX_CONCURRENT_CLONES", "MAX_REPO_SIZE_GB",
				"DISK_LIMIT_GB", "ACTIVE_SCAN_WINDOW_HOURS", "FRESH_TRACK_SLEEP_SECONDS",
				"NOTIFIER_TYPES", "NOTIFICATION_RATE_LIMIT",
				"EMAIL_CONFIDENCE_THRESHOLD", "SMTP_HOST", "SMTP_PORT",
				"SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM", "SMTP_TO",
			}
			for _, k := range keys {
				os.Unsetenv(k)
			}

			// Set env vars for this test
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestGenerateWorkerID(t *testing.T) {
	id1 := generateWorkerID()
	id2 := generateWorkerID()

	if id1 == "" {
		t.Error("generateWorkerID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateWorkerID() returned duplicate IDs")
	}
}

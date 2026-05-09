package source

import (
	"context"
	"time"
)

// Repository represents a discovered repository from any platform
type Repository struct {
	ID          int64
	Owner       string
	Name        string
	FullName    string
	CloneURL    string
	SizeKB      int
	HTMLURL     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string
}

// Source is the abstraction for repository discovery platforms
// Implement this interface for GitHub, GitLab, Bitbucket, etc.
type Source interface {
	// Name returns the platform name (e.g., "github", "gitlab")
	Name() string

	// DiscoverFresh returns repositories sorted by creation date (newest first)
	// The 'since' parameter filters out repositories created before this time
	DiscoverFresh(ctx context.Context, page, perPage int, since time.Time) ([]Repository, error)

	// DiscoverActive returns recently updated repositories sorted by update date
	// The 'since' parameter filters out repositories updated before this time
	DiscoverActive(ctx context.Context, page, perPage int, since time.Time) ([]Repository, error)

	// TestConnection validates the connection and credentials
	TestConnection(ctx context.Context) error

	// IsAuthenticated returns true if the source has valid authentication
	IsAuthenticated() bool
}

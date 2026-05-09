package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/lucasmacori/sniffy/internal/ratelimiter"
	"github.com/lucasmacori/sniffy/internal/source"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with rate limiting awareness
type Client struct {
	client *github.Client
	token  string
	limiter *ratelimiter.RateLimiter
}

// NewClient creates a new GitHub API client
func NewClient(token string, limiter *ratelimiter.RateLimiter) *Client {
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	return &Client{
		client:  github.NewClient(httpClient),
		token:   token,
		limiter: limiter,
	}
}

// Name returns the platform name
func (c *Client) Name() string {
	return "github"
}

// IsAuthenticated returns true if the client has a token
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// TestConnection validates the connection and credentials
func (c *Client) TestConnection(ctx context.Context) error {
	if c.token != "" {
		// Test authenticated access
		_, resp, err := c.client.Users.Get(ctx, "")
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("authentication failed: invalid or expired token")
			}
			return fmt.Errorf("connection test failed: %w", err)
		}
		return nil
	}

	// Test unauthenticated access with a simple search
	_, resp, err := c.client.Search.Repositories(ctx, "is:public", &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 1},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("rate limit exceeded or API access blocked")
		}
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

// DiscoverFresh returns repositories sorted by creation date (newest first)
func (c *Client) DiscoverFresh(ctx context.Context, page, perPage int, since time.Time) ([]source.Repository, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	// Build query for newly created repositories
	query := "is:public sort:created-desc"
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	// Wait for rate limit token
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	result, resp, err := c.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		c.handleRateLimitError(resp)
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("rate limit exceeded or forbidden: %w", err)
		}
		return nil, fmt.Errorf("search repositories failed: %w", err)
	}

	return c.convertRepositories(result.Repositories, since), nil
}

// DiscoverActive returns recently updated repositories
func (c *Client) DiscoverActive(ctx context.Context, page, perPage int, since time.Time) ([]source.Repository, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}

	// Build query for recently updated repositories
	// Use pushed: qualifier to find repos with recent activity
	sinceStr := since.Format("2006-01-02T15:04:05Z07:00")
	query := fmt.Sprintf("is:public sort:updated-desc pushed:>%s", sinceStr)
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	// Wait for rate limit token
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	result, resp, err := c.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		c.handleRateLimitError(resp)
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("rate limit exceeded or forbidden: %w", err)
		}
		return nil, fmt.Errorf("search repositories failed: %w", err)
	}

	return c.convertRepositories(result.Repositories, time.Time{}), nil
}

// handleRateLimitError checks if the error is due to rate limiting and falls back if needed
func (c *Client) handleRateLimitError(resp *github.Response) {
	if resp == nil {
		return
	}

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		// Check if it's a rate limit or auth issue
		if resp.Rate.Remaining == 0 || resp.StatusCode == http.StatusUnauthorized {
			c.limiter.FallbackToUnauthenticated()
		}
	}

	// Sleep until rate limit reset if we're close to the limit
	if resp.Rate.Remaining < 2 {
		sleepDuration := time.Until(resp.Rate.Reset.Time)
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}
	}
}

// convertRepositories converts GitHub repositories to source.Repository
func (c *Client) convertRepositories(repos []*github.Repository, since time.Time) []source.Repository {
	result := make([]source.Repository, 0, len(repos))
	for _, r := range repos {
		owner := ""
		if r.Owner != nil {
			owner = r.Owner.GetLogin()
		}

		createdAt := time.Time{}
		if r.CreatedAt != nil {
			createdAt = r.CreatedAt.Time
		}

		updatedAt := time.Time{}
		if r.UpdatedAt != nil {
			updatedAt = r.UpdatedAt.Time
		}

		// For Fresh track, filter out repos created before 'since'
		if !since.IsZero() && createdAt.Before(since) {
			continue
		}

		result = append(result, source.Repository{
			ID:          r.GetID(),
			Owner:       owner,
			Name:        r.GetName(),
			FullName:    r.GetFullName(),
			CloneURL:    r.GetCloneURL(),
			SizeKB:      r.GetSize(),
			HTMLURL:     r.GetHTMLURL(),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			Description: r.GetDescription(),
		})
	}
	return result
}

// GetRepositorySize returns the size of a repository in KB
func (c *Client) GetRepositorySize(ctx context.Context, owner, name string) (int, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return 0, err
	}

	repo, resp, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		c.handleRateLimitError(resp)
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return 0, fmt.Errorf("rate limit exceeded or forbidden: %w", err)
		}
		return 0, fmt.Errorf("get repository failed: %w", err)
	}

	return repo.GetSize(), nil
}

// RateLimit returns the current rate limit status
func (c *Client) RateLimit(ctx context.Context) (*github.RateLimits, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	rate, resp, err := c.client.RateLimits(ctx)
	if err != nil {
		c.handleRateLimitError(resp)
		return nil, fmt.Errorf("get rate limit failed: %w", err)
	}
	return rate, nil
}

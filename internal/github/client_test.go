package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/lucasmacori/sniffy/internal/source"
)

func TestClient_Name(t *testing.T) {
	c := NewClient("", nil)
	if c.Name() != "github" {
		t.Errorf("Name() = %q; want github", c.Name())
	}
}

func TestClient_IsAuthenticated(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		c := NewClient("ghp_test", nil)
		if !c.IsAuthenticated() {
			t.Error("IsAuthenticated() = false; want true")
		}
	})

	t.Run("without token", func(t *testing.T) {
		c := NewClient("", nil)
		if c.IsAuthenticated() {
			t.Error("IsAuthenticated() = true; want false")
		}
	})
}

func TestClient_convertRepositories(t *testing.T) {
	c := NewClient("", nil)

	now := time.Now()
	repo1 := &github.Repository{
		ID:          github.Int64(1),
		Name:        github.String("repo1"),
		FullName:    github.String("owner/repo1"),
		CloneURL:    github.String("https://github.com/owner/repo1.git"),
		Size:        github.Int(100),
		HTMLURL:     github.String("https://github.com/owner/repo1"),
		CreatedAt:   &github.Timestamp{Time: now},
		UpdatedAt:   &github.Timestamp{Time: now},
		Description: github.String("A test repo"),
		Owner: &github.User{
			Login: github.String("owner"),
		},
	}

	repo2 := &github.Repository{
		ID:       github.Int64(2),
		Name:     github.String("repo2"),
		FullName: github.String("owner/repo2"),
		Owner: &github.User{
			Login: github.String("owner"),
		},
	}

	t.Run("convert with filtering", func(t *testing.T) {
		// Filter out repos created before 'since'
		since := now.Add(time.Hour)
		result := c.convertRepositories([]*github.Repository{repo1, repo2}, since)

		if len(result) != 0 {
			t.Errorf("len(result) = %d; want 0 (both created before since)", len(result))
		}
	})

	t.Run("convert without filtering", func(t *testing.T) {
		result := c.convertRepositories([]*github.Repository{repo1, repo2}, time.Time{})

		if len(result) != 2 {
			t.Fatalf("len(result) = %d; want 2", len(result))
		}

		r := result[0]
		if r.ID != 1 {
			t.Errorf("ID = %d; want 1", r.ID)
		}
		if r.Owner != "owner" {
			t.Errorf("Owner = %q; want owner", r.Owner)
		}
		if r.Name != "repo1" {
			t.Errorf("Name = %q; want repo1", r.Name)
		}
		if r.FullName != "owner/repo1" {
			t.Errorf("FullName = %q; want owner/repo1", r.FullName)
		}
		if r.CloneURL != "https://github.com/owner/repo1.git" {
			t.Errorf("CloneURL = %q; want https://github.com/owner/repo1.git", r.CloneURL)
		}
		if r.HTMLURL != "https://github.com/owner/repo1" {
			t.Errorf("HTMLURL = %q; want https://github.com/owner/repo1", r.HTMLURL)
		}
		if r.Description != "A test repo" {
			t.Errorf("Description = %q; want A test repo", r.Description)
		}
		if r.SizeKB != 100 {
			t.Errorf("SizeKB = %d; want 100", r.SizeKB)
		}
	})

	t.Run("convert nil fields", func(t *testing.T) {
		repo := &github.Repository{
			ID:   github.Int64(3),
			Name: github.String("repo3"),
		}
		result := c.convertRepositories([]*github.Repository{repo}, time.Time{})

		if len(result) != 1 {
			t.Fatalf("len(result) = %d; want 1", len(result))
		}
		if result[0].Owner != "" {
			t.Errorf("Owner = %q; want empty", result[0].Owner)
		}
		if !result[0].CreatedAt.IsZero() {
			t.Error("CreatedAt should be zero")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result := c.convertRepositories([]*github.Repository{}, time.Time{})
		if len(result) != 0 {
			t.Errorf("len(result) = %d; want 0", len(result))
		}
	})
}

func TestClient_convertRepositories_WithSince(t *testing.T) {
	c := NewClient("", nil)

	oldTime := time.Now().Add(-24 * time.Hour)
	newTime := time.Now()

	oldRepo := &github.Repository{
		ID:        github.Int64(1),
		Name:      github.String("old"),
		CreatedAt: &github.Timestamp{Time: oldTime},
		Owner:     &github.User{Login: github.String("owner")},
	}
	newRepo := &github.Repository{
		ID:        github.Int64(2),
		Name:      github.String("new"),
		CreatedAt: &github.Timestamp{Time: newTime},
		Owner:     &github.User{Login: github.String("owner")},
	}

	// Only newRepo should pass the filter
	since := oldTime.Add(time.Hour)
	result := c.convertRepositories([]*github.Repository{oldRepo, newRepo}, since)

	if len(result) != 1 {
		t.Fatalf("len(result) = %d; want 1", len(result))
	}
	if result[0].Name != "new" {
		t.Errorf("Name = %q; want new", result[0].Name)
	}
}

func TestRepository_Struct(t *testing.T) {
	// Basic sanity check for source.Repository
	r := source.Repository{
		ID:          1,
		Owner:       "owner",
		Name:        "repo",
		FullName:    "owner/repo",
		CloneURL:    "https://github.com/owner/repo.git",
		SizeKB:      100,
		HTMLURL:     "https://github.com/owner/repo",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "desc",
	}

	if r.FullName != "owner/repo" {
		t.Errorf("FullName = %q; want owner/repo", r.FullName)
	}
}

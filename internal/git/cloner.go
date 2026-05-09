package git

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lucasmacori/sniffy/internal/models"
)

// Cloner handles git repository cloning with concurrency and resource limits
type Cloner struct {
	maxConcurrent int
	maxRepoSizeGB float64
	diskLimitGB   float64

	semaphore chan struct{}
	mu        sync.Mutex
	activeGB  float64
}

// NewCloner creates a new git cloner with resource limits
func NewCloner(maxConcurrent int, maxRepoSizeGB, diskLimitGB float64) *Cloner {
	return &Cloner{
		maxConcurrent: maxConcurrent,
		maxRepoSizeGB: maxRepoSizeGB,
		diskLimitGB:   diskLimitGB,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// CloneResult contains information about a cloned repository
type CloneResult struct {
	Path      string
	RepoOwner string
	RepoName  string
	CloneURL  string
	SizeMB    float64
}

// Clone clones a repository to a temporary directory
func (c *Cloner) Clone(ctx context.Context, cloneURL, owner, name string) (*CloneResult, error) {
	// Acquire semaphore
	select {
	case c.semaphore <- struct{}{}:
		// Acquired
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-c.semaphore }()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("sniffy-%s-%s-*", owner, name))
	if err != nil {
		return nil, fmt.Errorf("create temp dir failed: %w", err)
	}

	repoPath := filepath.Join(tempDir, "repo")

	// Clone the repository (full clone with all refs)
	cmd := exec.CommandContext(ctx, "git", "clone", "--mirror", cloneURL, repoPath)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("git clone failed: %w\noutput: %s", err, string(output))
	}

	// Get repository size
	sizeBytes, err := c.dirSize(repoPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("calculate repo size failed: %w", err)
	}

	sizeMB := float64(sizeBytes) / (1024 * 1024)
	sizeGB := sizeMB / 1024

	// Check repo size limit
	if sizeGB > c.maxRepoSizeGB {
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("repository size %.2f GB exceeds limit %.2f GB", sizeGB, c.maxRepoSizeGB)
	}

	// Check disk limit
	c.mu.Lock()
	if c.activeGB+sizeGB > c.diskLimitGB {
		c.mu.Unlock()
		_ = os.RemoveAll(tempDir)
		return nil, fmt.Errorf("disk limit would be exceeded: active %.2f GB + new %.2f GB > limit %.2f GB",
			c.activeGB, sizeGB, c.diskLimitGB)
	}
	c.activeGB += sizeGB
	c.mu.Unlock()

	return &CloneResult{
		Path:      repoPath,
		RepoOwner: owner,
		RepoName:  name,
		CloneURL:  cloneURL,
		SizeMB:    sizeMB,
	}, nil
}

// Cleanup removes the cloned repository and frees resources
func (c *Cloner) Cleanup(result *CloneResult) error {
	if result == nil || result.Path == "" {
		return nil
	}

	// Get parent temp directory
	parent := filepath.Dir(result.Path)

	// Calculate size to subtract
	sizeBytes, err := c.dirSize(result.Path)
	if err == nil {
		sizeGB := float64(sizeBytes) / (1024 * 1024 * 1024)
		c.mu.Lock()
		c.activeGB = math.Max(0, c.activeGB-sizeGB)
		c.mu.Unlock()
	}

	// Remove the directory
	if err := os.RemoveAll(parent); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}

// dirSize calculates the total size of a directory in bytes
func (c *Cloner) dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// Inspector handles git repository inspection
type Inspector struct{}

// NewInspector creates a new git inspector
func NewInspector() *Inspector {
	return &Inspector{}
}



// GetAllCommits returns all commits from a repository including reflog
func (i *Inspector) GetAllCommits(ctx context.Context, repoPath string) ([]models.CommitInfo, error) {
	// First, let's get all refs including reflog
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "reflog", "show", "--all", "--format=%H|%an|%ae|%at|%s")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without reflog if it fails
		return i.getCommitsFromLog(ctx, repoPath)
	}

	commits := make([]models.CommitInfo, 0)
	seen := make(map[string]bool)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		hash := parts[0]
		if seen[hash] {
			continue
		}
		seen[hash] = true

		commits = append(commits, models.CommitInfo{
			Hash:      hash,
			Author:    parts[1],
			Email:     parts[2],
			Timestamp: parts[3],
			Message:   parts[4],
		})
	}

	return commits, nil
}

// getCommitsFromLog gets commits using git log as fallback
func (i *Inspector) getCommitsFromLog(ctx context.Context, repoPath string) ([]models.CommitInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "--all", "--format=%H|%an|%ae|%at|%s")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w\noutput: %s", err, string(output))
	}

	commits := make([]models.CommitInfo, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		commits = append(commits, models.CommitInfo{
			Hash:      parts[0],
			Author:    parts[1],
			Email:     parts[2],
			Timestamp: parts[3],
			Message:   parts[4],
		})
	}

	return commits, nil
}

// GetFileContentAtCommit returns the content of a file at a specific commit
func (i *Inspector) GetFileContentAtCommit(ctx context.Context, repoPath, commitHash, filePath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", fmt.Sprintf("%s:%s", commitHash, filePath))
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

// GetDiff returns the diff for a commit
func (i *Inspector) GetDiff(ctx context.Context, repoPath, commitHash string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "show", "--patch", commitHash)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show diff failed: %w", err)
	}
	return string(output), nil
}

// GetChangedFiles returns files changed in a commit
func (i *Inspector) GetChangedFiles(ctx context.Context, repoPath, commitHash string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff-tree", "--no-commit-id", "--name-only", "-r", commitHash)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree failed: %w", err)
	}

	files := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// GetWorktreeFiles returns all files in the current worktree
func (i *Inspector) GetWorktreeFiles(repoPath string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip git internal files
		if strings.Contains(path, "/.git/") {
			return nil
		}
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}
		files = append(files, relPath)
		return nil
	})
	return files, err
}

// GetFileContent reads a file from the worktree
func (i *Inspector) GetFileContent(repoPath, filePath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(repoPath, filePath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// IsBinary checks if a file is binary
func (i *Inspector) IsBinary(repoPath, filePath string) bool {
	content, err := os.ReadFile(filepath.Join(repoPath, filePath))
	if err != nil {
		return true
	}
	// Simple binary check: look for null bytes
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	return false
}

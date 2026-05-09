package detector

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/lucasmacori/sniffy/internal/models"
)

// Finding represents a detected credential leak
type Finding struct {
	Repository   string
	CommitHash   string
	CommitAuthor string
	CommitEmail  string
	FilePath     string
	LineNumber   int
	SecretType   string
	SecretValue  string
	Confidence   float64
	Source       string // "worktree", "commit", "reflog"
	HTMLURL      string
}

// Detector is the interface for credential detection strategies
type Detector interface {
	Name() string
	Detect(ctx context.Context, repoPath string, inspector GitInspector) ([]Finding, error)
}

// GitInspector provides methods to inspect git repositories
type GitInspector interface {
	GetAllCommits(ctx context.Context, repoPath string) ([]models.CommitInfo, error)
	GetFileContentAtCommit(ctx context.Context, repoPath, commitHash, filePath string) (string, error)
	GetDiff(ctx context.Context, repoPath, commitHash string) (string, error)
	GetChangedFiles(ctx context.Context, repoPath, commitHash string) ([]string, error)
	GetWorktreeFiles(repoPath string) ([]string, error)
	GetFileContent(repoPath, filePath string) (string, error)
	IsBinary(repoPath, filePath string) bool
}

// CompositeDetector combines multiple detection strategies
type CompositeDetector struct {
	detectors []Detector
	threshold float64
}

// NewCompositeDetector creates a new composite detector
func NewCompositeDetector(threshold float64, detectors ...Detector) *CompositeDetector {
	return &CompositeDetector{
		detectors: detectors,
		threshold: threshold,
	}
}

// Name returns the detector name
func (c *CompositeDetector) Name() string {
	return "composite"
}

// Detect runs all detectors and filters by confidence threshold
func (c *CompositeDetector) Detect(ctx context.Context, repoPath string, inspector GitInspector) ([]Finding, error) {
	var allFindings []Finding
	seen := make(map[string]bool)

	for _, d := range c.detectors {
		findings, err := d.Detect(ctx, repoPath, inspector)
		if err != nil {
			// Log error but continue with other detectors
			continue
		}

		for _, f := range findings {
			if f.Confidence >= c.threshold {
				key := fmt.Sprintf("%s|%s|%s|%d", f.Repository, f.CommitHash, f.FilePath, f.LineNumber)
				if !seen[key] {
					seen[key] = true
					allFindings = append(allFindings, f)
				}
			}
		}
	}

	return allFindings, nil
}

// RegexDetector uses regex patterns to detect credentials
type RegexDetector struct {
	patterns map[string]*regexp.Regexp
}

// NewRegexDetector creates a new regex detector with common secret patterns
func NewRegexDetector() *RegexDetector {
	return &RegexDetector{
		patterns: map[string]*regexp.Regexp{
			"AWS Access Key":       regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`),
			"AWS Secret Key":       regexp.MustCompile(`(?i)(aws.{0,20}secret.{0,20}['"\"][0-9a-zA-Z\/+]{40}['"\"])`),
			"GitHub Token":         regexp.MustCompile(`(?i)(gh[pousr]_[A-Za-z0-9_]{36,})`),
			"GitHub Classic Token": regexp.MustCompile(`(?i)(ghp_[A-Za-z0-9]{36})`),
			"Slack Token":          regexp.MustCompile(`(?i)(xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*)`),
			"Private Key":          regexp.MustCompile(`(?i)(-----BEGIN (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----)`),
			"API Key Generic":      regexp.MustCompile(`(?i)(api[_-]?key['"\"\s]*[:=]['"\"\s]*[a-zA-Z0-9_\-]{16,})`),
			"Secret Generic":       regexp.MustCompile(`(?i)(secret['"\"\s]*[:=]['"\"\s]*[a-zA-Z0-9_\-]{8,})`),
			"Password Assignment":  regexp.MustCompile(`(?i)(password['"\"\s]*[:=]['"\"\s]*[^\s]{8,})`),
			"Bearer Token":         regexp.MustCompile(`(?i)(bearer\s+[a-zA-Z0-9_\-\.=]{20,})`),
			"Basic Auth":           regexp.MustCompile(`(?i)(basic\s+[a-zA-Z0-9+/]{20,}=*)`),
			"JWT Token":            regexp.MustCompile(`(?i)(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`),
		},
	}
}

// Name returns the detector name
func (r *RegexDetector) Name() string {
	return "regex"
}

// Detect scans files for regex patterns
func (r *RegexDetector) Detect(ctx context.Context, repoPath string, inspector GitInspector) ([]Finding, error) {
	var findings []Finding

	files, err := inspector.GetWorktreeFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("get worktree files failed: %w", err)
	}

	for _, file := range files {
		if inspector.IsBinary(repoPath, file) {
			continue
		}

		content, err := inspector.GetFileContent(repoPath, file)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			select {
			case <-ctx.Done():
				return findings, ctx.Err()
			default:
			}

			for secretType, pattern := range r.patterns {
				matches := pattern.FindAllString(line, -1)
				for _, match := range matches {
					confidence := r.calculateConfidence(match, line, file)
					findings = append(findings, Finding{
						Repository: repoPath,
						FilePath:   file,
						LineNumber: lineNum + 1,
						SecretType: secretType,
						SecretValue: match,
						Confidence: confidence,
						Source:     "worktree",
					})
				}
			}
		}
	}

	return findings, nil
}

// calculateConfidence calculates a confidence score based on match characteristics
func (r *RegexDetector) calculateConfidence(match, line, file string) float64 {
	score := 50.0 // Base score

	// Entropy boost
	entropy := calculateEntropy(match)
	if entropy > 4.5 {
		score += 20
	} else if entropy > 3.5 {
		score += 10
	}

	// File type indicators
	lowerFile := strings.ToLower(file)
	if strings.Contains(lowerFile, ".env") ||
		strings.Contains(lowerFile, "config") ||
		strings.Contains(lowerFile, "secret") ||
		strings.Contains(lowerFile, "credential") {
		score += 15
	}

	// Line context indicators
	lowerLine := strings.ToLower(line)
	if strings.Contains(lowerLine, "secret") ||
		strings.Contains(lowerLine, "token") ||
		strings.Contains(lowerLine, "key") ||
		strings.Contains(lowerLine, "password") ||
		strings.Contains(lowerLine, "auth") {
		score += 10
	}

	// Length check - very short or very long might be false positives
	matchLen := len(match)
	if matchLen < 8 {
		score -= 20
	} else if matchLen > 200 {
		score -= 10
	}

	// High entropy and reasonable length is strong indicator
	if entropy > 4.0 && matchLen >= 16 {
		score += 10
	}

	return math.Min(100, math.Max(0, score))
}

// EntropyDetector uses entropy analysis to find high-entropy strings
type EntropyDetector struct {
	minEntropy float64
	minLength  int
}

// NewEntropyDetector creates a new entropy detector
func NewEntropyDetector() *EntropyDetector {
	return &EntropyDetector{
		minEntropy: 4.5,
		minLength:  20,
	}
}

// Name returns the detector name
func (e *EntropyDetector) Name() string {
	return "entropy"
}

// Detect finds high-entropy strings that might be secrets
func (e *EntropyDetector) Detect(ctx context.Context, repoPath string, inspector GitInspector) ([]Finding, error) {
	var findings []Finding

	files, err := inspector.GetWorktreeFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("get worktree files failed: %w", err)
	}

	// Common variable names that might contain secrets
	secretIndicators := []string{"api_key", "apikey", "secret", "token", "password", "passwd", "auth"}

	for _, file := range files {
		if inspector.IsBinary(repoPath, file) {
			continue
		}

		content, err := inspector.GetFileContent(repoPath, file)
		if err != nil {
			continue
		}

		lines := strings.Split(content, "\n")
		for lineNum, line := range lines {
			select {
			case <-ctx.Done():
				return findings, ctx.Err()
			default:
			}

			// Look for potential secret assignments
			for _, indicator := range secretIndicators {
				if strings.Contains(strings.ToLower(line), indicator) {
					// Extract potential secret value
					secret := e.extractPotentialSecret(line)
					if secret != "" && len(secret) >= e.minLength {
						entropy := calculateEntropy(secret)
						if entropy >= e.minEntropy {
							confidence := math.Min(100, (entropy/6.0)*70+30)
							findings = append(findings, Finding{
								Repository:  repoPath,
								FilePath:    file,
								LineNumber:  lineNum + 1,
								SecretType:  "High Entropy String",
								SecretValue: secret,
								Confidence:  confidence,
								Source:      "worktree",
							})
						}
					}
				}
			}
		}
	}

	return findings, nil
}

// extractPotentialSecret tries to extract a secret value from a line
func (e *EntropyDetector) extractPotentialSecret(line string) string {
	// Look for quoted strings after assignment
	patterns := []string{
		`[=:]\s*["']([^"']{10,})["']`,
		`[=:]\s*([a-zA-Z0-9_\-+/=]{20,})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// GitHistoryDetector scans git history and reflog for secrets
type GitHistoryDetector struct {
	regexDetector *RegexDetector
}

// NewGitHistoryDetector creates a new git history detector
func NewGitHistoryDetector() *GitHistoryDetector {
	return &GitHistoryDetector{
		regexDetector: NewRegexDetector(),
	}
}

// Name returns the detector name
func (g *GitHistoryDetector) Name() string {
	return "git-history"
}

// Detect scans git history for secrets
func (g *GitHistoryDetector) Detect(ctx context.Context, repoPath string, inspector GitInspector) ([]Finding, error) {
	var findings []Finding

	commits, err := inspector.GetAllCommits(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("get all commits failed: %w", err)
	}

	for _, commit := range commits {
		select {
		case <-ctx.Done():
			return findings, ctx.Err()
		default:
		}

		// Get diff for this commit
		diff, err := inspector.GetDiff(ctx, repoPath, commit.Hash)
		if err != nil {
			continue
		}

		lines := strings.Split(diff, "\n")
		filePath := ""
		lineNumber := 0

		for _, line := range lines {
			// Track file path from diff header
			if strings.HasPrefix(line, "+++") {
				parts := strings.SplitN(line, "\t", 2)
				if len(parts) > 0 {
					filePath = strings.TrimPrefix(parts[0], "+++ b/")
					if filePath == "+++ /dev/null" {
						filePath = ""
					}
				}
				continue
			}

			// Track line numbers
			if strings.HasPrefix(line, "@@") {
				// Extract line number from hunk header
				re := regexp.MustCompile(`\+([0-9]+)`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					_, _ = fmt.Sscanf(matches[1], "%d", &lineNumber)
				}
				continue
			}

			// Only check added lines (starting with +)
			if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
				continue
			}

			cleanLine := strings.TrimPrefix(line, "+")
			lineNumber++

			for secretType, pattern := range g.regexDetector.patterns {
				matches := pattern.FindAllString(cleanLine, -1)
				for _, match := range matches {
					confidence := g.regexDetector.calculateConfidence(match, cleanLine, filePath)
					confidence += 5 // Slight boost for being in a diff (newly added)
					confidence = math.Min(100, confidence)

					findings = append(findings, Finding{
						Repository:   repoPath,
						CommitHash:   commit.Hash,
						CommitAuthor: commit.Author,
						CommitEmail:  commit.Email,
						FilePath:     filePath,
						LineNumber:   lineNumber,
						SecretType:   secretType,
						SecretValue:  match,
						Confidence:   confidence,
						Source:       "commit",
					})
				}
			}
		}
	}

	return findings, nil
}

// calculateEntropy calculates Shannon entropy of a string
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}

	var entropy float64
	length := float64(len(s))
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

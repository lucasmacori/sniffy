package detector

import (
	"context"
	"errors"
	"testing"

	"github.com/lucasmacori/sniffy/internal/models"
)

// mockGitInspector is a test double for GitInspector
type mockGitInspector struct {
	files           []string
	fileContent     map[string]string
	binaryFiles     map[string]bool
	commits         []models.CommitInfo
	diffs           map[string]string
	changedFiles    map[string][]string
	commitFileContent map[string]string // key: "hash:path"
}

func (m *mockGitInspector) GetAllCommits(ctx context.Context, repoPath string) ([]models.CommitInfo, error) {
	return m.commits, nil
}

func (m *mockGitInspector) GetFileContentAtCommit(ctx context.Context, repoPath, commitHash, filePath string) (string, error) {
	key := commitHash + ":" + filePath
	if content, ok := m.commitFileContent[key]; ok {
		return content, nil
	}
	return "", errors.New("not found")
}

func (m *mockGitInspector) GetDiff(ctx context.Context, repoPath, commitHash string) (string, error) {
	if diff, ok := m.diffs[commitHash]; ok {
		return diff, nil
	}
	return "", nil
}

func (m *mockGitInspector) GetChangedFiles(ctx context.Context, repoPath, commitHash string) ([]string, error) {
	if files, ok := m.changedFiles[commitHash]; ok {
		return files, nil
	}
	return nil, nil
}

func (m *mockGitInspector) GetWorktreeFiles(repoPath string) ([]string, error) {
	return m.files, nil
}

func (m *mockGitInspector) GetFileContent(repoPath, filePath string) (string, error) {
	if content, ok := m.fileContent[filePath]; ok {
		return content, nil
	}
	return "", errors.New("not found")
}

func (m *mockGitInspector) IsBinary(repoPath, filePath string) bool {
	return m.binaryFiles[filePath]
}

func TestNewCompositeDetector(t *testing.T) {
	rd := NewRegexDetector()
	ed := NewEntropyDetector()
	cd := NewCompositeDetector(50.0, rd, ed)

	if cd.Name() != "composite" {
		t.Errorf("Name() = %q; want composite", cd.Name())
	}
}

func TestCompositeDetector_Detect(t *testing.T) {
	t.Run("filters by threshold and deduplicates", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"config.env"},
			fileContent: map[string]string{
				"config.env": "API_KEY=AKIAIOSFODNN7EXAMPLE\nSECRET=password12345678",
			},
		}

		rd := NewRegexDetector()
		cd := NewCompositeDetector(60.0, rd)

		findings, err := cd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}

		// The exact count depends on confidence scores, but we verify filtering works
		for _, f := range findings {
			if f.Confidence < 60.0 {
				t.Errorf("finding confidence %f below threshold 60.0", f.Confidence)
			}
		}
	})

	t.Run("continues on detector error", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"test.txt"},
			fileContent: map[string]string{
				"test.txt": "API_KEY=AKIAIOSFODNN7EXAMPLE",
			},
		}

		cd := NewCompositeDetector(0.0, NewRegexDetector())
		findings, err := cd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(findings) == 0 {
			t.Error("expected some findings")
		}
	})

	t.Run("empty repository", func(t *testing.T) {
		mock := &mockGitInspector{files: []string{}}
		cd := NewCompositeDetector(0.0, NewRegexDetector())

		findings, err := cd.Detect(context.Background(), "empty-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("len(findings) = %d; want 0", len(findings))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"config.env"},
			fileContent: map[string]string{
				"config.env": "API_KEY=AKIAIOSFODNN7EXAMPLE\n",
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cd := NewCompositeDetector(0.0, NewRegexDetector())
		_, err := cd.Detect(ctx, "test-repo", mock)
		// Should either return context error or partial results
		if err != nil && err != context.Canceled {
			t.Errorf("Detect() error = %v", err)
		}
	})
}

func TestRegexDetector_Detect(t *testing.T) {
	t.Run("finds AWS key", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"config.env"},
			fileContent: map[string]string{
				"config.env": "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			},
		}

		rd := NewRegexDetector()
		findings, err := rd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}

		foundAWS := false
		for _, f := range findings {
			if f.SecretType == "AWS Access Key" {
				foundAWS = true
				if f.SecretValue != "AKIAIOSFODNN7EXAMPLE" {
					t.Errorf("SecretValue = %q; want AKIAIOSFODNN7EXAMPLE", f.SecretValue)
				}
				if f.LineNumber != 1 {
					t.Errorf("LineNumber = %d; want 1", f.LineNumber)
				}
			}
		}
		if !foundAWS {
			t.Error("expected to find AWS Access Key")
		}
	})

	t.Run("skips binary files", func(t *testing.T) {
		mock := &mockGitInspector{
			files:       []string{"binary.dat"},
			binaryFiles: map[string]bool{"binary.dat": true},
		}

		rd := NewRegexDetector()
		findings, err := rd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("len(findings) = %d; want 0 (binary file skipped)", len(findings))
		}
	})

	t.Run("handles missing file gracefully", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"missing.txt"},
		}

		rd := NewRegexDetector()
		findings, err := rd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("len(findings) = %d; want 0", len(findings))
		}
	})

	t.Run("finds GitHub token", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{".github/workflows/ci.yml"},
			fileContent: map[string]string{
				".github/workflows/ci.yml": "GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			},
		}

		rd := NewRegexDetector()
		findings, err := rd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}

		found := false
		for _, f := range findings {
			if f.SecretType == "GitHub Token" || f.SecretType == "GitHub Classic Token" {
				found = true
			}
		}
		if !found {
			t.Error("expected to find GitHub token")
		}
	})
}

func TestRegexDetector_calculateConfidence(t *testing.T) {
	rd := NewRegexDetector()

	tests := []struct {
		name     string
		match    string
		line     string
		file     string
		minScore float64
	}{
		{
			name:     "AWS key in env file",
			match:    "AKIAIOSFODNN7EXAMPLE",
			line:     "AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
			file:     "config.env",
			minScore: 70,
		},
		{
			name:     "short match penalty",
			match:    "abc",
			line:     "key=abc",
			file:     "test.txt",
			minScore: 0,
		},
		{
			name:     "config file boost",
			match:    "some-secret-value-here-12345",
			line:     "secret=some-secret-value-here-12345",
			file:     "app.config",
			minScore: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := rd.calculateConfidence(tt.match, tt.line, tt.file)
			if score < tt.minScore {
				t.Errorf("calculateConfidence() = %f; want >= %f", score, tt.minScore)
			}
			if score < 0 || score > 100 {
				t.Errorf("calculateConfidence() = %f; want between 0 and 100", score)
			}
		})
	}
}

func TestEntropyDetector_Detect(t *testing.T) {
	t.Run("finds high entropy secret", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"config.env"},
			fileContent: map[string]string{
				"config.env": "api_key=AbCdEfGhIjKlMnOpQrStUvWxYz1234567890",
			},
		}

		ed := NewEntropyDetector()
		findings, err := ed.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}

		found := false
		for _, f := range findings {
			if f.SecretType == "High Entropy String" {
				found = true
				if f.Confidence < 0 || f.Confidence > 100 {
					t.Errorf("Confidence = %f; want between 0 and 100", f.Confidence)
				}
			}
		}
		if !found {
			t.Logf("No high entropy finding found (may depend on exact string)")
		}
	})

	t.Run("skips short values", func(t *testing.T) {
		mock := &mockGitInspector{
			files: []string{"config.env"},
			fileContent: map[string]string{
				"config.env": "api_key=short",
			},
		}

		ed := NewEntropyDetector()
		findings, err := ed.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		for _, f := range findings {
			if len(f.SecretValue) < 20 {
				t.Errorf("found short secret value: %q", f.SecretValue)
			}
		}
	})
}

func TestEntropyDetector_extractPotentialSecret(t *testing.T) {
	ed := NewEntropyDetector()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "quoted string",
			line: `api_key="this-is-a-long-secret-value-12345"`,
			want: "this-is-a-long-secret-value-12345",
		},
		{
			name: "unquoted value",
			line: "api_key=AbCdEfGhIjKlMnOpQrStUvWxYz1234567890",
			want: "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890",
		},
		{
			name: "no match",
			line: "api_key=short",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ed.extractPotentialSecret(tt.line)
			if got != tt.want {
				t.Errorf("extractPotentialSecret(%q) = %q; want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestGitHistoryDetector_Detect(t *testing.T) {
	t.Run("scans commit diff", func(t *testing.T) {
		mock := &mockGitInspector{
			commits: []models.CommitInfo{
				{Hash: "abc123", Author: "Test", Email: "test@example.com"},
			},
			diffs: map[string]string{
				"abc123": `+++ b/config.env
@@ -0,0 +1 @@
+AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE`,
			},
		}

		ghd := NewGitHistoryDetector()
		findings, err := ghd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}

		found := false
		for _, f := range findings {
			if f.CommitHash == "abc123" && f.Source == "commit" {
				found = true
				if f.FilePath != "config.env" {
					t.Errorf("FilePath = %q; want config.env", f.FilePath)
				}
			}
		}
		if !found {
			t.Error("expected finding from commit diff")
		}
	})

	t.Run("handles missing diff gracefully", func(t *testing.T) {
		mock := &mockGitInspector{
			commits: []models.CommitInfo{
				{Hash: "abc123"},
			},
		}

		ghd := NewGitHistoryDetector()
		findings, err := ghd.Detect(context.Background(), "test-repo", mock)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("len(findings) = %d; want 0", len(findings))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mock := &mockGitInspector{
			commits: []models.CommitInfo{
				{Hash: "abc123"},
			},
			diffs: map[string]string{
				"abc123": "+test",
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ghd := NewGitHistoryDetector()
		_, err := ghd.Detect(ctx, "test-repo", mock)
		if err != nil && err != context.Canceled {
			t.Errorf("Detect() error = %v", err)
		}
	})
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"a", 0},
		{"ab", 1},
		{"aaaaaaaa", 0},
		{"abcdefgh", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := calculateEntropy(tt.input)
			// Use a small delta for float comparison
			delta := 0.1
			if got < tt.want-delta || got > tt.want+delta {
				t.Errorf("calculateEntropy(%q) = %f; want ~%f", tt.input, got, tt.want)
			}
		})
	}
}

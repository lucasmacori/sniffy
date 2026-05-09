package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewCloner(t *testing.T) {
	c := NewCloner(5, 1.5, 20)
	if c.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d; want 5", c.maxConcurrent)
	}
	if c.maxRepoSizeGB != 1.5 {
		t.Errorf("maxRepoSizeGB = %f; want 1.5", c.maxRepoSizeGB)
	}
	if c.diskLimitGB != 20 {
		t.Errorf("diskLimitGB = %f; want 20", c.diskLimitGB)
	}
	if cap(c.semaphore) != 5 {
		t.Errorf("semaphore capacity = %d; want 5", cap(c.semaphore))
	}
}

func TestCloner_Cleanup(t *testing.T) {
	c := NewCloner(1, 10, 100)

	t.Run("nil result", func(t *testing.T) {
		err := c.Cleanup(nil)
		if err != nil {
			t.Fatalf("Cleanup(nil) error = %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		err := c.Cleanup(&CloneResult{})
		if err != nil {
			t.Fatalf("Cleanup(empty) error = %v", err)
		}
	})

	t.Run("removes directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		repoPath := filepath.Join(tmpDir, "repo")
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoPath, "test.txt"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		result := &CloneResult{Path: repoPath}
		err := c.Cleanup(result)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		_, err = os.Stat(tmpDir)
		if !os.IsNotExist(err) {
			t.Error("expected temp directory to be removed")
		}
	})
}

func TestCloner_dirSize(t *testing.T) {
	c := NewCloner(1, 10, 100)
	tmpDir := t.TempDir()

	// Create some files
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "sub", "c.txt"), []byte("!"), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := c.dirSize(tmpDir)
	if err != nil {
		t.Fatalf("dirSize() error = %v", err)
	}
	if size != 11 { // "hello" + "world" + "!" = 5 + 5 + 1
		t.Errorf("dirSize() = %d; want 11", size)
	}
}

func TestInspector_GetWorktreeFiles(t *testing.T) {
	i := NewInspector()
	tmpDir := t.TempDir()

	// Create files
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("git"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".hidden"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden", "secret"), []byte("s"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := i.GetWorktreeFiles(tmpDir)
	if err != nil {
		t.Fatalf("GetWorktreeFiles() error = %v", err)
	}

	expected := map[string]bool{"a.txt": true, "b.go": true}
	got := make(map[string]bool)
	for _, f := range files {
		got[f] = true
	}

	for name := range expected {
		if !got[name] {
			t.Errorf("expected %s in files", name)
		}
	}
	if got[".git/config"] {
		t.Error("expected .git files to be excluded")
	}
	if got[".hidden/secret"] {
		t.Error("expected hidden dirs to be skipped")
	}
}

func TestInspector_GetFileContent(t *testing.T) {
	i := NewInspector()
	tmpDir := t.TempDir()
	content := []byte("hello world")
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := i.GetFileContent(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}
	if got != "hello world" {
		t.Errorf("GetFileContent() = %q; want hello world", got)
	}

	_, err = i.GetFileContent(tmpDir, "missing.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestInspector_IsBinary(t *testing.T) {
	i := NewInspector()
	tmpDir := t.TempDir()

	// Text file
	if err := os.WriteFile(filepath.Join(tmpDir, "text.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if i.IsBinary(tmpDir, "text.txt") {
		t.Error("IsBinary(text.txt) = true; want false")
	}

	// Binary file with null byte
	if err := os.WriteFile(filepath.Join(tmpDir, "binary.dat"), []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatal(err)
	}
	if !i.IsBinary(tmpDir, "binary.dat") {
		t.Error("IsBinary(binary.dat) = false; want true")
	}

	// Missing file
	if !i.IsBinary(tmpDir, "missing.dat") {
		t.Error("IsBinary(missing.dat) = false; want true (returns true on error)")
	}
}

func TestInspector_GetAllCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	i := NewInspector()
	tmpDir := t.TempDir()

	// Initialize git repo
	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}

	// Create a commit
	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "first").Run(); err != nil {
		t.Fatal(err)
	}

	commits, err := i.GetAllCommits(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("GetAllCommits() error = %v", err)
	}
	if len(commits) == 0 {
		t.Error("expected at least one commit")
	}
	if commits[0].Author != "Test" {
		t.Errorf("Author = %q; want Test", commits[0].Author)
	}
}

func TestInspector_GetDiff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	i := NewInspector()
	tmpDir := t.TempDir()

	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "first").Run(); err != nil {
		t.Fatal(err)
	}

	// Get commit hash
	out, _ := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").Output()
	hash := string(out)
	hash = hash[:len(hash)-1] // trim newline

	diff, err := i.GetDiff(context.Background(), tmpDir, hash)
	if err != nil {
		t.Fatalf("GetDiff() error = %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestInspector_GetChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	i := NewInspector()
	tmpDir := t.TempDir()

	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "first").Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "second").Run(); err != nil {
		t.Fatal(err)
	}

	out, _ := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").Output()
	hash := string(out)
	hash = hash[:len(hash)-1]

	files, err := i.GetChangedFiles(context.Background(), tmpDir, hash)
	if err != nil {
		t.Fatalf("GetChangedFiles() error = %v", err)
	}
	found := false
	for _, f := range files {
		if f == "b.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected b.txt in changed files, got %v", files)
	}
}

func TestInspector_GetFileContentAtCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	i := NewInspector()
	tmpDir := t.TempDir()

	if err := exec.Command("git", "init", tmpDir).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "first").Run(); err != nil {
		t.Fatal(err)
	}

	out, _ := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD").Output()
	hash := string(out)
	hash = hash[:len(hash)-1]

	content, err := i.GetFileContentAtCommit(context.Background(), tmpDir, hash, "a.txt")
	if err != nil {
		t.Fatalf("GetFileContentAtCommit() error = %v", err)
	}
	if content != "v1" {
		t.Errorf("GetFileContentAtCommit() = %q; want v1", content)
	}
}

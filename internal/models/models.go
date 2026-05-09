package models

// CommitInfo holds information about a git commit
type CommitInfo struct {
	Hash      string
	Author    string
	Email     string
	Message   string
	Timestamp string
	Files     []FileChange
}

// FileChange represents a file change in a commit
type FileChange struct {
	Path     string
	Addition bool
	Content  string
}

package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCovignoreModuleDir(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, ".covignore")
	os.WriteFile(covPath, []byte("**/*.pb.go\n"), 0644)

	got := FindCovignore(dir)
	if got != covPath {
		t.Errorf("FindCovignore(%q) = %q, want %q", dir, got, covPath)
	}
}

func TestFindCovignoreFallback(t *testing.T) {
	dir := t.TempDir()

	got := FindCovignore(dir)
	if got != ".covignore" {
		t.Errorf("FindCovignore(%q) = %q, want %q", dir, got, ".covignore")
	}
}

func TestFindGoWorkDirFrom(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0755)
	os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n"), 0644)

	got := findGoWorkDirFrom(nested)
	if got != dir {
		t.Errorf("findGoWorkDirFrom(%q) = %q, want %q", nested, got, dir)
	}
}

func TestFindGoWorkDirFromNoWork(t *testing.T) {
	dir := t.TempDir()

	got := findGoWorkDirFrom(dir)
	if got != "" {
		t.Errorf("findGoWorkDirFrom(%q) = %q, want empty", dir, got)
	}
}

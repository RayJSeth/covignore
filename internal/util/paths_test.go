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

func TestModulePath(t *testing.T) {
	// We're inside a real Go module, so this should work.
	path, err := ModulePath()
	if err != nil {
		t.Fatal(err)
	}
	if path != "github.com/RayJSeth/covignore" {
		t.Errorf("ModulePath() = %q, want github.com/RayJSeth/covignore", path)
	}
}

func TestModuleRoot(t *testing.T) {
	info, err := ModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != "github.com/RayJSeth/covignore" {
		t.Errorf("Path = %q", info.Path)
	}
	if info.Dir == "" {
		t.Error("Dir should not be empty")
	}
	// Dir should contain a go.mod file
	if _, err := os.Stat(filepath.Join(info.Dir, "go.mod")); err != nil {
		t.Errorf("go.mod not found in Dir %q", info.Dir)
	}
}

func TestWorkspaceModulesNoWorkspace(t *testing.T) {
	// No go.work in this project, should return nil.
	modules, err := WorkspaceModules()
	if err != nil {
		t.Fatal(err)
	}
	if modules != nil {
		t.Errorf("expected nil modules without go.work, got %v", modules)
	}
}

func TestFindGoWorkDir(t *testing.T) {
	// From this project root, there's no go.work, so should return "".
	got := findGoWorkDir()
	if got != "" {
		t.Errorf("findGoWorkDir() = %q, want empty (no go.work in project)", got)
	}
}

func TestFindCovignoreCwdFallback(t *testing.T) {
	// When moduleDir has no .covignore but cwd does, it should find cwd's.
	dir := t.TempDir()
	t.Chdir(dir)

	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("*.pb.go\n"), 0644)
	otherDir := filepath.Join(dir, "submod")
	os.MkdirAll(otherDir, 0755)

	got := FindCovignore(otherDir)
	if got != ".covignore" {
		t.Errorf("FindCovignore(%q) = %q, want .covignore (cwd fallback)", otherDir, got)
	}
}

func TestWorkspaceModulesWithGoWork(t *testing.T) {
	dir := t.TempDir()
	modA := filepath.Join(dir, "modA")
	modB := filepath.Join(dir, "modB")
	os.MkdirAll(modA, 0755)
	os.MkdirAll(modB, 0755)
	os.WriteFile(filepath.Join(modA, "go.mod"), []byte("module example.com/modA\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modA, "a.go"), []byte("package modA\n"), 0644)
	os.WriteFile(filepath.Join(modB, "go.mod"), []byte("module example.com/modB\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modB, "b.go"), []byte("package modB\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n\nuse (\n\t./modA\n\t./modB\n)\n"), 0644)
	t.Chdir(dir)

	modules, err := WorkspaceModules()
	if err != nil {
		t.Fatalf("WorkspaceModules error: %v", err)
	}
	if len(modules) < 2 {
		t.Fatalf("expected at least 2 modules, got %d: %v", len(modules), modules)
	}
	paths := make(map[string]bool)
	for _, m := range modules {
		paths[m.Path] = true
	}
	if !paths["example.com/modA"] {
		t.Error("expected example.com/modA in modules")
	}
	if !paths["example.com/modB"] {
		t.Error("expected example.com/modB in modules")
	}
}

func TestFindGoWorkDirFromSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "deep", "nested")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n"), 0644)
	t.Chdir(sub)

	got := findGoWorkDir()
	if got == "" {
		t.Error("findGoWorkDir() should find go.work from subdirectory")
	}
}

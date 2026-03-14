package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatcherBasicGlobs(t *testing.T) {
	m := NewMatcher([]Pattern{
		{Glob: "**/*.pb.go"},
		{Glob: "**/mock/**"},
	})

	tests := []struct {
		path string
		want bool
	}{
		{"internal/api/service.pb.go", true},
		{"service.pb.go", true},
		{"internal/api/service.go", false},
		{"mock/client.go", true},
		{"internal/mock/client.go", true},
		{"internal/service.go", false},
	}

	for _, tt := range tests {
		got := m.Match(tt.path)
		if got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestMatcherNegation(t *testing.T) {
	m := NewMatcher([]Pattern{
		{Glob: "**/*_generated.go"},
		{Glob: "important_generated.go", Negate: true},
	})

	if !m.Match("internal/foo_generated.go") {
		t.Error("expected foo_generated.go to be ignored")
	}
	if m.Match("important_generated.go") {
		t.Error("expected important_generated.go to NOT be ignored (negation)")
	}
}

func TestMatcherEmpty(t *testing.T) {
	m := NewMatcher(nil)
	if !m.Empty() {
		t.Error("expected Empty() == true for nil patterns")
	}
	if m.Match("anything.go") {
		t.Error("expected no match on empty matcher")
	}
}

func TestLoadNonexistent(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), ".covignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Empty() {
		t.Error("expected empty matcher for nonexistent file")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covignore")

	content := `# comment
**/*.pb.go
!important.pb.go

**/mock/**
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if m.Empty() {
		t.Fatal("expected non-empty matcher")
	}

	if !m.Match("api/foo.pb.go") {
		t.Error("expected api/foo.pb.go to be ignored")
	}
	if m.Match("important.pb.go") {
		t.Error("expected important.pb.go to NOT be ignored")
	}
	if !m.Match("internal/mock/client.go") {
		t.Error("expected mock path to be ignored")
	}
}

func BenchmarkMatch(b *testing.B) {
	m := NewMatcher([]Pattern{
		{Glob: "**/*.pb.go"},
		{Glob: "**/mock/**"},
		{Glob: "**/*_generated.go"},
		{Glob: "vendor/**"},
		{Glob: "!important.pb.go", Negate: true},
	})

	paths := []string{
		"internal/api/service.pb.go",
		"internal/handler.go",
		"mock/client.go",
		"cmd/main.go",
		"internal/gen/types_generated.go",
		"pkg/util/helpers.go",
		"vendor/github.com/foo/bar.go",
		"important.pb.go",
		"deep/nested/path/to/file.go",
		"another/deep/path/mock/thing.go",
	}

	for i := range b.N {
		m.Match(paths[i%len(paths)])
	}
}

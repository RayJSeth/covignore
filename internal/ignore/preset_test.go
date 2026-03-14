package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPresetGenerated(t *testing.T) {
	patterns, ok := Presets["generated"]
	if !ok {
		t.Fatal("expected 'generated' preset to exist")
	}

	m := NewMatcher(patterns)

	tests := []struct {
		path string
		want bool
	}{
		{"api/service.pb.go", true},
		{"internal/types_gen.go", true},
		{"internal/schema_generated.go", true},
		{"mock/client.go", true},
		{"mocks/store.go", true},
		{"internal/handler_mock.go", true},
		{"ent/client.go", true},
		{"sqlc/queries.go", true},
		{"internal/handler.go", false},
		{"cmd/main.go", false},
	}

	for _, tt := range tests {
		got := m.Match(tt.path)
		if got != tt.want {
			t.Errorf("preset Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestPresetNames(t *testing.T) {
	names := PresetNames()
	if len(names) == 0 {
		t.Fatal("expected at least one preset")
	}
	found := false
	for _, n := range names {
		if n == "generated" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'generated' in preset names")
	}
}

func TestLoadWithPresetEmpty(t *testing.T) {
	m, err := LoadWithPreset(filepath.Join(t.TempDir(), ".covignore"), "")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Empty() {
		t.Error("expected empty matcher with no file and no preset")
	}
}

func TestLoadWithPresetOnly(t *testing.T) {
	m, err := LoadWithPreset(filepath.Join(t.TempDir(), ".covignore"), "generated")
	if err != nil {
		t.Fatal(err)
	}
	if m.Empty() {
		t.Error("expected non-empty matcher with preset")
	}
	if !m.Match("api/service.pb.go") {
		t.Error("expected preset to match .pb.go")
	}
}

func TestLoadWithPresetCombined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covignore")
	err := os.WriteFile(path, []byte("!important.pb.go\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	m, err := LoadWithPreset(path, "generated")
	if err != nil {
		t.Fatal(err)
	}

	if !m.Match("api/service.pb.go") {
		t.Error("expected service.pb.go to still be ignored")
	}
	if m.Match("important.pb.go") {
		t.Error("expected important.pb.go to NOT be ignored (file negation overrides preset)")
	}
}

func TestLoadWithPresetUnknown(t *testing.T) {
	_, err := LoadWithPreset(filepath.Join(t.TempDir(), ".covignore"), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/RayJSeth/covignore/internal/coverage"
	"github.com/RayJSeth/covignore/internal/ignore"
)

func TestSplitOnSeparatorWithDash(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCov  []string
		wantPass []string
	}{
		{
			name:     "separator splits cleanly",
			args:     []string{"--min=80", "--", "-race", "./..."},
			wantCov:  []string{"--min=80"},
			wantPass: []string{"-race", "./..."},
		},
		{
			name:     "nothing before separator",
			args:     []string{"--", "-v", "./..."},
			wantCov:  nil,
			wantPass: []string{"-v", "./..."},
		},
		{
			name:     "nothing after separator",
			args:     []string{"--min=80", "--"},
			wantCov:  []string{"--min=80"},
			wantPass: nil,
		},
		{
			name:     "only separator",
			args:     []string{"--"},
			wantCov:  nil,
			wantPass: nil,
		},
		{
			name:     "empty",
			args:     nil,
			wantCov:  nil,
			wantPass: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov, pass := splitOnSeparator(tt.args)
			if !sliceEqual(cov, tt.wantCov) {
				t.Errorf("covArgs = %v, want %v", cov, tt.wantCov)
			}
			if !sliceEqual(pass, tt.wantPass) {
				t.Errorf("passthrough = %v, want %v", pass, tt.wantPass)
			}
		})
	}
}

func TestSplitArgsByKnownFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCov  []string
		wantPass []string
	}{
		{
			name:     "covignore flags only",
			args:     []string{"--min=80", "--json", "--verbose"},
			wantCov:  []string{"--min=80", "--json", "--verbose"},
			wantPass: nil,
		},
		{
			name:     "go test flags only",
			args:     []string{"-v", "-race", "-run", "TestFoo", "./..."},
			wantCov:  nil,
			wantPass: []string{"-v", "-race", "-run", "TestFoo", "./..."},
		},
		{
			name:     "mixed flags",
			args:     []string{"--min=80", "-v", "-race", "--summary", "./..."},
			wantCov:  []string{"--min=80", "--summary"},
			wantPass: []string{"-v", "-race", "./..."},
		},
		{
			name:     "-o is covignore flag",
			args:     []string{"-o", "output.cov", "-race", "./..."},
			wantCov:  []string{"-o", "output.cov"},
			wantPass: []string{"-race", "./..."},
		},
		{
			name:     "-o= form",
			args:     []string{"-o=output.cov", "./..."},
			wantCov:  []string{"-o=output.cov"},
			wantPass: []string{"./..."},
		},
		{
			name:     "-v passes through to go test",
			args:     []string{"-v", "./..."},
			wantCov:  nil,
			wantPass: []string{"-v", "./..."},
		},
		{
			name:     "-race passes through",
			args:     []string{"-race", "./..."},
			wantCov:  nil,
			wantPass: []string{"-race", "./..."},
		},
		{
			name:     "-count passes through",
			args:     []string{"-count=1", "./..."},
			wantCov:  nil,
			wantPass: []string{"-count=1", "./..."},
		},
		{
			name:     "preset with space-separated value",
			args:     []string{"--preset", "generated", "./..."},
			wantCov:  []string{"--preset", "generated"},
			wantPass: []string{"./..."},
		},
		{
			name:     "preset with equals",
			args:     []string{"--preset=generated", "./..."},
			wantCov:  []string{"--preset=generated"},
			wantPass: []string{"./..."},
		},
		{
			name:     "html with path",
			args:     []string{"--html=report.html", "-v", "./..."},
			wantCov:  []string{"--html=report.html"},
			wantPass: []string{"-v", "./..."},
		},
		{
			name:     "html with space-separated value",
			args:     []string{"--html", "report.html", "./..."},
			wantCov:  []string{"--html", "report.html"},
			wantPass: []string{"./..."},
		},
		{
			name:     "all covignore flags at once",
			args:     []string{"--min=80", "--json", "--summary", "--verbose", "--dry-run", "--preset=generated", "-o", "out.cov"},
			wantCov:  []string{"--min=80", "--json", "--summary", "--verbose", "--dry-run", "--preset=generated", "-o", "out.cov"},
			wantPass: nil,
		},
		{
			name:     "empty input",
			args:     nil,
			wantCov:  nil,
			wantPass: nil,
		},
		{
			name:     "only packages",
			args:     []string{"./...", "./cmd/..."},
			wantCov:  nil,
			wantPass: []string{"./...", "./cmd/..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov, pass := splitArgsByKnownFlags(tt.args)
			if !sliceEqual(cov, tt.wantCov) {
				t.Errorf("covArgs = %v, want %v", cov, tt.wantCov)
			}
			if !sliceEqual(pass, tt.wantPass) {
				t.Errorf("passthrough = %v, want %v", pass, tt.wantPass)
			}
		})
	}
}

func TestLooksLikeCoverageFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "coverage.out")
	os.WriteFile(existing, []byte("mode: set\n"), 0644)

	tests := []struct {
		arg  string
		want bool
	}{
		{existing, true},
		{filepath.Join(dir, "nonexistent.out"), false},
		{"./...", false},
		{"-race", false},
		{"mypackage", false},
		{"-v", false},
	}

	for _, tt := range tests {
		got := looksLikeCoverageFile(tt.arg)
		if got != tt.want {
			t.Errorf("looksLikeCoverageFile(%q) = %v, want %v", tt.arg, got, tt.want)
		}
	}
}

func TestResolvePostProcessOutput(t *testing.T) {
	tests := []struct {
		source     string
		flagOutput string
		want       string
	}{
		{"coverage.out", "", "coverage.out"},
		{"-", "", "-"},
		{"coverage.out", "filtered.out", "filtered.out"},
		{"-", "filtered.out", "filtered.out"},
	}

	for _, tt := range tests {
		got := resolvePostProcessOutput(tt.source, tt.flagOutput)
		if got != tt.want {
			t.Errorf("resolvePostProcessOutput(%q, %q) = %q, want %q",
				tt.source, tt.flagOutput, got, tt.want)
		}
	}
}

func TestApplyFilterDedup(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "github.com/x/mock/a.go:1.1,5.2 2 1", File: "github.com/x/mock/a.go"},
			{Raw: "github.com/x/mock/a.go:6.1,10.2 3 0", File: "github.com/x/mock/a.go"},
			{Raw: "github.com/x/handler.go:1.1,5.2 2 1", File: "github.com/x/handler.go"},
		},
	}

	matcher := ignore.NewMatcher([]ignore.Pattern{{Glob: "**/mock/**"}})
	matchers := []coverage.ModuleMatcher{{ModulePrefix: "github.com/x", Matcher: matcher}}

	a := &app{stdout: io.Discard, stderr: io.Discard}
	result := a.applyFilter(profile, matchers, false)

	if len(result.FilteredFiles) != 1 {
		t.Fatalf("got %d filtered files, want 1: %v", len(result.FilteredFiles), result.FilteredFiles)
	}
	if result.FilteredFiles[0] != "github.com/x/mock/a.go" {
		t.Errorf("filtered file = %q, want github.com/x/mock/a.go", result.FilteredFiles[0])
	}
	if result.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", result.TotalEntries)
	}
	if len(result.Profile.Entries) != 1 {
		t.Errorf("kept entries = %d, want 1", len(result.Profile.Entries))
	}
}

func TestApplyFilterNoMatch(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "github.com/x/handler.go:1.1,5.2 2 1", File: "github.com/x/handler.go"},
		},
	}

	matcher := ignore.NewMatcher([]ignore.Pattern{{Glob: "**/mock/**"}})
	matchers := []coverage.ModuleMatcher{{ModulePrefix: "github.com/x", Matcher: matcher}}

	a := &app{stdout: io.Discard, stderr: io.Discard}
	result := a.applyFilter(profile, matchers, false)

	if len(result.FilteredFiles) != 0 {
		t.Errorf("expected 0 filtered files, got %v", result.FilteredFiles)
	}
	if len(result.Profile.Entries) != 1 {
		t.Errorf("all entries should be kept, got %d", len(result.Profile.Entries))
	}
}

func TestWriteOutputStdout(t *testing.T) {
	profile := &coverage.Profile{Mode: "set"}

	var buf bytes.Buffer
	a := &app{stdout: &buf, stderr: io.Discard}
	err := a.writeOutput("-", profile)
	if err != nil {
		t.Fatalf("writeOutput(\"-\") error: %v", err)
	}
	if !strings.Contains(buf.String(), "mode: set") {
		t.Error("stdout should contain mode line")
	}
}

func TestWriteOutputFile(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "file.go:1.1,5.2 2 1", File: "file.go"},
		},
	}

	a := &app{stdout: io.Discard, stderr: io.Discard}
	path := filepath.Join(t.TempDir(), "out.cov")
	err := a.writeOutput(path, profile)
	if err != nil {
		t.Fatalf("writeOutput error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "mode: set") {
		t.Error("output file missing mode line")
	}
	if !strings.Contains(string(data), "file.go:1.1,5.2 2 1") {
		t.Error("output file missing entry")
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

func TestRunWithVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := RunWith([]string{"--version"}, &stdout, io.Discard, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "version:") {
		t.Errorf("version output missing version line, got: %s", out)
	}
	if !strings.Contains(out, "go:") {
		t.Errorf("version output missing go line, got: %s", out)
	}
	if !strings.Contains(out, "platform:") {
		t.Errorf("version output missing platform line, got: %s", out)
	}
}

func TestRunWithHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--help"}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "Usage: covignore") {
		t.Errorf("help output missing branded usage, got: %s", stderr.String())
	}
}

func TestRunWithCheck(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.26\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "mock"), 0755)
	os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package proj\n"), 0644)
	os.WriteFile(filepath.Join(dir, "mock", "client.go"), []byte("package mock\n"), 0644)

	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--check"}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "mock/client.go") {
		t.Errorf("expected mock/client.go in check output, got: %s", output)
	}
	if !strings.Contains(output, "1 files ignored") {
		t.Errorf("expected '1 files ignored' in output, got: %s", output)
	}
}

func TestRunWithPostProcess(t *testing.T) {
	dir := t.TempDir()

	// Write a .covignore file.
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)

	// Write a coverage profile.
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/client.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)

	// Create a go.mod for module path detection.
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)

	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	// Verify the filtered profile doesn't contain mock entries.
	data, _ := os.ReadFile(covPath)
	if strings.Contains(string(data), "mock/client.go") {
		t.Error("mock entry should have been filtered out")
	}
	if !strings.Contains(string(data), "handler.go") {
		t.Error("handler entry should be kept")
	}
}

func TestRunWithJSON(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)

	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--json", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout.String())
	}
	if _, ok := result["coverage"]; !ok {
		t.Error("JSON output missing 'coverage' field")
	}
	if _, ok := result["filtered_files"]; !ok {
		t.Error("JSON output missing 'filtered_files' field")
	}
}

func TestRunWithPipe(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)

	t.Chdir(dir)

	input := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/m.go:1.1,5.2 2 1\n"
	stdin := strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"-o", "-", "-"}, &stdout, &stderr, stdin)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "mock/m.go") {
		t.Error("mock entry should be filtered in pipe output")
	}
	if !strings.Contains(output, "handler.go") {
		t.Error("handler entry should appear in pipe output")
	}
}

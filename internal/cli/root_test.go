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

func TestApplyFilterSummaryAlwaysPrinted(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "github.com/x/mock/a.go:1.1,5.2 2 1", File: "github.com/x/mock/a.go"},
			{Raw: "github.com/x/handler.go:1.1,5.2 2 1", File: "github.com/x/handler.go"},
		},
	}

	matcher := ignore.NewMatcher([]ignore.Pattern{{Glob: "**/mock/**"}})
	matchers := []coverage.ModuleMatcher{{ModulePrefix: "github.com/x", Matcher: matcher}}

	var stderr bytes.Buffer
	a := &app{stdout: io.Discard, stderr: &stderr}
	a.applyFilter(profile, matchers, false)

	output := stderr.String()
	if !strings.Contains(output, "covignore: filtered") {
		t.Errorf("expected summary line on stderr, got: %q", output)
	}
	if strings.Contains(output, "filtered: github.com") {
		t.Error("per-file details should not appear without --verbose")
	}
}

func TestApplyFilterVerboseShowsDetails(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "github.com/x/mock/a.go:1.1,5.2 2 1", File: "github.com/x/mock/a.go"},
			{Raw: "github.com/x/handler.go:1.1,5.2 2 1", File: "github.com/x/handler.go"},
		},
	}

	matcher := ignore.NewMatcher([]ignore.Pattern{{Glob: "**/mock/**"}})
	matchers := []coverage.ModuleMatcher{{ModulePrefix: "github.com/x", Matcher: matcher}}

	var stderr bytes.Buffer
	a := &app{stdout: io.Discard, stderr: &stderr}
	a.applyFilter(profile, matchers, true)

	output := stderr.String()
	if !strings.Contains(output, "filtered: github.com/x/mock/a.go") {
		t.Errorf("expected per-file detail with --verbose, got: %q", output)
	}
	if !strings.Contains(output, "covignore: filtered") {
		t.Errorf("expected summary line with --verbose, got: %q", output)
	}
}

func TestApplyFilterNoMatchSilentWithoutVerbose(t *testing.T) {
	profile := &coverage.Profile{
		Mode: "set",
		Entries: []coverage.Entry{
			{Raw: "github.com/x/handler.go:1.1,5.2 2 1", File: "github.com/x/handler.go"},
		},
	}

	matcher := ignore.NewMatcher([]ignore.Pattern{{Glob: "**/mock/**"}})
	matchers := []coverage.ModuleMatcher{{ModulePrefix: "github.com/x", Matcher: matcher}}

	var stderr bytes.Buffer
	a := &app{stdout: io.Discard, stderr: &stderr}
	a.applyFilter(profile, matchers, false)

	if stderr.Len() != 0 {
		t.Errorf("expected no output without --verbose when nothing filtered, got: %q", stderr.String())
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

func TestRunWithInit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--init"}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created") {
		t.Error("expected 'created' in output")
	}
	data, err := os.ReadFile(filepath.Join(dir, ".covignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "*.pb.go") {
		t.Error("default covignore should contain pb.go pattern")
	}
}

func TestRunWithInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("existing\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--init"}, &stdout, &stderr, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Error("expected 'already exists' error")
	}
}

func TestRunWithSummary(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 10 8\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--summary", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "coverage:") {
		t.Errorf("expected coverage summary, got: %s", stdout.String())
	}
}

func TestRunWithMinPass(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 10 10\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--min=80", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("expected pass, got exit code %d, stderr = %s", code, stderr.String())
	}
}

func TestRunWithMinFail(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	// 10 statements, 0 hits = 0% coverage
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 10 0\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--min=80", covPath}, &stdout, &stderr, nil)
	if code != 1 {
		t.Fatalf("expected fail (exit 1), got %d", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Error("expected FAIL in stderr")
	}
}

func TestRunWithJSONAndMinFail(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	// 10 statements, 0 hits = 0% coverage
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 10 0\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--json", "--min=80", covPath}, &stdout, &stderr, nil)
	if code != 1 {
		t.Fatalf("expected fail (exit 1) with --json --min, got %d", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Error("expected FAIL in stderr")
	}
	// JSON output should still be produced
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON output even on threshold failure: %v\n%s", err, stdout.String())
	}
}

func TestRunWithJSONAndMinPass(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 10 10\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--json", "--min=80", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("expected pass (exit 0), got %d, stderr = %s", code, stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout.String())
	}
}

func TestRunWithDryRun(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/m.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--dry-run", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	// File should NOT be modified in dry-run mode.
	data, _ := os.ReadFile(covPath)
	if !strings.Contains(string(data), "mock/m.go") {
		t.Error("dry-run should not modify the file")
	}
}

func TestRunWithVerbose(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/m.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--verbose", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	if !strings.Contains(stderr.String(), "filtered:") {
		t.Errorf("expected per-file detail with --verbose, got: %s", stderr.String())
	}
}

func TestRunWithBadCoverageFile(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "bad.out")
	os.WriteFile(covPath, []byte("not a coverage file\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{covPath}, &stdout, &stderr, nil)
	if code != 1 {
		t.Fatalf("expected exit 1 for bad coverage file, got %d", code)
	}
}

func TestRunWithNoCovignoreWarning(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	// Intentionally no .covignore
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	RunWith([]string{covPath}, &stdout, &stderr, nil)
	if !strings.Contains(stderr.String(), "no .covignore found") {
		t.Errorf("expected warning about missing .covignore, got: %s", stderr.String())
	}
}

func TestRunWrapper(t *testing.T) {
	// Create a tiny Go project to run go test against.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--summary", "--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s, stdout = %s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "coverage:") {
		t.Errorf("expected coverage summary, got stdout: %s", stdout.String())
	}
}

func TestRunWrapperWithDryRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--dry-run", "--summary", "--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestRunWrapperWithJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--json", "--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	// stdout contains go test output followed by JSON; find the JSON object.
	output := stdout.String()
	idx := strings.Index(output, "{")
	if idx < 0 {
		t.Fatalf("no JSON object found in output: %s", output)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output[idx:]), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output[idx:])
	}
}

func TestFinalizeHTMLRequiresFile(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	// Post-process from stdin with HTML (requires file output)
	input := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 2 1\n"
	code := RunWith([]string{"--html=report.html", "-o", "-", "-"}, &stdout, &stderr, strings.NewReader(input))
	if code != 1 {
		t.Fatalf("expected exit 1 (html requires file), got %d", code)
	}
	if !strings.Contains(stderr.String(), "--html requires file output") {
		t.Errorf("expected html error, got: %s", stderr.String())
	}
}

func TestRunWithPostProcessOutputFlag(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "input.out")
	outPath := filepath.Join(dir, "filtered.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/m.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"-o", outPath, covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, _ := os.ReadFile(outPath)
	if strings.Contains(string(data), "mock/m.go") {
		t.Error("mock entry should be filtered")
	}
	// Original should be untouched
	orig, _ := os.ReadFile(covPath)
	if !strings.Contains(string(orig), "mock/m.go") {
		t.Error("original file should not be modified when -o is used")
	}
}

func TestRunWithPreset(t *testing.T) {
	dir := t.TempDir()
	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\ngithub.com/x/proj/handler.go:1.1,5.2 2 1\ngithub.com/x/proj/gen_foo.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	// No .covignore — rely on preset
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--preset=generated", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestRunWithCheckPreset(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.26\n"), 0644)
	os.Mkdir(filepath.Join(dir, "internal"), 0755)
	os.WriteFile(filepath.Join(dir, "handler.go"), []byte("package proj\n"), 0644)
	os.WriteFile(filepath.Join(dir, "gen_foo.go"), []byte("package proj\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--check", "--preset=generated"}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
}

func TestRunWithBadFlag(t *testing.T) {
	// Use -- separator so the bad flag reaches flag.Parse
	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--nonexistent-flag", "--"}, &stdout, &stderr, nil)
	if code != 2 {
		t.Fatalf("expected exit 2 for bad flag, got %d", code)
	}
}

func TestParseFlagName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"--min=80", "min"},
		{"--json", "json"},
		{"-o", "o"},
		{"-o=out.cov", "o"},
		{"-v", ""},
		{"./...", ""},
		{"--", ""},
	}
	for _, tt := range tests {
		got := parseFlagName(tt.input)
		if got != tt.want {
			t.Errorf("parseFlagName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWantsHelp(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"-h"}, true},
		{[]string{"-help"}, true},
		{[]string{"--help"}, true},
		{[]string{"--", "-h"}, false},
		{[]string{"--json"}, false},
		{nil, false},
	}
	for _, tt := range tests {
		got := wantsHelp(tt.args)
		if got != tt.want {
			t.Errorf("wantsHelp(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestRunWrapperFailingTest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fail\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package fail\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package fail\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) { t.Fatal(\"oops\") }\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--", "./..."}, &stdout, &stderr, nil)
	if code == 0 {
		t.Fatal("expected non-zero exit for failing test")
	}
}

func TestRunWrapperWithMin(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--min=50", "--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("expected pass with min=50, got exit %d, stderr: %s", code, stderr.String())
	}
}

func TestRunWrapperWithMinFail(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nimport \"fmt\"\n\nfunc Hello() string { return \"hi\" }\nfunc Unused() string { fmt.Sprintf(\"x\"); return \"unused\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	// 100% min should fail since Unused() isn't tested
	code := RunWith([]string{"--min=100", "--", "./..."}, &stdout, &stderr, nil)
	if code != 1 {
		t.Fatalf("expected fail with min=100, got exit %d", code)
	}
	if !strings.Contains(stderr.String(), "FAIL") {
		t.Errorf("expected FAIL in stderr, got: %s", stderr.String())
	}
}

func TestRunWrapperNoTestFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/empty\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package empty\n"), 0644)
	// No test files
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("expected exit 0 for no test files, got %d, stderr: %s", code, stderr.String())
	}
}

func TestRunWrapperWithFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("lib.go\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--verbose", "--", "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "filtered") {
		t.Errorf("expected filter output, got stderr: %s", stderr.String())
	}
}

func TestRunWrapperWithUserCoverprofile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/wrap\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package wrap\n\nfunc Hello() string { return \"hi\" }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package wrap\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"hi\" { t.Fatal() }\n}\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte(""), 0644)
	t.Chdir(dir)

	customProfile := filepath.Join(dir, "custom.out")
	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--summary", "--", "-coverprofile=" + customProfile, "./..."}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(customProfile); err != nil {
		t.Errorf("custom profile should exist: %v", err)
	}
}

func TestRunWithPostProcessStdinDryRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/x/proj\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".covignore"), []byte("**/mock/**\n"), 0644)
	t.Chdir(dir)

	input := "mode: set\ngithub.com/x/proj/a.go:1.1,5.2 2 1\ngithub.com/x/proj/mock/m.go:1.1,5.2 2 1\n"
	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--dry-run", "--summary", "-"}, &stdout, &stderr, strings.NewReader(input))
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "coverage:") {
		t.Errorf("expected summary, got: %s", stdout.String())
	}
}

func TestCheckMultiModule(t *testing.T) {
	// Create a workspace with two modules.
	dir := t.TempDir()
	modA := filepath.Join(dir, "modA")
	modB := filepath.Join(dir, "modB")
	os.MkdirAll(modA, 0755)
	os.MkdirAll(modB, 0755)
	os.WriteFile(filepath.Join(modA, "go.mod"), []byte("module example.com/modA\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modB, "go.mod"), []byte("module example.com/modB\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modA, "a.go"), []byte("package modA\n"), 0644)
	os.WriteFile(filepath.Join(modB, "b.go"), []byte("package modB\n"), 0644)
	os.WriteFile(filepath.Join(modA, ".covignore"), []byte(""), 0644)
	os.WriteFile(filepath.Join(modB, ".covignore"), []byte("b.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n\nuse (\n\t./modA\n\t./modB\n)\n"), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--check"}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "module example.com/modA") {
		t.Errorf("expected modA in output, got: %s", output)
	}
	if !strings.Contains(output, "module example.com/modB") {
		t.Errorf("expected modB in output, got: %s", output)
	}
	if !strings.Contains(output, "total:") {
		t.Errorf("expected total line, got: %s", output)
	}
}

func TestPostProcessMultiModule(t *testing.T) {
	dir := t.TempDir()
	modA := filepath.Join(dir, "modA")
	modB := filepath.Join(dir, "modB")
	os.MkdirAll(modA, 0755)
	os.MkdirAll(modB, 0755)
	os.WriteFile(filepath.Join(modA, "go.mod"), []byte("module example.com/modA\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modB, "go.mod"), []byte("module example.com/modB\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(modA, ".covignore"), []byte(""), 0644)
	os.WriteFile(filepath.Join(modB, ".covignore"), []byte("gen_*.go\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go.work"), []byte("go 1.22\n\nuse (\n\t./modA\n\t./modB\n)\n"), 0644)

	covPath := filepath.Join(dir, "coverage.out")
	covData := "mode: set\nexample.com/modA/a.go:1.1,5.2 2 1\nexample.com/modB/gen_foo.go:1.1,5.2 2 1\nexample.com/modB/real.go:1.1,5.2 2 1\n"
	os.WriteFile(covPath, []byte(covData), 0644)
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := RunWith([]string{"--summary", covPath}, &stdout, &stderr, nil)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, _ := os.ReadFile(covPath)
	if strings.Contains(string(data), "gen_foo.go") {
		t.Error("gen_foo.go should have been filtered")
	}
}

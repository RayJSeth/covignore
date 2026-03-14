package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGoTestBasic(t *testing.T) {
	// Create a tiny Go module with a passing test.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package tiny\n\nfunc Add(a, b int) int { return a + b }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package tiny\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1,2) != 3 { t.Fatal() }\n}\n"), 0644)

	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	raw := filepath.Join(dir, "cover.raw")
	result, err := RunGoTest([]string{"./..."}, raw, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("RunGoTest error: %v\nstderr: %s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0\nstderr: %s", result.ExitCode, stderr.String())
	}
	if result.RawProfile != raw {
		t.Errorf("RawProfile = %q, want %q", result.RawProfile, raw)
	}
	// Coverage file should exist and contain mode line.
	data, err := os.ReadFile(raw)
	if err != nil {
		t.Fatalf("coverage file not created: %v", err)
	}
	if !strings.Contains(string(data), "mode:") {
		t.Error("coverage file missing mode line")
	}
}

func TestRunGoTestWithExistingCoverprofile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package tiny\n\nfunc Add(a, b int) int { return a + b }\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package tiny\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1,2) != 3 { t.Fatal() }\n}\n"), 0644)

	t.Chdir(dir)

	customProfile := filepath.Join(dir, "custom.out")
	var stdout, stderr bytes.Buffer
	result, err := RunGoTest([]string{"-coverprofile=" + customProfile, "./..."}, "unused.raw", &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("RunGoTest error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	// Should use the user-provided profile path, not the default.
	if result.RawProfile != customProfile {
		t.Errorf("RawProfile = %q, want %q", result.RawProfile, customProfile)
	}
	if _, err := os.Stat(customProfile); err != nil {
		t.Errorf("custom profile not created: %v", err)
	}
}

func TestRunGoTestFailingTest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.22\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib.go"), []byte("package tiny\n"), 0644)
	os.WriteFile(filepath.Join(dir, "lib_test.go"), []byte("package tiny\n\nimport \"testing\"\n\nfunc TestFail(t *testing.T) { t.Fatal(\"oops\") }\n"), 0644)

	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	result, err := RunGoTest([]string{"./..."}, filepath.Join(dir, "cover.raw"), &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("RunGoTest should not return error for test failure: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for failing test")
	}
}

func TestExtractCoverProfile(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"equals single dash", []string{"-coverprofile=cov.out"}, "cov.out"},
		{"space single dash", []string{"-coverprofile", "cov.out"}, "cov.out"},
		{"equals double dash", []string{"--coverprofile=cov.out"}, "cov.out"},
		{"space double dash", []string{"--coverprofile", "cov.out"}, "cov.out"},
		{"not present", []string{"-v", "./..."}, ""},
		{"trailing flag no value", []string{"-coverprofile"}, ""},
		{"mixed args", []string{"-v", "-coverprofile=out.cov", "./..."}, "out.cov"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCoverProfile(tt.args)
			if got != tt.want {
				t.Errorf("extractCoverProfile(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

package coverage

import (
	"bytes"
	"strings"
	"testing"

	"github.com/RayJSeth/covignore/internal/ignore"
)

const testCoverageData = `mode: set
github.com/user/project/internal/api/handler.go:10.1,15.2 3 1
github.com/user/project/internal/api/service.pb.go:1.1,5.2 2 1
github.com/user/project/mock/client.go:1.1,10.2 5 0
github.com/user/project/internal/util/helper.go:1.1,20.2 4 1
`

func TestParse(t *testing.T) {
	p, err := Parse(strings.NewReader(testCoverageData))
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != "set" {
		t.Errorf("Mode = %q, want %q", p.Mode, "set")
	}
	if len(p.Entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(p.Entries))
	}
	if p.Entries[0].File != "github.com/user/project/internal/api/handler.go" {
		t.Errorf("wrong file: %s", p.Entries[0].File)
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseInvalidMode(t *testing.T) {
	_, err := Parse(strings.NewReader("not a mode line\n"))
	if err == nil {
		t.Fatal("expected error for invalid mode line")
	}
}

func TestFilter(t *testing.T) {
	p, _ := Parse(strings.NewReader(testCoverageData))

	matcher := ignore.NewMatcher([]ignore.Pattern{
		{Glob: "**/*.pb.go"},
		{Glob: "**/mock/**"},
	})

	filtered := Filter(p, matcher, "github.com/user/project")

	if len(filtered.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(filtered.Entries))
	}

	for _, e := range filtered.Entries {
		rel := strings.TrimPrefix(e.File, "github.com/user/project/")
		if strings.HasSuffix(rel, ".pb.go") || strings.HasPrefix(rel, "mock/") {
			t.Errorf("entry %q should have been filtered", rel)
		}
	}
}

func TestFilterEmptyMatcher(t *testing.T) {
	p, _ := Parse(strings.NewReader(testCoverageData))
	matcher := ignore.NewMatcher(nil)

	filtered := Filter(p, matcher, "github.com/user/project")
	if len(filtered.Entries) != len(p.Entries) {
		t.Error("empty matcher should not filter anything")
	}
}

func TestFilterMultiModule(t *testing.T) {
	input := `mode: set
github.com/org/svcA/handler.go:1.1,10.2 3 1
github.com/org/svcA/handler.pb.go:1.1,5.2 2 1
github.com/org/svcB/server.go:1.1,8.2 4 1
github.com/org/svcB/mock/client.go:1.1,6.2 3 0
github.com/org/svcB/util.go:1.1,4.2 2 1
`
	p, _ := Parse(strings.NewReader(input))

	matcherA := ignore.NewMatcher([]ignore.Pattern{
		{Glob: "**/*.pb.go"},
	})
	matcherB := ignore.NewMatcher([]ignore.Pattern{
		{Glob: "**/mock/**"},
	})

	filtered := FilterMultiModule(p, []ModuleMatcher{
		{ModulePrefix: "github.com/org/svcA", Matcher: matcherA},
		{ModulePrefix: "github.com/org/svcB", Matcher: matcherB},
	})

	// svcA/handler.pb.go filtered by matcherA, svcB/mock/client.go filtered by matcherB
	if len(filtered.Entries) != 3 {
		t.Fatalf("got %d entries, want 3; entries: %v", len(filtered.Entries), entryFiles(filtered))
	}

	kept := entryFiles(filtered)
	for _, f := range kept {
		if strings.HasSuffix(f, ".pb.go") {
			t.Errorf("pb.go file should have been filtered: %s", f)
		}
		if strings.Contains(f, "mock/") {
			t.Errorf("mock file should have been filtered: %s", f)
		}
	}
}

func TestFilterMultiModuleUnmatchedEntry(t *testing.T) {
	input := `mode: set
github.com/other/pkg/file.go:1.1,5.2 2 1
`
	p, _ := Parse(strings.NewReader(input))

	matcher := ignore.NewMatcher([]ignore.Pattern{
		{Glob: "**/*.go"},
	})

	filtered := FilterMultiModule(p, []ModuleMatcher{
		{ModulePrefix: "github.com/org/svcA", Matcher: matcher},
	})

	if len(filtered.Entries) != 1 {
		t.Fatalf("unmatched module entries should be kept, got %d", len(filtered.Entries))
	}
}

func entryFiles(p *Profile) []string {
	var files []string
	for _, e := range p.Entries {
		files = append(files, e.File)
	}
	return files
}

func TestWriteRoundTrip(t *testing.T) {
	p, _ := Parse(strings.NewReader(testCoverageData))

	var buf bytes.Buffer
	if err := Write(&buf, p); err != nil {
		t.Fatal(err)
	}

	p2, err := Parse(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Mode != p.Mode {
		t.Errorf("mode mismatch: %q vs %q", p2.Mode, p.Mode)
	}
	if len(p2.Entries) != len(p.Entries) {
		t.Errorf("entry count mismatch: %d vs %d", len(p2.Entries), len(p.Entries))
	}
}

func TestComputeStats(t *testing.T) {
	p, _ := Parse(strings.NewReader(testCoverageData))
	stats := ComputeStats(p)

	// 3+2+5+4 = 14 statements; covered (count>0): 3+2+4 = 9
	if stats.Statements != 14 {
		t.Errorf("Statements = %d, want 14", stats.Statements)
	}
	if stats.Covered != 9 {
		t.Errorf("Covered = %d, want 9", stats.Covered)
	}

	pct := stats.Percent()
	expected := float64(9) / float64(14) * 100
	if pct != expected {
		t.Errorf("Percent = %f, want %f", pct, expected)
	}
}

func TestComputeStatsFiles(t *testing.T) {
	p, _ := Parse(strings.NewReader(testCoverageData))
	stats := ComputeStats(p)

	if len(stats.Files) != 4 {
		t.Fatalf("got %d files, want 4", len(stats.Files))
	}
}

func TestStatsZeroStatements(t *testing.T) {
	stats := Stats{}
	if stats.Percent() != 0 {
		t.Errorf("expected 0%% for zero statements, got %f", stats.Percent())
	}
}

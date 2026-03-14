package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/RayJSeth/covignore/internal/coverage"
	"github.com/RayJSeth/covignore/internal/ignore"
	"github.com/RayJSeth/covignore/internal/runner"
	"github.com/RayJSeth/covignore/internal/threshold"
	"github.com/RayJSeth/covignore/internal/util"
)

const (
	covignoreFile  = ".covignore"
	rawProfileFile = ".covignore.raw"
	defaultOutput  = "coverage.out"
	flagDryRun     = "dry-run"
)

const defaultCovignore = `# Generated files
**/*.pb.go
**/*_generated.go
**/*_gen.go

# Mocks
**/mock/**
**/mocks/**
**/*_mock.go

# Code generation tools
**/ent/**
**/sqlc/**
`

type app struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (a *app) errorf(format string, args ...any) {
	fmt.Fprintf(a.stderr, "covignore: "+format+"\n", args...)
}

const logo = `
                _                            
  ___ _____   _(_) __ _ _ __   ___  _ __ ___ 
 / __/ _ \ \ / / |/ _` + "`" + ` | '_ \ / _ \| '__/ _ \
| (_| (_) \ V /| | (_| | | | | (_) | | |  __/
 \___\___/ \_/ |_|\__, |_| |_|\___/|_|  \___|
                  |___/                       
`

func (a *app) printVersion() {
	fmt.Fprint(a.stdout, logo)
	fmt.Fprintf(a.stdout, "  version:  %s\n", Version)

	goVersion := runtime.Version()
	if info, ok := debug.ReadBuildInfo(); ok && info.GoVersion != "" {
		goVersion = info.GoVersion
	}
	fmt.Fprintf(a.stdout, "  go:       %s\n", goVersion)
	fmt.Fprintf(a.stdout, "  platform: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
}

func Run() int {
	return RunWith(os.Args[1:], os.Stdout, os.Stderr, os.Stdin)
}

// Really intended just for integration testing.
func RunWith(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	a := &app{stdout: stdout, stderr: stderr, stdin: stdin}
	return a.run(args)
}

func (a *app) run(args []string) int {
	flags := &Flags{}

	fs := flag.NewFlagSet("covignore", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	fs.Usage = func() {
		fmt.Fprintf(a.stderr, `Usage: covignore [flags] [-- go-test-args...]
       covignore [flags] <coverage-file>
       ... | covignore [flags] -

Flags:
`)
		fs.PrintDefaults()
	}
	fs.Float64Var(&flags.Min, "min", 0, "minimum coverage threshold percentage")
	fs.StringVar(&flags.HTML, "html", "", "write HTML coverage report to PATH")
	fs.BoolVar(&flags.JSON, "json", false, "output coverage report as JSON")
	fs.BoolVar(&flags.Summary, "summary", false, "print coverage summary line")
	fs.BoolVar(&flags.Init, "init", false, "create a default .covignore file")
	fs.StringVar(&flags.Preset, "preset", "", "built-in ignore preset (e.g. generated)")
	fs.StringVar(&flags.Output, "o", "", "output file path (default: coverage.out, or - for stdout)")
	fs.BoolVar(&flags.DryRun, flagDryRun, false, "show what would be filtered without writing")
	fs.BoolVar(&flags.Verbose, "verbose", false, "show which entries are filtered")
	fs.BoolVar(&flags.Check, "check", false, "list files matched by current patterns")
	fs.BoolVar(&flags.ShowVer, "version", false, "print version")

	// fix issue where help was getting passed to go test.
	if wantsHelp(args) {
		fs.Usage()
		return 0
	}

	covArgs, passthrough := splitOnSeparator(args)
	if err := fs.Parse(covArgs); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if flags.ShowVer {
		a.printVersion()
		return 0
	}

	if flags.Init {
		return a.runInit()
	}

	if flags.Check {
		return a.runCheck(flags)
	}

	remaining := append(fs.Args(), passthrough...)

	// post process mode
	if len(remaining) == 1 && (remaining[0] == "-" || looksLikeCoverageFile(remaining[0])) {
		return a.runPostProcess(remaining[0], flags)
	}

	// wrapper mode
	if flags.Output == "" {
		flags.Output = defaultOutput
	}
	return a.runWrapper(remaining, flags)
}

func splitOnSeparator(args []string) ([]string, []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return splitArgsByKnownFlags(args)
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "-help" || arg == "--help" {
			return true
		}
		if arg == "--" {
			return false
		}
	}
	return false
}

func splitArgsByKnownFlags(args []string) ([]string, []string) {
	var covArgs, passthrough []string
	known := map[string]bool{
		"min": true, "html": true, "json": true, "summary": true,
		"init": true, "preset": true, "o": true, flagDryRun: true,
		"verbose": true, "check": true, "version": true,
	}
	boolFlags := map[string]bool{
		"json": true, "summary": true, "init": true, flagDryRun: true,
		"verbose": true, "check": true, "version": true,
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		name := parseFlagName(a)

		// fix: CutPrefix("--", "--") yields "", which is not a real flag
		if name != "" && known[name] {
			covArgs = append(covArgs, a)
			if !strings.Contains(a, "=") && !boolFlags[name] {
				if i+1 < len(args) {
					i++
					covArgs = append(covArgs, args[i])
				}
			}
		} else {
			passthrough = append(passthrough, a)
		}
	}
	return covArgs, passthrough
}

func parseFlagName(a string) string {
	if after, ok := strings.CutPrefix(a, "--"); ok {
		name, _, _ := strings.Cut(after, "=")
		return name
	}
	if a == "-o" || strings.HasPrefix(a, "-o=") {
		return "o"
	}
	return ""
}

func looksLikeCoverageFile(arg string) bool {
	if !strings.HasSuffix(arg, ".out") && !strings.HasSuffix(arg, ".raw") && !strings.HasSuffix(arg, ".cov") {
		return false
	}
	// must also exist on disk to avoid confusing package names like mypackage.out.
	_, err := os.Stat(arg)
	return err == nil
}

func (a *app) runInit() int {
	targetDir := "."
	if info, err := util.ModuleRoot(); err == nil {
		targetDir = info.Dir
	}
	path := filepath.Join(targetDir, covignoreFile)
	if _, err := os.Stat(path); err == nil {
		a.errorf("%s already exists", path)
		return 1
	}
	if err := os.WriteFile(path, []byte(defaultCovignore), 0644); err != nil {
		a.errorf("%v", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "created %s\n", path)
	return 0
}

func (a *app) buildModuleMatchers(preset string) ([]coverage.ModuleMatcher, error) {
	modules, err := util.WorkspaceModules()
	if err != nil {
		a.errorf("warning: %v; falling back to single-module mode", err)
		modules = nil
	}

	if len(modules) > 1 {
		var matchers []coverage.ModuleMatcher
		for _, mod := range modules {
			covPath := util.FindCovignore(mod.Dir)
			m, err := ignore.LoadWithPreset(covPath, preset)
			if err != nil {
				return nil, fmt.Errorf("loading %s for module %s: %w", covPath, mod.Path, err)
			}
			matchers = append(matchers, coverage.ModuleMatcher{
				ModulePrefix: mod.Path,
				Matcher:      m,
			})
		}
		return matchers, nil
	}

	modPath, _ := util.ModulePath()
	m, err := ignore.LoadWithPreset(covignoreFile, preset)
	if err != nil {
		return nil, err
	}

	if m.Empty() && preset == "" {
		if _, statErr := os.Stat(covignoreFile); os.IsNotExist(statErr) {
			a.errorf("no .covignore found; use --init to create one")
		}
	}

	return []coverage.ModuleMatcher{{ModulePrefix: modPath, Matcher: m}}, nil
}

func (a *app) runCheck(flags *Flags) int {
	modules, err := util.WorkspaceModules()
	if err != nil {
		a.errorf("warning: %v; falling back to single-module mode", err)
		modules = nil
	}

	if len(modules) > 1 {
		var totalIgnored, totalIncluded int
		for _, mod := range modules {
			covPath := util.FindCovignore(mod.Dir)
			matcher, err := ignore.LoadWithPreset(covPath, flags.Preset)
			if err != nil {
				a.errorf("%v", err)
				return 1
			}
			fmt.Fprintf(a.stdout, "module %s:\n", mod.Path)
			ignored, included := a.checkDir(mod.Dir, matcher)
			totalIgnored += ignored
			totalIncluded += included
		}
		fmt.Fprintf(a.stdout, "\ntotal: %d files ignored, %d files included\n", totalIgnored, totalIncluded)
		return 0
	}

	modInfo, err := util.ModuleRoot()
	if err != nil {
		a.errorf("%v", err)
		return 1
	}

	matcher, err := ignore.LoadWithPreset(covignoreFile, flags.Preset)
	if err != nil {
		a.errorf("%v", err)
		return 1
	}

	ignored, included := a.checkDir(modInfo.Dir, matcher)
	fmt.Fprintf(a.stdout, "\n%d files ignored, %d files included\n", ignored, included)
	return 0
}

func (a *app) checkDir(dir string, matcher *ignore.Matcher) (int, int) {
	var ignored, included int
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			a.errorf("walk %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "vendor" || base == "node_modules" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if matcher.Match(rel) {
			fmt.Fprintf(a.stdout, "  ignore: %s\n", rel)
			ignored++
		} else {
			included++
		}
		return nil
	})
	return ignored, included
}

func (a *app) runPostProcess(source string, flags *Flags) int {
	matchers, err := a.buildModuleMatchers(flags.Preset)
	if err != nil {
		a.errorf("%v", err)
		return 1
	}

	var profile *coverage.Profile
	if source == "-" {
		profile, err = coverage.Parse(a.stdin)
	} else {
		profile, err = parseCoverageFile(source)
	}
	if err != nil {
		a.errorf("%v", err)
		return 1
	}

	result := a.applyFilter(profile, matchers, flags.Verbose)

	outputPath := resolvePostProcessOutput(source, flags.Output)
	if !flags.DryRun {
		if err := a.writeOutput(outputPath, result.Profile); err != nil {
			a.errorf("%v", err)
			return 1
		}
	}

	return a.finalize(result, flags, outputPath)
}

func resolvePostProcessOutput(source, flagOutput string) string {
	if flagOutput != "" {
		return flagOutput
	}
	if source == "-" {
		return "-"
	}
	return source
}

func (a *app) runWrapper(args []string, flags *Flags) int {
	if len(args) == 0 {
		args = []string{"./..."}
	}

	result, err := runner.RunGoTest(args, rawProfileFile, a.stdout, a.stderr, a.stdin)
	if err != nil {
		os.Remove(rawProfileFile)
		a.errorf("%v", err)
		return 1
	}

	ownProfile := result.RawProfile == rawProfileFile
	if ownProfile {
		defer os.Remove(rawProfileFile)
	}

	if result.ExitCode != 0 {
		return result.ExitCode
	}

	matchers, err := a.buildModuleMatchers(flags.Preset)
	if err != nil {
		a.errorf("%v", err)
		return 1
	}

	profile, err := parseCoverageFile(result.RawProfile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0 // no coverage generated (e.g. no test files)
		}
		a.errorf("%v", err)
		return 1
	}

	filtered := a.applyFilter(profile, matchers, flags.Verbose)

	outputPath := flags.Output
	if !ownProfile {
		outputPath = result.RawProfile
	}

	if !flags.DryRun {
		if err := a.writeOutput(outputPath, filtered.Profile); err != nil {
			a.errorf("%v", err)
			return 1
		}
	}

	return a.finalize(filtered, flags, outputPath)
}

type filterOutcome struct {
	Profile       *coverage.Profile
	TotalEntries  int
	FilteredFiles []string
	FilteredCount int
}

func (a *app) applyFilter(profile *coverage.Profile, matchers []coverage.ModuleMatcher, verbose bool) *filterOutcome {
	filtered := coverage.FilterMultiModule(profile, matchers)

	kept := make(map[string]struct{}, len(filtered.Entries))
	for _, e := range filtered.Entries {
		kept[e.File] = struct{}{}
	}
	var filteredFiles []string
	seen := make(map[string]struct{})
	for _, e := range profile.Entries {
		if _, ok := kept[e.File]; ok {
			continue
		}
		if _, dup := seen[e.File]; dup {
			continue
		}
		seen[e.File] = struct{}{}
		filteredFiles = append(filteredFiles, e.File)
	}
	sort.Strings(filteredFiles)

	filteredCount := len(profile.Entries) - len(filtered.Entries)

	if verbose {
		if len(filteredFiles) > 0 {
			for _, f := range filteredFiles {
				fmt.Fprintf(a.stderr, "  filtered: %s\n", f)
			}
			fmt.Fprintf(a.stderr, "covignore: filtered %d entries (%d files) of %d total\n",
				filteredCount, len(filteredFiles), len(profile.Entries))
		} else {
			fmt.Fprintf(a.stderr, "covignore: no entries filtered\n")
		}
	}

	return &filterOutcome{
		Profile:       filtered,
		TotalEntries:  len(profile.Entries),
		FilteredFiles: filteredFiles,
		FilteredCount: filteredCount,
	}
}

func (a *app) writeOutput(path string, profile *coverage.Profile) error {
	if path == "-" {
		return coverage.Write(a.stdout, profile)
	}
	return coverage.WriteFile(path, profile)
}

func parseCoverageFile(path string) (*coverage.Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return coverage.Parse(f)
}

func (a *app) finalize(result *filterOutcome, flags *Flags, outputPath string) int {
	stats := coverage.ComputeStats(result.Profile)

	if flags.JSON {
		return a.writeJSON(stats, result)
	}

	if flags.Summary || flags.Min > 0 {
		fmt.Fprintf(a.stdout, "coverage: %s\n", stats)
	}

	if flags.HTML != "" && !flags.DryRun {
		if outputPath == "-" {
			a.errorf("--html requires file output (use -o PATH)")
			return 1
		}
		cmd := exec.Command("go", "tool", "cover", "-html="+outputPath, "-o", flags.HTML)
		cmd.Stderr = a.stderr
		if err := cmd.Run(); err != nil {
			a.errorf("generating HTML: %v", err)
			return 1
		}
	}

	if flags.Min > 0 {
		if err := threshold.Check(stats.Percent(), flags.Min); err != nil {
			fmt.Fprintf(a.stderr, "FAIL\n%v\n", err)
			return 1
		}
	}

	return 0
}

func (a *app) writeJSON(stats coverage.Stats, result *filterOutcome) int {
	files := stats.Files
	if files == nil {
		files = []string{}
	}
	sort.Strings(files)
	filteredFiles := result.FilteredFiles
	if filteredFiles == nil {
		filteredFiles = []string{}
	}
	out := struct {
		Coverage        float64  `json:"coverage"`
		Statements      int      `json:"statements"`
		Covered         int      `json:"covered"`
		Files           []string `json:"files"`
		FilteredFiles   []string `json:"filtered_files"`
		TotalEntries    int      `json:"total_entries"`
		FilteredEntries int      `json:"filtered_entries"`
	}{
		Coverage:        stats.Percent(),
		Statements:      stats.Statements,
		Covered:         stats.Covered,
		Files:           files,
		FilteredFiles:   filteredFiles,
		TotalEntries:    result.TotalEntries,
		FilteredEntries: result.FilteredCount,
	}
	enc := json.NewEncoder(a.stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
	return 0
}

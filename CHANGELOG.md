# Changelog

All notable changes to covignore are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.1] - 2026-03-14
QoL fix after dogfooding on another project.

### Changed

- Filter summary line now always prints to stderr when entries are filtered (e.g. `covignore: filtered 2 entries (1 files) of 5 total`). Previously this only showed with `--verbose`. The per-file detail lines still require `--verbose`.
- Windows added to CI test matrix.
- Goreleaser config updated to use `formats` instead of deprecated `format`.
- Added `go fix -diff` check to CI lint job.

## [1.0.0] - 2026-03-14
Transition off private hosting and start over on GH now that this has scaled enough.

### Added

- Added compilation info (go version, architecture) and sweet ascii logo to version output for pizzazz reasons.
- Custom branded `--help` output with usage examples.
- `--check` integration test.
- GitHub Actions CI pipeline (Go 1.26 + stable, ubuntu + macOS).
- GitHub Actions release pipeline with goreleaser.

### Changed

- Reduced cognitive complexity again at the behest of Sonar.

### Fixed

- `--help` was misrouted as a `go test` flag when no `--` separator was present. Also it was returning nonzero exit code.
- another issue with bare `--` where it was being treated as a known flag.
- `--check` silently swallowed `WorkspaceModules()` errors instead of warning and falling back.
- `WalkDir` errors in `--check` were silently ignored.

## [0.9.0] - 2026-02-28
The go 1.26 update/quality improvements

### Added

- `--check` flag to list files matched by current `.covignore` patterns without running tests.
- `Matcher.Patterns()` accessor for safe cross-package pattern combination.
- `--dry-run` flag to preview filtering without writing output files.
- `--verbose` flag to show which entries are filtered.

### Changed

- Replaced `os.Chdir` with `t.Chdir()` in integration tests.
- Removed a bunch of empty `return` statements.
- Update to go 1.26, ran `go fix` on the codebase.
- Linted with SonarQube and refactored some "cognitive complexity"
- Replaced `filepath.Walk` with `filepath.WalkDir` for better performance.

## [0.8.0] - 2026-02-07

### Added

- `--preset=generated` built-in pattern set (protobuf, codegen, mocks, ent, sqlc).
- `PresetNames()` for listing available presets.
- Presets compose with `.covignore` - file-level negations override preset ignores.

### Changed

- Reorganize the README to start with simpler usage first.

## [0.7.0] - 2026-02-01
The monorepo update!

### Added

- Monorepo support via `go.work` workspace awareness.
- Per-module `.covignore` files by prioritizing ones with longest prefix, and thus are certainly child modules (TODO, check if this has any strange behavior with symlinks).
- `FindCovignore()` for locating the closest `.covignore` to a module directory.
- Multi-module `--check` output with per-module and total summaries.

## [0.6.0] - 2026-01-25

### Added

- `--json` flag for custom JSON out (coverage stats, file lists, filter counts). Might be useful in auditing since it contains info about the ignores.
- `--summary` flag for one-line coverage percentage.
- `--html=PATH` flag to generate standard HTML coverage report.

### Fixed

- Duplicate parsing of statement/count in `ComputeStats` - now uses `Entry.NumStmt`/`Count` populated at parse time.

## [0.5.0] - 2026-01-11

### Added

- `--min=N` coverage threshold flag - exits 1 if filtered coverage is below %.
- `threshold.Check()` with clear error messages.

### Fixed

- `--min` and `--summary` both work at the same time!

## [0.4.0] - 2025-12-28

### Added

- Pipe mode: read from stdin with `-`, write to stdout with `-o -`
- Full pipe chaining support: `go test ... | covignore - -o - | other-tool`
- `-o PATH` flag for custom output path.

### Changed

- Post-process mode now defaults to in-place overwrite (stdin => stdout).

## [0.3.0] - 2025-12-27

### Added

- Requested "post-process" mode: `covignore coverage.out` filters an existing profile in-place.
- `looksLikeCoverageFile()` func just in case someone has a package name that looks similar to a covignore file.

### Changed

- Default to `./...` when no packages are specified.
- Refactored structure to separate files a lot more cleanly.

### Fixed

- bug when `--` separator supplied but no args are present after it.

## [0.2.0] - 2025-11-28


### Added

- Wrapper mode: `covignore -- ./...` runs `go test` and filters the result.
- Automatic `-coverprofile` injection into `go test` invocations.
- flag splitting - single-dash flags (`-v`, `-race`) pass through to `go test`. Fixes growing confusion with tag intent.
- `--init` flag to scaffold a default `.covignore`.
- `--version` flag with build-time version injection via ldflags.

### Fixed

- Module prefix is now stripped from file paths before matching.

## [0.1.0] - 2025-11-15

### Added

- `.covignore` file format with gitignore-style glob patterns.
- `**`, `*`, `?` glob support.
- `!pattern` negation to re-include previously matched files.
- `coverage.Parse()` for reading Go coverage profiles
- `coverage.Filter()` for applying ignore rules to a profile.
- `coverage.Write()` / `WriteFile()` for output.
- removed doublestar dependency - custom glob matcher using `path.Match`.

## [0.0.1] - 2025-11-09

### Added

- Initial project structure.
- Basic `.covignore` parsing (comments, blank lines, glob patterns).
- `ignore.Matcher` with pattern evaluation in order.
- Proof-of-concept filtering of a hardcoded coverage profile.

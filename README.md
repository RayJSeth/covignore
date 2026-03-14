# covignore

Standardized coverage ignoring for Go projects via `.covignore` glob patterns.

## Why covignore?

Go projects accumulate generated code, mocks, and third-party wrappers that inflate or deflate coverage numbers. The lack of real singular solution in this space leads to - at best a sprawl of fragile grep, sed, or awk solutions, and at worst frustration with perceived false-positives that lead to "temporarily" (see forever) switching test gates off. The latter of those led me to make this project. But the project has grown large enough that I want to make it available generally.

covignore introduces a single `.covignore` file (think `.gitignore` for coverage) that:

- **Standardizes** what gets excluded across all environments
- **Composes** with `go test` transparently - wrap your command or post-process a profile
- **Pipes** - read from stdin, write to stdout, chain with other tools
- **Works in monorepos** - `go.work` awareness with per-module `.covignore` files
- **Runs everywhere** - zero dependencies, single binary, no runtime overhead
- **Gates filtered coverage** - build in `--min` flag for simple pass fail execution based on percentage of filtered coverage
- **Supports presets** - at this point there's only one preset for generated files, but could be expanded

## Install

```sh
go install github.com/RayJSeth/covignore/cmd/covignore@latest
```

Pre-built binaries for macOS, Linux, and Windows are available on the [Releases](https://github.com/RayJSeth/covignore/releases) page.

Requires Go 1.26 or later when building from source.

## Quickstart

```sh
# Zero-config: use the built-in preset to ignore generated code and mocks
covignore --preset=generated -- ./...

# Or create a .covignore for more granular control
covignore --init

# Run tests with coverage filtering
covignore -- ./...

# Post-process an existing coverage file
covignore coverage.out

# Chain coverage via pipe
go test -coverprofile=/dev/stdout ./... | covignore - --json
```

## .covignore

Plain text file with gitignore-style glob patterns.
Example common usage:

```gitignore
# Generated protobuf
**/*.pb.go

# Code generators
**/*_generated.go
**/*_gen.go

# Mocks
**/mock/**
**/mocks/**
**/*_mock.go
```

Lines starting with `#` are comments. Prefix a pattern with `!` to negate (re-include a previously matched file).

| Pattern | Matches |
|---------|---------|
| `**` | Zero or more path segments |
| `*` | Any characters within a single segment |
| `?` | A single character |
| `!{pattern}` | Negates a previous match (re-include) |

## Usage

### Wrapper mode (recommended usage)

```sh
covignore -- ./...
covignore --min=80 -- -race ./...
covignore --json -- -run TestIntegration ./...
```

Use `--` to separate covignore flags from `go test` flags. The separator is optional when there's no ambiguity - single-dash flags like `-v` and `-race` always pass through to `go test` because covignore only uses double-dash flags.

### Post-process mode

```sh
covignore coverage.out
```

Filters an existing coverage profile in-place.

### Pipe mode

```sh
# Read from file, write filtered profile to stdout
covignore -o - coverage.out

# Full pipeline
go test -coverprofile=/dev/stdout ./... | covignore - -o - | some-other-tool
```

Use `-` as the input to read from stdin. Use `-o -` to write the filtered profile to stdout.

### Coverage threshold

```sh
covignore --min=80 -- ./...
```

Exits with code 1 if filtered coverage is below the threshold.

### Presets

```sh
covignore --preset=generated -- ./...
```

Built-in pattern sets that can be used alongside `.covignore`. Currently available: `generated` (protobuf, codegen, mocks, ent, sqlc).

### Output formats

```sh
covignore --json -- ./...                # JSON with coverage stats + file lists
covignore --summary -- ./...             # One-line coverage percentage
covignore --html=coverage.html -- ./...  # HTML report saved to file
covignore -o filtered.out -- ./...       # Custom output path
```

### Inspect patterns

```sh
covignore --check
```

Lists all `.go` files in the module matched by the current `.covignore` patterns, without running tests. Particularly usefule in monorepo mode as those nested .covignore files can get hard to track!

### Other flags

```sh
covignore --dry-run -- ./...   # Preview filtering without writing
covignore --verbose -- ./...   # Show per-file filter details
covignore --version            # Print version
covignore --init               # Create default .covignore
```

## Flags

| Flag | Description |
|------|-------------|
| `--init` | Create a default `.covignore` file |
| `--min=N` | Minimum coverage threshold (exits 1 if below) |
| `--json` | Output coverage report as JSON |
| `--summary` | Print coverage summary line |
| `--html=PATH` | Write HTML coverage report to PATH |
| `--preset=NAME` | Use a built-in ignore preset |
| `-o PATH` | Output file path (default: `coverage.out`; use `-` for stdout) |
| `--dry-run` | Preview filtering without writing files |
| `--verbose` | Show per-file filter details (summary always prints when entries are filtered) |
| `--check` | List files matched by current patterns |
| `--version` | Print version |

## Monorepo Support

In a `go.work` workspace, covignore automatically discovers all modules and applies per-module `.covignore` files:

```
myproject/
├── go.work
├── .covignore          # fallback patterns
├── svcA/
│   ├── go.mod
│   └── .covignore      # svcA-specific patterns
└── svcB/
    ├── go.mod
    └── .covignore      # svcB-specific patterns
```

Each module's coverage entries are filtered using the closest `.covignore`. The `--check` flag inspects each module independently.

## CI

### GitHub Actions

```yaml
- name: Test with coverage
  run: |
    go install github.com/RayJSeth/covignore/cmd/covignore@latest
    covignore --min=80 -- ./...
```

### JSON output for downstream tools

```yaml
- name: Coverage report
  run: covignore --json -- ./... > coverage.json
```

### Build with version injection

```yaml
- name: Build
  run: |
    go build -ldflags="-X github.com/RayJSeth/covignore/internal/cli.Version=${{ github.ref_name }}" \
      ./cmd/covignore
```

## Configuration

covignore uses flags and `.covignore` files for all configuration. There is no separate config file by design - your CI script is your config - leaves the control local and traceable.

For shared team defaults, check `.covignore` into version control (this is the intended use case) and set flags in your CI configuration or Makefile:

```makefile
.PHONY: test
test:
	covignore --min=80 --preset=generated -- -race ./...
```

## How It Works

1. Runs `go test -coverprofile=.covignore.raw <your args>`
2. Parses the raw coverage profile
3. Loads `.covignore` patterns (+ optional preset)
4. Strips module prefixes and filters matching entries
5. Writes filtered `coverage.out`
6. Reports coverage, checks threshold, generates HTML - as requested

In post-process or pipe mode, step 1 is skipped in lieu of an existing profile.

## License

MIT - see [LICENSE](LICENSE).

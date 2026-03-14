# Contributing to covignore

## Development

### Prerequisites

- Go 1.26 or later

### Build

```sh
make build
```

### Test

```sh
make test
```

### Coverage (dogfooding)

```sh
make cover
```

This runs covignore's own test suite through covignore itself - dogfooding FTW!

### Lint

```sh
make lint
```

## Making Changes

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Run `make test && make lint`
5. Submit a pull request

## Guidelines

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep the zero-dependency constraint - no external modules
- Add tests for new functionality
- Keep the README up to date with any user-facing changes
- Update [CHANGELOG.md](CHANGELOG.md) under an `## [Unreleased]` heading for every user-facing change

BINARY := bin/covignore
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags="-X github.com/RayJSeth/covignore/internal/cli.Version=$(VERSION)"
COVERAGE_DIR := coverage

.PHONY: all
all: build

.PHONY: build
build:
	@mkdir -p $(dir $(BINARY))
	go build $(LDFLAGS) -o $(BINARY) ./cmd/covignore

.PHONY: install
install:
	go install $(LDFLAGS) ./cmd/covignore

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: make-coverage-dir
make-coverage-dir:
	mkdir -p $(COVERAGE_DIR)

.PHONY: cover
cover: build make-coverage-dir
	$(BINARY) -o $(COVERAGE_DIR)/coverage.out --preset=generated --summary -- ./...

.PHONY: cover-html
cover-html: build make-coverage-dir
	$(BINARY) -o $(COVERAGE_DIR)/coverage.out --preset=generated --html=$(COVERAGE_DIR)/coverage.html --summary -- ./...

.PHONY: cover-check
cover-check: build
	$(BINARY) --check

.PHONY: lint
lint:
	go vet ./...

.PHONY: clean
clean:
	rm -f $(BINARY) .covignore.raw
	rm -rf bin/ $(COVERAGE_DIR)

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

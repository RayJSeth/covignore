BINARY := bin/covignore
COVERAGE_DIR := coverage

.PHONY: all
all: build

.PHONY: build
build:
	@mkdir -p $(dir $(BINARY))
	go build -o $(BINARY) ./cmd/covignore

.PHONY: install
install:
	go install ./cmd/covignore

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
lint: vet fmt-check fix-check

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt-check
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: fix-check
fix-check:
	@OUTPUT=$$(go fix -diff ./... 2>&1); \
	if [ -n "$$OUTPUT" ]; then \
		echo "go fix found modernization opportunities:"; \
		echo ""; \
		echo "$$OUTPUT"; \
		echo ""; \
		echo "Run 'go fix ./...' locally and commit the changes."; \
		exit 1; \
	fi

.PHONY: ci-build
ci-build:
	go build -o /dev/null ./cmd/covignore

.PHONY: ci-cover
ci-cover: build make-coverage-dir
	$(BINARY) -o $(COVERAGE_DIR)/coverage.out --preset=generated --summary -- -race ./...

.PHONY: clean
clean:
	rm -f $(BINARY) .covignore.raw
	rm -rf bin/ $(COVERAGE_DIR)

.PHONY: release-snapshot
release-snapshot:
	goreleaser release --snapshot --clean

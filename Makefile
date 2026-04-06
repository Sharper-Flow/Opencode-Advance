# OpenCode Advance — Makefile
#
# Standard targets for local development. The real build pipeline runs in CI
# via goreleaser (configured in Phase 7).

.PHONY: help build test vet fmt check clean scaffold-check

help:
	@echo "OpenCode Advance — Makefile targets"
	@echo ""
	@echo "  build          Build the oca binary to ./bin/oca"
	@echo "  test           Run all Go tests"
	@echo "  vet            Run go vet"
	@echo "  fmt            Run gofmt on all Go files"
	@echo "  check          Run vet + fmt + test (full verification)"
	@echo "  scaffold-check Quick sanity check that the scaffold builds"
	@echo "  clean          Remove build artifacts"
	@echo ""

build:
	@mkdir -p bin
	go build -o bin/oca ./cmd/oca

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "gofmt violations:"; \
		gofmt -l .; \
		exit 1; \
	fi

check: vet fmt-check test

scaffold-check: build
	@./bin/oca

clean:
	rm -rf bin/ dist/
	rm -f coverage.out

# OpenCode Advance — Makefile
#
# Standard targets for local development. The real build pipeline runs in CI
# via goreleaser (configured in Phase 7).

.PHONY: help build test shell-test vet fmt check clean baseline-check

help:
	@echo "OpenCode Advance — Makefile targets"
	@echo ""
	@echo "  build          Build the oca binary to ./bin/oca"
	@echo "  test           Run all Go tests"
	@echo "  shell-test     Run shell verification tests"
	@echo "  vet            Run go vet"
	@echo "  fmt            Run gofmt on all Go files"
	@echo "  check          Run vet + fmt + test (full verification)"
	@echo "  baseline-check Quick sanity check that the baseline CLI builds and runs"
	@echo "  clean          Remove build artifacts"
	@echo ""

build:
	@mkdir -p bin
	go build -o bin/oca ./cmd/oca

test:
	go test ./...

shell-test:
	bash tests/shell/brand_helpers_test.sh
	bash tests/shell/verification_workflow_test.sh

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

check: vet fmt-check test shell-test

baseline-check: build
	@./bin/oca

clean:
	rm -rf bin/ dist/
	rm -f coverage.out

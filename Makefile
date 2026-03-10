GO ?= go
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null)
ifeq ($(GOLANGCI_LINT),)
GOLANGCI_LINT := $(shell $(GO) env GOPATH)/bin/golangci-lint
endif

.PHONY: test vet lint check quality-gate smoke-gate load-steady load-spike load-soak failure-drill

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || (echo "golangci-lint not found. install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8" && exit 1)
	$(GOLANGCI_LINT) run

check: test vet lint

quality-gate:
	./scripts/quality_gate.sh

smoke-gate:
	./scripts/smoke_gate.sh

load-steady:
	./scripts/load_gate.sh steady

load-spike:
	./scripts/load_gate.sh spike

load-soak:
	./scripts/load_gate.sh soak

failure-drill:
	./scripts/failure_drill.sh

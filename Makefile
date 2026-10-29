# sdd build tooling. `make verify` needs only the Go toolchain; `make check` also runs golangci-lint v2.
.NOTPARALLEL:
.PHONY: build install test vet fmt fmt-check lint verify check dist clean

MODULE  := github.com/vikashkumar21/sdd
VERSION ?= 0.1.0
LDFLAGS := -s -w -X $(MODULE)/internal/version.Version=$(VERSION)
BIN     := bin/sdd

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/sdd

# Installs into $(go env GOBIN), or $(go env GOPATH)/bin when GOBIN is unset. Make sure that directory is on your PATH.
install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/sdd

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -s -w cmd internal

fmt-check:
	@test -z "$$(gofmt -s -l cmd internal)" || { echo "gofmt -s needed on:"; gofmt -s -l cmd internal; exit 1; }

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed (v2 expected); see https://golangci-lint.run/"; exit 1; }
	golangci-lint run

# verify = everything that needs only the Go toolchain; check = verify plus the linter.
verify: fmt-check vet test

check: fmt-check vet lint test

# Cross-compiles static binaries (no cgo) for macOS and Linux and writes SHA256SUMS.
dist:
	rm -rf dist && mkdir -p dist
	@for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do \
		os=$${t%/*}; arch=$${t#*/}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/sdd_$${os}_$${arch} ./cmd/sdd || exit 1; \
	done
	@cd dist && if command -v sha256sum >/dev/null 2>&1; then sha256sum sdd_* > SHA256SUMS; else shasum -a 256 sdd_* > SHA256SUMS; fi

clean:
	rm -rf bin dist

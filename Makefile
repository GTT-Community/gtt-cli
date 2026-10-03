GO ?= go

.PHONY: build test test-unit vet

build:
	$(GO) build -o gtt ./cmd/gtt

vet:
	$(GO) vet ./...

# Unit and architecture tests: no Bootstrap needed, a few seconds.
test-unit:
	$(GO) test ./internal/... ./test/arch/

# Everything, including the integration suite against a real Bootstrap
# catalog (GTT_TEST_BOOTSTRAP, default ../gtt-bootstrap). Each Bootstrap
# validation takes several seconds, so the default 10 minute limit is raised.
test: vet
	$(GO) test -timeout 30m ./...

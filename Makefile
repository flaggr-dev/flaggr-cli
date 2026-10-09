BINARY = flaggr

# govulncheck version for `make vuln` (CI uses the same one).
GOVULNCHECK_VERSION ?= v1.8.0

default: build

# Builds ./flaggr. It reports the version Go records (see README.md); release
# builds come only from the Release workflow.
build:
	go build -o $(BINARY) ./cmd/flaggr

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test -race -count=1 -cover ./...

# Reports known vulnerabilities in code the CLI calls.
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

# Regenerates internal/flaggrv1 from proto/ with buf (buf.gen.yaml).
generate:
	buf generate

# Builds the release archives in dist/ without publishing anything.
snapshot:
	goreleaser release --snapshot --clean

.PHONY: default build fmt vet test vuln generate snapshot

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

# Writes THIRD_PARTY_NOTICES, the licenses of Go and of the modules compiled
# into flaggr, which every release archive ships. Run it after changing
# dependencies: CI fails while the committed file is out of date.
notices:
	./scripts/third-party-notices.sh >THIRD_PARTY_NOTICES.tmp && mv THIRD_PARTY_NOTICES.tmp THIRD_PARTY_NOTICES || { rm -f THIRD_PARTY_NOTICES.tmp; exit 1; }

# Builds the release archives in dist/ without publishing anything.
snapshot:
	goreleaser release --snapshot --clean

.PHONY: default build fmt vet test vuln generate notices snapshot

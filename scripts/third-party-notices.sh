#!/bin/sh
# Prints THIRD_PARTY_NOTICES, which every release archive ships next to
# flaggr's LICENSE: the license texts (and any NOTICE or PATENTS file) of Go,
# whose standard library and runtime are in every Go binary, and of each
# module compiled into the binaries GoReleaser builds (.goreleaser.yml:
# darwin, linux and windows on amd64 and arm64).
#
# Run `make notices` after changing dependencies; CI fails while the
# committed file differs from this script's output.
set -eu

main_package=./cmd/flaggr
rule='================================================================================'

# "<module path> <module directory>" for each module linked into a release
# binary. Some are linked on one OS only (cobra uses mousetrap on Windows).
listed=$(mktemp)
trap 'rm -f "$listed"' EXIT
for goos in darwin linux windows; do
	for goarch in amd64 arm64; do
		CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go list -deps \
			-f '{{with .Module}}{{if not .Main}}{{.Path}} {{.Dir}}{{end}}{{end}}' \
			"$main_package" >>"$listed"
	done
done
modules=$(LC_ALL=C sort -u "$listed")

# The license, notice and patent files at the top of directory $1.
license_files() {
	find "$1" -maxdepth 1 -type f \( -iname 'LICENSE*' -o -iname 'LICENCE*' \
		-o -iname 'COPYING*' -o -iname 'NOTICE*' -o -iname 'PATENTS*' \) |
		LC_ALL=C sort
}

# A section for $1 (its name) with the license files in directory $2.
section() {
	if [ -z "$(license_files "$2")" ]; then
		echo "third-party-notices.sh: no license file in $2, for $1" >&2
		exit 1
	fi
	printf '\n%s\n%s\n%s\n' "$rule" "$1" "$rule"
	license_files "$2" | while IFS= read -r file; do
		printf '\n--- %s ---\n\n' "${file##*/}"
		cat "$file"
	done
}

cat <<'EOF'
Third-party notices for flaggr

flaggr is MIT-licensed (see LICENSE). Its binaries also contain the Go
standard library and runtime and the Go modules below, which come under
their own licenses: their texts follow.

scripts/third-party-notices.sh generates this file (make notices).
EOF

section "Go: the standard library and runtime (https://go.dev)" "$(go env GOROOT)"

printf '%s\n' "$modules" | while read -r path dir; do
	[ -n "$path" ] || continue
	section "$path" "$dir"
done

#!/bin/sh
# Installs the Flaggr CLI (flaggr) on macOS or Linux from its GitHub releases:
#
#   curl -fsSL https://raw.githubusercontent.com/flaggr-dev/flaggr-cli/main/install.sh | sh
#
# FLAGGR_VERSION      the release to install, such as 0.5.0 (default: the latest)
# FLAGGR_INSTALL_DIR  where to install flaggr (default: /usr/local/bin, with
#                     sudo when that directory isn't writable)
#
# The archive is checked against the release's checksums.txt before anything
# is installed.
set -eu

REPO="flaggr-dev/flaggr-cli"
BINARY="flaggr"
INSTALL_DIR="${FLAGGR_INSTALL_DIR:-/usr/local/bin}"

fail() {
  echo "Error: $*" >&2
  exit 1
}

download() {
  curl --proto '=https' --tlsv1.2 -fsSL "$@"
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

os="$(uname -s)"
case "$os" in
  Linux*) os="linux" ;;
  Darwin*) os="darwin" ;;
  *) fail "unsupported OS: ${os}. Download flaggr for your system from https://github.com/${REPO}/releases" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: ${arch}. Release builds are for amd64 and arm64" ;;
esac

if [ -n "${FLAGGR_VERSION:-}" ]; then
  version="${FLAGGR_VERSION#v}"
else
  echo "Finding the latest flaggr release..."
  # releases/latest redirects to the latest release's tag, or to the release
  # list when there's no release.
  latest="$(download -I -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")" ||
    fail "could not reach https://github.com/${REPO}/releases/latest"
  case "$latest" in
    */releases/tag/v*) version="${latest##*/releases/tag/v}" ;;
    *) fail "found no flaggr release. With Go 1.26 or later: go install github.com/${REPO}/cmd/flaggr@latest" ;;
  esac
fi

case "$version" in
  "" | *[!0-9A-Za-z.+-]*) fail "not a release version: ${version}" ;;
esac

archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/v${version}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

echo "Downloading flaggr ${version} (${os}/${arch})..."
download -o "${tmp}/${archive}" "${base}/${archive}" || fail "could not download ${base}/${archive}"
download -o "${tmp}/checksums.txt" "${base}/checksums.txt" || fail "could not download ${base}/checksums.txt"

expected="$(awk -v name="$archive" '$2 == name { print $1 }' "${tmp}/checksums.txt")"
[ -n "$expected" ] || fail "checksums.txt has no entry for ${archive}"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp}/${archive}" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmp}/${archive}" | awk '{ print $1 }')"
else
  fail "sha256sum or shasum is needed to check the download"
fi
[ "$actual" = "$expected" ] || fail "${archive} doesn't match checksums.txt (expected ${expected}, got ${actual})"

tar -xzf "${tmp}/${archive}" -C "$tmp" "$BINARY" || fail "could not extract ${BINARY} from ${archive}"
chmod 755 "${tmp}/${BINARY}"

# install, not mv: mv keeps the downloaded file's owner, so a sudo install
# would leave a binary the user can rewrite inside a root-owned directory.
sudo=""
if [ -d "$INSTALL_DIR" ]; then
  [ -w "$INSTALL_DIR" ] || sudo="sudo"
elif ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
  sudo="sudo"
fi
if [ -n "$sudo" ]; then
  command -v sudo >/dev/null 2>&1 ||
    fail "can't write to ${INSTALL_DIR}: set FLAGGR_INSTALL_DIR to a directory you can write to"
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mkdir -p "$INSTALL_DIR"
  sudo install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  install -m 0755 "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "  flaggr ${version} installed to ${INSTALL_DIR}/${BINARY}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "  ${INSTALL_DIR} isn't on your PATH: add it to run flaggr by name." ;;
esac
echo ""
echo "  Get started:"
echo "    flaggr login       # sign in through your browser"
echo "    flaggr status      # check the connection"
echo "    flaggr projects    # list your projects"
echo ""

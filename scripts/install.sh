#!/usr/bin/env bash
# Install Logy from GitHub Releases into ~/.local/bin (or LOGY_INSTALL_DIR).
set -euo pipefail

REPO="${LOGY_GITHUB_REPO:-LuizFer1/logy}"
INSTALL_DIR="${LOGY_INSTALL_DIR:-${HOME}/.local/bin}"
API_BASE="https://api.github.com"

auth_headers=()
if [[ -n "${GH_TOKEN:-}" ]]; then
  auth_headers=(-H "Authorization: Bearer ${GH_TOKEN}")
elif [[ -n "${GITHUB_TOKEN:-}" ]]; then
  auth_headers=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
fi

die() {
  echo "error: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

need_cmd curl
need_cmd tar
need_cmd mktemp

uname_s="$(uname -s)"
case "${uname_s}" in
  Linux*) OS=linux ;;
  Darwin*) OS=darwin ;;
  *) die "unsupported OS: ${uname_s} (supported: Linux, macOS)" ;;
esac

uname_m="$(uname -m)"
case "${uname_m}" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) die "unsupported architecture: ${uname_m} (supported: amd64, arm64)" ;;
esac

github_get() {
  local url="$1"
  curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    "${auth_headers[@]}" \
    "$url"
}

normalize_tag() {
  local t="$1"
  t="${t#"${t%%[![:space:]]*}"}"
  t="${t%"${t##*[![:space:]]}"}"
  case "$t" in
    v*|V*) printf '%s\n' "$t" ;;
    *) printf 'v%s\n' "$t" ;;
  esac
}

strip_v() {
  local t="$1"
  t="${t#v}"
  t="${t#V}"
  printf '%s\n' "$t"
}

json_tag_name() {
  # Extract "tag_name":"..." without requiring jq.
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

if [[ -n "${LOGY_VERSION:-}" ]]; then
  TAG="$(normalize_tag "${LOGY_VERSION}")"
else
  echo "Resolving latest release for ${REPO}..."
  latest_json="$(github_get "${API_BASE}/repos/${REPO}/releases/latest")" || \
    die "failed to fetch latest release (is the repo public? set GH_TOKEN if rate-limited)"
  TAG="$(printf '%s' "$latest_json" | json_tag_name)"
  [[ -n "$TAG" ]] || die "could not parse tag_name from GitHub API response"
fi

VERSION="$(strip_v "$TAG")"
ASSET="logy_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ASSET_URL="${BASE_URL}/${ASSET}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${ASSET}..."
curl -fsSL -o "${TMPDIR}/${ASSET}" "$ASSET_URL" || \
  die "failed to download ${ASSET_URL}"

echo "Downloading checksums.txt..."
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" || \
  die "failed to download ${CHECKSUMS_URL}"

want_hash="$(
  awk -v f="$ASSET" '
    NF >= 2 {
      name = $NF
      sub(/^\*\.?/, "", name)
      if (name == f) { print $1; exit }
    }
  ' "${TMPDIR}/checksums.txt"
)"
[[ -n "$want_hash" ]] || die "checksum not found for ${ASSET} in checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  got_hash="$(sha256sum "${TMPDIR}/${ASSET}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  got_hash="$(shasum -a 256 "${TMPDIR}/${ASSET}" | awk '{print $1}')"
else
  die "need sha256sum or shasum to verify download"
fi

got_lc="$(printf '%s' "$got_hash" | tr '[:upper:]' '[:lower:]')"
want_lc="$(printf '%s' "$want_hash" | tr '[:upper:]' '[:lower:]')"
[[ "$got_lc" == "$want_lc" ]] || die "sha256 mismatch: got ${got_hash} want ${want_hash}"
echo "Checksum OK"

mkdir -p "$INSTALL_DIR"
tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"

BIN_SRC=""
if [[ -f "${TMPDIR}/logy" ]]; then
  BIN_SRC="${TMPDIR}/logy"
else
  BIN_SRC="$(find "$TMPDIR" -type f -name logy | head -n 1 || true)"
fi
[[ -n "$BIN_SRC" && -f "$BIN_SRC" ]] || die "logy binary not found in archive"

install -m 755 "$BIN_SRC" "${INSTALL_DIR}/logy"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo
    echo "Note: ${INSTALL_DIR} is not on your PATH."
    echo "Add it for the current shell:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo "Or append that line to your shell profile (~/.bashrc, ~/.zshrc, etc.)."
    ;;
esac

echo
echo "Installed logy ${VERSION} to ${INSTALL_DIR}/logy"
"${INSTALL_DIR}/logy" version

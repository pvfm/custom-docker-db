#!/bin/sh
# Installs custom-docker-db from the latest GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/pvfm/custom-docker-db/main/install.sh | sh
#
# Environment:
#   CDD_VERSION      release tag to install, e.g. v1.0.0 (default: latest)
#   CDD_INSTALL_DIR  where to put the binary (default: $HOME/.local/bin)
#   CDD_BASE_URL     download base URL, for mirrors and tests
set -eu

REPO="pvfm/custom-docker-db"
BIN="custom-docker-db"

say() { printf '%s\n' "$*"; }
die() { printf 'erro: %s\n' "$*" >&2; exit 1; }

os=$(uname -s)
[ "$os" = "Linux" ] || die "sistema não suportado: $os (só Linux por enquanto)"

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *) die "arquitetura não suportada: $(uname -m) (esperado x86_64 ou aarch64)" ;;
esac

if [ -n "${CDD_BASE_URL:-}" ]; then
  base="$CDD_BASE_URL"
elif [ -n "${CDD_VERSION:-}" ]; then
  base="https://github.com/$REPO/releases/download/$CDD_VERSION"
else
  base="https://github.com/$REPO/releases/latest/download"
fi

install_dir="${CDD_INSTALL_DIR:-$HOME/.local/bin}"
asset="${BIN}_linux_${arch}.tar.gz"

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
else
  die "curl ou wget é necessário"
fi

if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  die "sha256sum ou shasum é necessário para conferir o download"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Baixando $asset..."
fetch "$base/$asset" "$tmp/$asset" || die "não foi possível baixar $base/$asset"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || die "não foi possível baixar $base/checksums.txt"

expected=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$expected" ] || die "$asset não consta em checksums.txt"
actual=$(sha256 "$tmp/$asset")
[ "$expected" = "$actual" ] || die "checksum não confere para $asset (esperado $expected, obtido $actual); nada foi instalado"

tar -xzf "$tmp/$asset" -C "$tmp" "$BIN" || die "não foi possível extrair $BIN de $asset"

mkdir -p "$install_dir"
install -m 755 "$tmp/$BIN" "$install_dir/$BIN"
say "Instalado em $install_dir/$BIN"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) say "Aviso: $install_dir não está no seu PATH. Adicione com:"
     say "  export PATH=\"$install_dir:\$PATH\"" ;;
esac
"$install_dir/$BIN" --version

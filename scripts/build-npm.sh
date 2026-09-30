#!/bin/sh
# Builds the npm packages under dist/npm for a version (without the leading v).
#
#   scripts/build-npm.sh 1.0.0
#
# Produces the platform packages (each carrying the Go binary) and the main
# package. Publish the platform packages first, then the main one.
set -eu

version="${1:?uso: scripts/build-npm.sh <versão sem o v>}"
root=$(cd "$(dirname "$0")/.." && pwd)
out="$root/dist/npm"
rm -rf "$out"

build_platform() { # <npm os-cpu suffix> <GOARCH> <npm cpu>
  suffix="$1"; goarch="$2"; cpu="$3"
  dir="$out/cdd-$suffix"
  mkdir -p "$dir/bin"
  (cd "$root" && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build -trimpath \
    -ldflags "-s -w -X github.com/pvfm/custom-docker-db/internal/cli.Version=$version" \
    -o "$dir/bin/custom-docker-db" ./cmd/custom-docker-db)
  cat > "$dir/package.json" <<JSON
{
  "name": "@pvfm/cdd-$suffix",
  "version": "$version",
  "description": "custom-docker-db binary for $suffix",
  "license": "MIT",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/pvfm/custom-docker-db.git"
  },
  "os": ["linux"],
  "cpu": ["$cpu"],
  "files": ["bin"]
}
JSON
  cp "$root/LICENSE" "$dir/LICENSE"
}

build_platform linux-x64 amd64 x64
build_platform linux-arm64 arm64 arm64

main="$out/custom-docker-db"
mkdir -p "$main"
cp -R "$root/npm/main/bin" "$main/bin"
cp "$root/LICENSE" "$root/README.md" "$main/"
sed "s/__VERSION__/$version/g" "$root/npm/main/package.json" > "$main/package.json"

echo "Pacotes gerados em $out:"
ls "$out"

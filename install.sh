#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

fail() { echo "install: $1" >&2; exit 1; }

command -v go >/dev/null     || fail "Go is not installed — see https://go.dev/doc/install"
command -v docker >/dev/null || fail "Docker is not installed — see https://docs.docker.com/engine/install/"
docker info >/dev/null 2>&1  || fail "Docker is installed but not running (or your user can't access it)"
command -v gcloud >/dev/null || fail "gcloud CLI is not installed — see https://cloud.google.com/sdk/docs/install"

echo "Building huginn..."
go build -o huginn ./cmd/huginn
echo "Built $(pwd)/huginn"

./huginn init

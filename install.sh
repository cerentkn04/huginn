#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

fail() { echo "install: $1" >&2; exit 1; }

confirm() { # confirm "question" -> 0 if yes
  local reply
  read -r -p "$1 [y/N]: " reply
  [[ "$reply" == "y" || "$reply" == "Y" || "$reply" == "yes" ]]
}

install_docker() {
  command -v apt-get >/dev/null || fail \
    "Docker is not installed, and automatic installation only supports Debian/Ubuntu.
     Install it manually: https://docs.docker.com/engine/install/"

  echo "Docker is not installed."
  echo "Huginn can install it using Docker's official script (https://get.docker.com),"
  echo "then add '$USER' to the 'docker' group. This requires sudo."
  confirm "Install Docker now?" || fail \
    "Docker is required — see https://docs.docker.com/engine/install/"

  command -v curl >/dev/null || {
    echo "Installing curl..."
    sudo apt-get update -qq
    sudo apt-get install -y curl
  }

  echo "Downloading and running Docker's install script..."
  curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
  sudo sh /tmp/get-docker.sh
  rm -f /tmp/get-docker.sh

  sudo systemctl enable --now docker
  sudo usermod -aG docker "$USER"

  echo
  echo "Docker installed, and '$USER' was added to the 'docker' group."
  echo "Group membership only applies to new login sessions, so:"
  echo
  echo "    log out and back in (or run: newgrp docker)"
  echo "    then re-run ./install.sh"
  echo
  exit 0
}

# --- Docker ---------------------------------------------------------------
if ! command -v docker >/dev/null; then
  install_docker
fi

if ! docker info >/dev/null 2>&1; then
  if sudo docker info >/dev/null 2>&1; then
    fail "Docker is running, but '$USER' can't access it.
     Run: sudo usermod -aG docker $USER
     Then log out and back in (or run: newgrp docker) and re-run ./install.sh"
  fi
  echo "Docker is installed but not running. Trying to start it..."
  sudo systemctl start docker || fail "could not start Docker — check: systemctl status docker"
  docker info >/dev/null 2>&1 || fail "Docker still isn't responding — check: systemctl status docker"
fi

# --- gcloud ---------------------------------------------------------------
command -v gcloud >/dev/null || fail \
  "gcloud CLI is not installed — see https://cloud.google.com/sdk/docs/install"

# --- Go / build -----------------------------------------------------------
if [ -x ./huginn ]; then
  echo "Found existing ./huginn binary — skipping build."
elif command -v go >/dev/null; then
  echo "Building huginn..."
  go build -o huginn ./cmd/huginn
  echo "Built $(pwd)/huginn"
else
  fail "No ./huginn binary found and Go is not installed. Either:
     - download a prebuilt binary: curl -L -o huginn https://github.com/cerentkn04/huginn/releases/latest/download/huginn && chmod +x huginn
     - or install Go: https://go.dev/doc/install"
fi

./huginn init

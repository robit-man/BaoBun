#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

log() {
  printf "[setup] %s\n" "$1"
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

SUDO=""
if [[ "$(id -u)" -ne 0 ]] && has_cmd sudo; then
  SUDO="sudo"
fi

install_linux_deps() {
  if has_cmd apt-get; then
    ${SUDO} apt-get update
    ${SUDO} apt-get install -y ca-certificates curl git golang-go nodejs npm
    return
  fi

  if has_cmd dnf; then
    ${SUDO} dnf install -y ca-certificates curl git golang nodejs npm
    return
  fi

  if has_cmd yum; then
    ${SUDO} yum install -y ca-certificates curl git golang nodejs npm
    return
  fi

  if has_cmd pacman; then
    ${SUDO} pacman -Sy --noconfirm ca-certificates curl git go nodejs npm
    return
  fi

  if has_cmd zypper; then
    ${SUDO} zypper --non-interactive install ca-certificates curl git go nodejs npm
    return
  fi

  echo "Unsupported Linux package manager. Install Go + Node.js + npm manually."
  exit 1
}

install_macos_deps() {
  if ! has_cmd brew; then
    log "Homebrew not found. Installing Homebrew."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    if [[ -x /opt/homebrew/bin/brew ]]; then
      eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [[ -x /usr/local/bin/brew ]]; then
      eval "$(/usr/local/bin/brew shellenv)"
    fi
  fi

  brew update
  brew install go node
}

install_prereqs() {
  if has_cmd go && has_cmd node && has_cmd npm; then
    log "Go, Node.js and npm already installed."
    return
  fi

  case "$(uname -s)" in
    Linux*) install_linux_deps ;;
    Darwin*) install_macos_deps ;;
    *)
      echo "Unsupported OS. Use Windows setup script or install dependencies manually."
      exit 1
      ;;
  esac
}

build_project() {
  log "Building web UI."
  cd "${REPO_ROOT}/internal/webui"
  npm ci --no-audit --no-fund
  npm run build

  log "Compiling Go binaries."
  cd "${REPO_ROOT}"
  mkdir -p bin
  go mod download
  go build -o ./bin/baobun-client ./cmd/client
  go build -o ./bin/baobun-maker ./cmd/maker
}

install_prereqs

if ! has_cmd go || ! has_cmd npm; then
  echo "Go and npm must be available after installation. Exiting."
  exit 1
fi

build_project

if [[ "$(uname -s)" == "Darwin" || "$(uname -s)" == "Linux" ]]; then
  log "Done."
  log "Run: ./bin/baobun-client"
fi

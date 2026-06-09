#!/bin/sh
set -eu

MODE="both"
PREFIX="user"

usage() {
  printf '%s\n' "Usage: ./scripts/install.sh [--gui|--cli|--both] [--user|--system]"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --gui) MODE="gui" ;;
    --cli) MODE="cli" ;;
    --both) MODE="both" ;;
    --user) PREFIX="user" ;;
    --system) PREFIX="system" ;;
    -h|--help) usage; exit 0 ;;
    *) printf '%s\n' "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
  shift
done

if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' "Go is required to build Tar.gz Installer." >&2
  exit 1
fi

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BUILD_DIR="$ROOT_DIR/build"
mkdir -p "$BUILD_DIR"

build_cli() {
  printf '%s\n' "Building CLI..."
  (cd "$ROOT_DIR" && go build -buildvcs=false -o "$BUILD_DIR/targz-installer" ./cmd/targz-installer)
}

build_gui() {
  printf '%s\n' "Building GUI..."
  (cd "$ROOT_DIR" && go build -buildvcs=false -o "$BUILD_DIR/targz-installer-gui" ./cmd/targz-installer-gui)
}

install_file() {
  src="$1"
  dest="$2"
  mode="$3"
  if [ "$PREFIX" = "system" ]; then
    sudo install -Dm "$mode" "$src" "$dest"
  else
    install -Dm "$mode" "$src" "$dest"
  fi
}

install_cli() {
  if [ "$PREFIX" = "system" ]; then
    install_file "$BUILD_DIR/targz-installer" "/usr/local/bin/targz-installer" 0755
  else
    install_file "$BUILD_DIR/targz-installer" "$HOME/.local/bin/targz-installer" 0755
  fi
}

install_gui() {
  if [ "$PREFIX" = "system" ]; then
    install_file "$BUILD_DIR/targz-installer-gui" "/usr/local/bin/targz-installer-gui" 0755
    install_file "$ROOT_DIR/assets/logo.svg" "/usr/local/share/icons/hicolor/scalable/apps/targz-installer.svg" 0644
    install_file "$ROOT_DIR/packaging/targz-installer.desktop" "/usr/local/share/applications/targz-installer.desktop" 0644
  else
    install_file "$BUILD_DIR/targz-installer-gui" "$HOME/.local/bin/targz-installer-gui" 0755
    install_file "$ROOT_DIR/assets/logo.svg" "$HOME/.local/share/icons/hicolor/scalable/apps/targz-installer.svg" 0644
    install_file "$ROOT_DIR/packaging/targz-installer.desktop" "$HOME/.local/share/applications/targz-installer.desktop" 0644
  fi
}

case "$MODE" in
  cli)
    build_cli
    install_cli
    ;;
  gui)
    build_gui
    install_gui
    ;;
  both)
    build_cli
    build_gui
    install_cli
    install_gui
    ;;
esac

printf '%s\n' "Installed Tar.gz Installer ($MODE, $PREFIX)."

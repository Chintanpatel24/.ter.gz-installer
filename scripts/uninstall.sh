#!/bin/sh
set -eu

MODE="both"
PREFIX="user"

usage() {
  printf '%s\n' "Usage: ./scripts/uninstall.sh [--gui|--cli|--both] [--user|--system]"
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

remove_path() {
  path="$1"
  if [ "$PREFIX" = "system" ]; then
    sudo rm -f "$path"
  else
    rm -f "$path"
  fi
}

remove_cli() {
  if [ "$PREFIX" = "system" ]; then
    remove_path "/usr/local/bin/targz-installer"
  else
    remove_path "$HOME/.local/bin/targz-installer"
  fi
}

remove_gui() {
  if [ "$PREFIX" = "system" ]; then
    remove_path "/usr/local/bin/targz-installer-gui"
    remove_path "/usr/local/share/icons/hicolor/scalable/apps/targz-installer.svg"
    remove_path "/usr/local/share/applications/targz-installer.desktop"
  else
    remove_path "$HOME/.local/bin/targz-installer-gui"
    remove_path "$HOME/.local/share/icons/hicolor/scalable/apps/targz-installer.svg"
    remove_path "$HOME/.local/share/applications/targz-installer.desktop"
  fi
}

case "$MODE" in
  cli) remove_cli ;;
  gui) remove_gui ;;
  both) remove_cli; remove_gui ;;
esac

printf '%s\n' "Removed Tar.gz Installer ($MODE, $PREFIX)."

# Tar.gz Installer

Tar.gz Installer is a small open source Linux app that installs `.tar.gz` application bundles from a clean GUI or a CLI.

The GUI opens from the desktop menu, accepts a dragged `.tar.gz` file, and installs it as a Linux application. The CLI provides the same installer behavior for terminal users.

## What It Installs

The installer extracts a `.tar.gz` archive, looks for a runnable file such as `AppRun`, a matching executable, or a binary in `bin/`, then creates:

- an application folder
- a launcher script
- a `.desktop` menu entry
- a local command symlink when possible

User installs do not need a password and go into `~/.local`. System installs use `pkexec` or `sudo` and may ask for your password.

## Install From GitHub

Clone the repository, then choose what to install:

```sh
./scripts/install.sh --gui
./scripts/install.sh --cli
./scripts/install.sh --both
```

By default, files are installed for the current user. To install system-wide:

```sh
./scripts/install.sh --both --system
```

The GUI uses Fyne. On some distributions you may need the standard desktop development packages required by Fyne before building.

## Use The GUI

Open **Tar.gz Installer** from your application menu, drag a `.tar.gz` file into the window, choose user or system install, then click install.

## Use The CLI

```sh
targz-installer install ~/Downloads/example.tar.gz
targz-installer install ~/Downloads/example.tar.gz --system
targz-installer install ~/Downloads/example.tar.gz --name example-app
```

## Uninstall This Tool

```sh
./scripts/uninstall.sh --gui
./scripts/uninstall.sh --cli
./scripts/uninstall.sh --both
```

For system-wide removal:

```sh
./scripts/uninstall.sh --both --system
```

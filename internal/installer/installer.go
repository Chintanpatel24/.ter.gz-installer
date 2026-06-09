package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

type Request struct {
	ArchivePath string
	AppName     string
	Scope       Scope
}

type Result struct {
	AppName    string
	InstallDir string
	Command    string
	Desktop    string
	Launches   string
}

type appInfo struct {
	name       string
	slug       string
	executable string
	icon       string
}

func Install(ctx context.Context, req Request) (Result, error) {
	if req.Scope == "" {
		req.Scope = ScopeUser
	}
	if req.Scope != ScopeUser && req.Scope != ScopeSystem {
		return Result{}, fmt.Errorf("unknown install scope: %s", req.Scope)
	}

	archivePath, err := filepath.Abs(req.ArchivePath)
	if err != nil {
		return Result{}, err
	}
	if err := validateArchive(archivePath); err != nil {
		return Result{}, err
	}

	if req.Scope == ScopeSystem && os.Geteuid() != 0 {
		return runPrivileged(ctx, archivePath, req.AppName)
	}

	tempDir, err := os.MkdirTemp("", "targz-installer-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tempDir)

	if err := extractTarGz(archivePath, tempDir); err != nil {
		return Result{}, err
	}

	root, err := contentRoot(tempDir)
	if err != nil {
		return Result{}, err
	}

	info, err := inspectApp(root, archivePath, req.AppName)
	if err != nil {
		return Result{}, err
	}

	paths, err := installPaths(req.Scope, info.slug)
	if err != nil {
		return Result{}, err
	}

	if err := os.RemoveAll(paths.installDir); err != nil {
		return Result{}, err
	}
	if err := copyDir(root, paths.installDir); err != nil {
		return Result{}, err
	}

	if err := os.MkdirAll(filepath.Dir(paths.commandPath), 0755); err != nil {
		return Result{}, err
	}

	launches := "application"
	if info.executable != "" {
		relativeExec, err := filepath.Rel(root, info.executable)
		if err != nil {
			return Result{}, err
		}
		installedExec := filepath.Join(paths.installDir, relativeExec)
		if err := writeLauncher(paths.commandPath, installedExec); err != nil {
			return Result{}, err
		}
	} else {
		launches = "installed folder"
		if err := writeFolderLauncher(paths.commandPath, paths.installDir); err != nil {
			return Result{}, err
		}
	}

	if err := os.MkdirAll(filepath.Dir(paths.desktopPath), 0755); err != nil {
		return Result{}, err
	}

	iconName := "application-x-executable"
	if info.executable == "" {
		iconName = "folder"
	}
	if info.icon != "" {
		iconDest, err := installIcon(info.icon, paths.iconPath)
		if err != nil {
			return Result{}, err
		}
		iconName = iconDest
	}
	if err := writeDesktopFile(paths.desktopPath, info.name, paths.commandPath, iconName); err != nil {
		return Result{}, err
	}

	return Result{
		AppName:    info.name,
		InstallDir: paths.installDir,
		Command:    paths.commandPath,
		Desktop:    paths.desktopPath,
		Launches:   launches,
	}, nil
}

func validateArchive(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return errors.New("archive path is a directory")
	}
	name := strings.ToLower(filepath.Base(path))
	if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".tgz") {
		return errors.New("please choose a .tar.gz or .tgz file")
	}
	return nil
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if header.Name == "" {
			continue
		}

		target, err := safeJoin(dest, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, fileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := writeFileFromTar(target, reader, fileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			linkTarget := header.Linkname
			if filepath.IsAbs(linkTarget) || strings.Contains(filepath.Clean(linkTarget), "..") {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.Symlink(linkTarget, target); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
	}
	return nil
}

func safeJoin(base, name string) (string, error) {
	cleanName := filepath.Clean(strings.TrimPrefix(name, "/"))
	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return "", fmt.Errorf("archive contains unsafe path: %s", name)
	}
	target := filepath.Join(base, cleanName)
	cleanBase := filepath.Clean(base)
	if target != cleanBase && !strings.HasPrefix(target, cleanBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive contains unsafe path: %s", name)
	}
	return target, nil
}

func fileMode(mode int64) os.FileMode {
	return os.FileMode(mode) & 0777
}

func writeFileFromTar(path string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func contentRoot(tempDir string) (string, error) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return "", err
	}
	visible := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		visible = append(visible, entry)
	}
	if len(visible) == 1 && visible[0].IsDir() {
		return filepath.Join(tempDir, visible[0].Name()), nil
	}
	return tempDir, nil
}

func inspectApp(root, archivePath, requestedName string) (appInfo, error) {
	name := strings.TrimSpace(requestedName)
	if name == "" {
		name = archiveName(archivePath)
	}
	name = titleName(name)
	slug := slugify(name)
	if slug == "" {
		return appInfo{}, errors.New("could not determine an application name")
	}

	return appInfo{
		name:       name,
		slug:       slug,
		executable: findExecutable(root, slug),
		icon:       findIcon(root),
	}, nil
}

func archiveName(path string) string {
	name := filepath.Base(path)
	for _, suffix := range []string{".tar.gz", ".tgz"} {
		if strings.HasSuffix(strings.ToLower(name), suffix) {
			return name[:len(name)-len(suffix)]
		}
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func titleName(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 1 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		} else {
			parts[i] = strings.ToUpper(part)
		}
	}
	return strings.Join(parts, " ")
}

var slugChars = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugChars.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func findExecutable(root, slug string) string {
	preferred := []string{
		filepath.Join(root, "AppRun"),
		filepath.Join(root, slug),
		filepath.Join(root, "bin", slug),
	}
	for _, path := range preferred {
		if isExecutable(path) {
			return path
		}
	}

	candidates := []string{}
	for _, dir := range []string{filepath.Join(root, "bin"), root} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if !entry.IsDir() && isExecutable(path) {
				candidates = append(candidates, path)
			}
		}
	}
	sort.Strings(candidates)
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func isExecutable(path string) bool {
	stat, err := os.Stat(path)
	if err != nil || stat.IsDir() {
		return false
	}
	return stat.Mode()&0111 != 0
}

func findIcon(root string) string {
	extensions := map[string]bool{".png": true, ".svg": true, ".xpm": true}
	var icons []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		lower := strings.ToLower(entry.Name())
		if strings.Contains(lower, "icon") && extensions[filepath.Ext(lower)] {
			icons = append(icons, path)
		}
		return nil
	})
	sort.Strings(icons)
	if len(icons) == 0 {
		return ""
	}
	return icons[0]
}

type paths struct {
	installDir  string
	commandPath string
	desktopPath string
	iconPath    string
}

func installPaths(scope Scope, slug string) (paths, error) {
	if scope == ScopeSystem {
		return paths{
			installDir:  filepath.Join("/opt/targz-installer/apps", slug),
			commandPath: filepath.Join("/usr/local/bin", slug),
			desktopPath: filepath.Join("/usr/local/share/applications", slug+".desktop"),
			iconPath:    filepath.Join("/usr/local/share/icons/hicolor/scalable/apps", slug),
		}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return paths{}, err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return paths{
		installDir:  filepath.Join(dataHome, "targz-installer", "apps", slug),
		commandPath: filepath.Join(home, ".local", "bin", slug),
		desktopPath: filepath.Join(dataHome, "applications", slug+".desktop"),
		iconPath:    filepath.Join(dataHome, "icons", "hicolor", "scalable", "apps", slug),
	}, nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode()&0777)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return copyFile(path, target, info.Mode()&0777)
	})
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, source)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func writeLauncher(path, executable string) error {
	content := fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", executable)
	return os.WriteFile(path, []byte(content), 0755)
}

func writeFolderLauncher(path, installDir string) error {
	content := fmt.Sprintf("#!/bin/sh\nif command -v xdg-open >/dev/null 2>&1; then\n  exec xdg-open %q\nfi\nprintf 'Installed files: %%s\\n' %q\n", installDir, installDir)
	return os.WriteFile(path, []byte(content), 0755)
}

func installIcon(source, iconBase string) (string, error) {
	ext := strings.ToLower(filepath.Ext(source))
	if ext == "" {
		ext = ".png"
	}
	dest := iconBase + ext
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return "", err
	}
	if err := copyFile(source, dest, 0644); err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(dest), ext), nil
}

func writeDesktopFile(path, name, commandPath, icon string) error {
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Icon=%s
Terminal=false
Categories=Utility;
StartupNotify=true
`, desktopValue(name), desktopValue(commandPath), desktopValue(icon))
	return os.WriteFile(path, []byte(content), 0644)
}

func desktopValue(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

func runPrivileged(ctx context.Context, archivePath, appName string) (Result, error) {
	exe, err := os.Executable()
	if err != nil {
		return Result{}, err
	}
	args := []string{exe, "install", archivePath, "--system", "--yes"}
	if strings.TrimSpace(appName) != "" {
		args = append(args, "--name", appName)
	}

	if commandExists("pkexec") {
		cmd := exec.CommandContext(ctx, "pkexec", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return Result{}, err
		}
		displayName := strings.TrimSpace(appName)
		if displayName == "" {
			displayName = titleName(archiveName(archivePath))
		}
		return Result{AppName: displayName}, nil
	}

	return Result{}, errors.New("system install requires root privileges or pkexec")
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

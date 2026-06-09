package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallUserArchive(t *testing.T) {
	home := t.TempDir()
	dataHome := filepath.Join(home, ".local", "share")
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", dataHome)

	archive := filepath.Join(t.TempDir(), "sample-app.tar.gz")
	writeArchive(t, archive, []archiveEntry{
		{name: "sample-app/AppRun", mode: 0755, body: "#!/bin/sh\necho sample\n"},
		{name: "sample-app/icon.svg", mode: 0644, body: "<svg xmlns=\"http://www.w3.org/2000/svg\"/>"},
	})

	result, err := Install(context.Background(), Request{ArchivePath: archive, Scope: ScopeUser})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if result.AppName != "Sample App" {
		t.Fatalf("AppName = %q, want Sample App", result.AppName)
	}
	assertFileExists(t, filepath.Join(dataHome, "targz-installer", "apps", "sample-app", "AppRun"))
	assertFileExists(t, filepath.Join(home, ".local", "bin", "sample-app"))
	assertFileExists(t, filepath.Join(dataHome, "applications", "sample-app.desktop"))
	assertFileExists(t, filepath.Join(dataHome, "icons", "hicolor", "scalable", "apps", "sample-app.svg"))
}

func TestInstallArchiveWithoutExecutable(t *testing.T) {
	home := t.TempDir()
	dataHome := filepath.Join(home, ".local", "share")
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", dataHome)

	archive := filepath.Join(t.TempDir(), "docs-bundle.tar.gz")
	writeArchive(t, archive, []archiveEntry{
		{name: "docs-bundle/readme.txt", mode: 0644, body: "plain files\n"},
	})

	result, err := Install(context.Background(), Request{ArchivePath: archive, Scope: ScopeUser})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Launches != "installed folder" {
		t.Fatalf("Launches = %q, want installed folder", result.Launches)
	}

	commandPath := filepath.Join(home, ".local", "bin", "docs-bundle")
	assertFileExists(t, filepath.Join(dataHome, "targz-installer", "apps", "docs-bundle", "readme.txt"))
	assertFileExists(t, commandPath)

	content, err := os.ReadFile(commandPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "xdg-open") {
		t.Fatalf("fallback launcher does not use xdg-open:\n%s", string(content))
	}
}

func TestInstallRejectsUnsafeArchivePath(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "unsafe.tar.gz")
	writeArchive(t, archive, []archiveEntry{
		{name: "../escape", mode: 0644, body: "bad"},
	})

	_, err := Install(context.Background(), Request{ArchivePath: archive, Scope: ScopeUser})
	if err == nil {
		t.Fatal("Install() error = nil, want unsafe path error")
	}
}

type archiveEntry struct {
	name string
	mode int64
	body string
}

func writeArchive(t *testing.T, path string, entries []archiveEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.name,
			Mode: entry.mode,
			Size: int64(len(entry.body)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
}

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Chintanpatel24/.tar.gz-installer/internal/commands"
	"github.com/Chintanpatel24/.tar.gz-installer/internal/guiassets"
	"github.com/Chintanpatel24/.tar.gz-installer/internal/installer"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(commands.Run(os.Args[1:]))
	}

	gui := app.NewWithID("org.opensource.targz-installer")
	gui.SetIcon(guiassets.Logo())

	window := gui.NewWindow("Tar.gz Installer")
	window.Resize(fyne.NewSize(560, 360))
	window.SetFixedSize(false)

	var lock sync.Mutex
	var archivePath string

	title := widget.NewLabel("Tar.gz Installer")
	title.TextStyle = fyne.TextStyle{Bold: true}

	status := widget.NewLabel("Drop a .tar.gz file into this window.")
	status.Wrapping = fyne.TextWrapWord

	pathLabel := widget.NewLabel("No archive selected")
	pathLabel.Wrapping = fyne.TextWrapWord

	nameInput := widget.NewEntry()
	nameInput.SetPlaceHolder("Application name")

	scopeUser := widget.NewRadioGroup([]string{"User install", "System install"}, nil)
	scopeUser.Horizontal = true
	scopeUser.SetSelected("User install")

	chooseButton := widget.NewButtonWithIcon("Choose file", theme.FolderOpenIcon(), func() {
		picker := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				showError(window, err)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			setArchive(reader.URI().Path(), &lock, &archivePath, pathLabel, nameInput, status)
		}, window)
		picker.Show()
	})

	var installButton *widget.Button
	installButton = widget.NewButtonWithIcon("Install", theme.ConfirmIcon(), func() {
		lock.Lock()
		selected := archivePath
		lock.Unlock()

		if selected == "" {
			showError(window, errors.New("choose or drop a .tar.gz file first"))
			return
		}

		scope := installer.ScopeUser
		if scopeUser.Selected == "System install" {
			scope = installer.ScopeSystem
			status.SetText("System install may ask for your password.")
		} else {
			status.SetText("Installing...")
		}
		installButton.Disable()
		chooseButton.Disable()

		result, err := installer.Install(context.Background(), installer.Request{
			ArchivePath: selected,
			AppName:     nameInput.Text,
			Scope:       scope,
		})
		installButton.Enable()
		chooseButton.Enable()
		if err != nil {
			status.SetText("Install failed.")
			showError(window, err)
			return
		}
		if result.AppName == "" {
			status.SetText("Install finished.")
			dialog.ShowInformation("Installed", "The application was installed.", window)
			return
		}
		status.SetText("Installed " + result.AppName + ".")
		message := result.AppName + " is ready from your application menu."
		if result.Launches == "installed folder" {
			message = result.AppName + " was installed. Its menu entry opens the installed folder because no executable file was found."
		}
		dialog.ShowInformation("Installed", message, window)
	})
	installButton.Importance = widget.HighImportance

	dropBox := container.NewBorder(
		nil,
		nil,
		widget.NewIcon(theme.DownloadIcon()),
		nil,
		container.NewVBox(pathLabel, status),
	)

	window.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		if len(uris) == 0 {
			return
		}
		setArchive(uris[0].Path(), &lock, &archivePath, pathLabel, nameInput, status)
	})

	content := container.NewPadded(container.NewVBox(
		title,
		dropBox,
		nameInput,
		scopeUser,
		container.NewHBox(chooseButton, installButton),
	))

	window.SetContent(content)
	window.ShowAndRun()
}

func setArchive(path string, lock *sync.Mutex, archivePath *string, pathLabel *widget.Label, nameInput *widget.Entry, status *widget.Label) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	lock.Lock()
	*archivePath = path
	lock.Unlock()
	pathLabel.SetText(path)
	status.SetText("Ready to install.")
	if strings.TrimSpace(nameInput.Text) == "" {
		nameInput.SetText(defaultName(path))
	}
}

func defaultName(path string) string {
	name := filepath.Base(path)
	for _, suffix := range []string{".tar.gz", ".tgz"} {
		if strings.HasSuffix(strings.ToLower(name), suffix) {
			return name[:len(name)-len(suffix)]
		}
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func showError(parent fyne.Window, err error) {
	dialog.ShowError(err, parent)
}

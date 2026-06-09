package guiassets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed logo.svg
var logo []byte

func Logo() fyne.Resource {
	return fyne.NewStaticResource("targz-installer.svg", logo)
}

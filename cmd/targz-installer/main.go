package main

import (
	"os"
	"github.com/open-source-labs/targz-installer/internal/commands"
)
func main() {
	os.Exit(commands.Run(os.Args[1:]))
}

package main

import (
	"os"
	"github.com/Chintanpatel24/.tar.gz-installer/internal/commands"
)
func main() {
	os.Exit(commands.Run(os.Args[1:]))
}

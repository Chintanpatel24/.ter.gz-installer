package commands

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/open-source-labs/targz-installer/internal/installer"
)

func Run(args []string) int {
	if len(args) < 1 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		Usage()
		return 0
	}

	switch args[0] {
	case "install":
		return install(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		Usage()
		return 2
	}
}

func install(args []string) int {
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	system := flags.Bool("system", false, "install system-wide")
	user := flags.Bool("user", false, "install for the current user")
	name := flags.String("name", "", "application name")
	yes := flags.Bool("yes", false, "accept install prompts")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "install requires one .tar.gz path")
		return 2
	}
	if *system && *user {
		fmt.Fprintln(os.Stderr, "choose either --user or --system")
		return 2
	}
	if !*yes && *system && os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "system install may ask for your password")
	}

	scope := installer.ScopeUser
	if *system {
		scope = installer.ScopeSystem
	}

	result, err := installer.Install(context.Background(), installer.Request{
		ArchivePath: flags.Arg(0),
		AppName:     *name,
		Scope:       scope,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
		return 1
	}

	if result.AppName != "" {
		fmt.Printf("installed %s\n", result.AppName)
	}
	if result.InstallDir != "" {
		fmt.Printf("files: %s\n", result.InstallDir)
	}
	if result.Command != "" {
		fmt.Printf("command: %s\n", result.Command)
	}
	if result.Desktop != "" {
		fmt.Printf("desktop entry: %s\n", result.Desktop)
	}
	if result.Launches != "" {
		fmt.Printf("launcher opens: %s\n", result.Launches)
	}
	return 0
}

func Usage() {
	fmt.Print(`Tar.gz Installer

Usage:
  targz-installer install PATH [--user|--system] [--name NAME]

Examples:
  targz-installer install ~/Downloads/app.tar.gz
  targz-installer install ~/Downloads/app.tar.gz --system
`)
}

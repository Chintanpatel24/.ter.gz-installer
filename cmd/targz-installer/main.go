package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/open-source-labs/targz-installer/internal/installer"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		usage()
		return
	}

	switch os.Args[1] {
	case "install":
		install(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func install(args []string) {
	flags := flag.NewFlagSet("install", flag.ExitOnError)
	system := flags.Bool("system", false, "install system-wide")
	user := flags.Bool("user", false, "install for the current user")
	name := flags.String("name", "", "application name")
	yes := flags.Bool("yes", false, "accept install prompts")
	flags.Parse(args)

	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "install requires one .tar.gz path")
		os.Exit(2)
	}
	if *system && *user {
		fmt.Fprintln(os.Stderr, "choose either --user or --system")
		os.Exit(2)
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
		os.Exit(1)
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
}

func usage() {
	fmt.Print(`Tar.gz Installer

Usage:
  targz-installer install PATH [--user|--system] [--name NAME]

Examples:
  targz-installer install ~/Downloads/app.tar.gz
  targz-installer install ~/Downloads/app.tar.gz --system
`)
}

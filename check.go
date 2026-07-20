package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Taka-S-dev/baton/internal/config"
)

// runCheck implements `baton check [project|path]`: it loads projects
// without starting the TUI and prints their diagnostics. Returns the
// process exit code — 0 when everything is clean, 1 otherwise — so
// scripts, CI, and AI agents can use it as a validation loop.
func runCheck(args []string) int {
	dirs, err := checkTargets(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	exit := 0
	for _, dir := range dirs {
		name := filepath.Base(dir)
		p, err := config.LoadProject(dir)
		if err != nil {
			fmt.Printf("%s: ERROR %v\n", name, err)
			exit = 1
			continue
		}
		if len(p.Warnings) == 0 {
			fmt.Printf("%s: OK (%d commands, %d workflows, %d lists)\n",
				name, len(p.Config.AllCommands()), len(p.Workflows), len(p.Lists))
			continue
		}
		exit = 1
		for _, w := range p.Warnings {
			fmt.Printf("%s: WARN %s\n", name, w)
		}
	}
	return exit
}

// checkTargets resolves the check argument: no argument means every
// project in the projects directory; an existing directory path is
// checked as-is; anything else is a project name under the projects dir.
func checkTargets(args []string) ([]string, error) {
	if len(args) > 0 {
		arg := args[0]
		if info, err := os.Stat(arg); err == nil && info.IsDir() {
			return []string{arg}, nil
		}
		projectsDir, err := config.FindProjectsDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(projectsDir, arg)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("project %q not found (looked for directory %s)", arg, dir)
		}
		return []string{dir}, nil
	}

	projectsDir, err := config.FindProjectsDir()
	if err != nil {
		return nil, err
	}
	names := config.ListProjects(projectsDir)
	if len(names) == 0 {
		return nil, fmt.Errorf("no projects found in %s", projectsDir)
	}
	dirs := make([]string, len(names))
	for i, n := range names {
		dirs[i] = filepath.Join(projectsDir, n)
	}
	return dirs, nil
}

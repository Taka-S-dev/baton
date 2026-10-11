package main

import "fmt"

type launchOptions struct{ recent, dryRun, help, version bool }

func parseLaunchArgs(args []string) (launchOptions, error) {
	var o launchOptions
	for _, arg := range args {
		switch arg {
		case "recent", "r":
			if o.recent {
				return o, fmt.Errorf("unexpected argument %q", arg)
			}
			o.recent = true
		case "--dry-run":
			o.dryRun = true
		case "-h", "--help":
			o.help = true
		case "--version":
			o.version = true
		default:
			return o, fmt.Errorf("unknown argument %q (see baton --help)", arg)
		}
	}
	return o, nil
}

package main

import "testing"

func TestLaunchArgs(t *testing.T) {
	for _, args := range [][]string{{"recent"}, {"r"}, {"recent", "--dry-run"}, {"--dry-run", "r"}} {
		o, err := parseLaunchArgs(args)
		if err != nil || !o.recent || o.dryRun != (len(args) == 2) {
			t.Fatalf("%v: %+v %v", args, o, err)
		}
	}
	for _, args := range [][]string{{"--recent"}, {"recent", "oops"}, {"recent", "r"}, {"--dryrun"}} {
		if _, err := parseLaunchArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if o, err := parseLaunchArgs(nil); err != nil || o.recent {
		t.Fatal("normal startup changed")
	}
}

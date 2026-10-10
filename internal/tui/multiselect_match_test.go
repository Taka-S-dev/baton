package tui

import (
	mdl "github.com/Taka-S-dev/baton/internal/model"
	"strings"
	"testing"
)

func TestCommandMatchReason(t *testing.T) {
	c := &mdl.Command{Name: "mul", Cmd: "echo " + strings.Repeat("x", 80) + " build_directory", Values: map[string]string{"env": "production"}}
	reason := commandMatchReason(c, "mul", []string{"BU"}, 32)
	if !strings.Contains(reason, "cmd:") || !strings.Contains(reason, "bu") {
		t.Fatalf("missing distant command match: %s", reason)
	}
	reason = commandMatchReason(c, "mul", []string{"prod"}, 32)
	if !strings.Contains(reason, "fixed env:") || !strings.Contains(reason, "prod") {
		t.Fatal(reason)
	}
	if commandMatchReason(c, "build [build]", []string{"bu"}, 32) != "" {
		t.Fatal("visible match repeated")
	}
}
func TestMultiSelectSearchRenderingKeepsSelection(t *testing.T) {
	m := exploreProject()
	m.setupMultiSelect()
	m.msSearchTI.SetValue("build")
	m.msSelected = []int{0}
	before := append([]int{}, m.msSelected...)
	view := m.viewMultiSelect(60)
	if len(m.msSelected) != len(before) || m.msSelected[0] != before[0] {
		t.Fatal("render changed selection")
	}
	if !strings.Contains(view, "Selected(1)") {
		t.Fatal("selection count missing")
	}
}

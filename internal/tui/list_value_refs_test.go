package tui

import (
	"strings"
	"testing"

	mdl "github.com/Taka-S-dev/baton/internal/model"
)

func TestFixedValueReferencesRequireMatchingListAndValue(t *testing.T) {
	m := exploreProject()
	m.config.Base = append(m.config.Base,
		mdl.Command{Name: "mapped", Cmd: "run {target}", Slots: map[string]string{"target": "env"}},
		mdl.Command{Name: "other", Cmd: "run {target}", Slots: map[string]string{"target": "elsewhere"}})
	m.config.Commands = []mdl.Command{
		{Name: "saved", Template: "mapped", Values: map[string]string{"target": "prod"}},
		{Name: "different", Template: "mapped", Values: map[string]string{"target": "dev"}},
		{Name: "unrelated", Template: "other", Values: map[string]string{"target": "prod"}},
		{Name: "interactive", Template: "mapped"},
	}
	refs := m.listValueRefs("env", "prod")
	if len(refs) != 1 || !strings.Contains(refs[0], "saved") {
		t.Fatalf("wrong references: %v", refs)
	}
	m.vars["saved.target"] = "dev"
	if len(m.listValueRefs("env", "prod")) != 0 {
		t.Fatal("scoped override ignored")
	}
}

func TestVariadicFixedValueMatchesWholeTokens(t *testing.T) {
	m := exploreProject()
	m.config.Base = append(m.config.Base, mdl.Command{Name: "multi", Cmd: "run {env...}"})
	m.config.Commands = []mdl.Command{{Name: "saved", Template: "multi", Values: map[string]string{"env": "dev prod"}}}
	if len(m.listValueRefs("env", "prod")) != 1 {
		t.Fatal("variadic selection missing")
	}
	if len(m.listValueRefs("env", "pro")) != 0 {
		t.Fatal("substring counted as selection")
	}
}

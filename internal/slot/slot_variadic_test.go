package slot_test

import (
	"testing"

	"github.com/Taka-S-dev/baton/internal/model"
	"github.com/Taka-S-dev/baton/internal/slot"
)

// ── Variadic {name...} slots ─────────────────────────────────────────────────

func TestGetSlots_Variadic(t *testing.T) {
	cmd := model.Command{Cmd: "docker compose up {services...}"}
	slots := slot.GetSlots(cmd)
	if len(slots) != 1 {
		t.Fatalf("got %d slots, want 1", len(slots))
	}
	if slots[0].Name != "services" || !slots[0].Variadic {
		t.Errorf("got %+v, want Name=services Variadic=true", slots[0])
	}
	if slots[0].ListName != "services" {
		t.Errorf("ListName = %q, want default to slot name", slots[0].ListName)
	}
}

func TestGetSlots_MixedFormsResolveOnceAsVariadic(t *testing.T) {
	// The same name as {x} and {x...} must not prompt twice; the single
	// resolution accepts multiple values.
	cmd := model.Command{Cmd: "echo {x} {x...}"}
	slots := slot.GetSlots(cmd)
	if len(slots) != 1 {
		t.Fatalf("got %d slots, want 1", len(slots))
	}
	if !slots[0].Variadic {
		t.Error("mixed {x}/{x...} must resolve as variadic")
	}
}

func TestGetSlots_VariadicUsesSlotsMapping(t *testing.T) {
	cmd := model.Command{
		Cmd:   "up {svcs...}",
		Slots: map[string]string{"svcs": "services"},
	}
	slots := slot.GetSlots(cmd)
	if len(slots) != 1 || slots[0].ListName != "services" {
		t.Fatalf("got %+v, want ListName=services", slots)
	}
}

func TestHasPlaceholders_Variadic(t *testing.T) {
	if !slot.HasPlaceholders(model.Command{Cmd: "up {services...}"}) {
		t.Error("variadic placeholder must count as a placeholder")
	}
}

func TestApply_Variadic(t *testing.T) {
	cmd := model.Command{Cmd: "docker compose up {services...}", Dir: "{root...}"}
	got := slot.Apply(cmd, map[string]string{"services": "api web", "root": "/srv"})
	if got.Cmd != "docker compose up api web" {
		t.Errorf("Cmd = %q", got.Cmd)
	}
	if got.Dir != "/srv" {
		t.Errorf("Dir = %q", got.Dir)
	}
}

func TestReplace_BothForms(t *testing.T) {
	got := slot.Replace("run {x} and {x...}", "x", "v")
	if got != "run v and v" {
		t.Errorf("got %q", got)
	}
}

func TestPlaceholder_Text(t *testing.T) {
	if got := (slot.Def{Name: "env"}).Placeholder(); got != "{env}" {
		t.Errorf("got %q", got)
	}
	if got := (slot.Def{Name: "env", Variadic: true}).Placeholder(); got != "{env...}" {
		t.Errorf("got %q", got)
	}
}

func TestHighlightSlot_VariadicResolvedReplaced(t *testing.T) {
	out := slot.HighlightSlot("up {services...}", "", map[string]string{"services": "api web"})
	if out != "up api web" {
		t.Errorf("got %q", out)
	}
}

func TestVariadicInvisibleToVars(t *testing.T) {
	// {name...} must never be captured by a project variable of the same name.
	out := slot.ApplyVars("up {services...}", map[string]string{"services": "X"})
	if out != "up {services...}" {
		t.Errorf("got %q — vars must not touch interactive slots", out)
	}
}

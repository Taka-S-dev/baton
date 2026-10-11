package tui

import (
	"fmt"

	"github.com/Taka-S-dev/baton/internal/slot"
)

// Completion highlights are suggestions, not accepted input.
func (m Model) pathInputReady() bool {
	s := m.sp
	return s != nil && !s.previewOnly && s.completion == nil && (len(s.picked) > 0 || (s.completedPath != "" && s.completedPath == s.search))
}

func (m Model) inputProgress() (done, total int) {
	if m.resolve == nil {
		return 0, 0
	}
	r := m.resolve
	for i, item := range r.rawItems {
		n := len(slot.GetSlots(*item.cmd))
		total += n
		if i < r.currentIdx {
			done += n
		} else if i == r.currentIdx {
			done += r.currentSlotIdx
			if m.pathInputReady() {
				done++
			}
		}
	}
	return
}

func (m Model) inputProgressLabel() string {
	if m.sp == nil || m.sp.pathDef.PathKind == "" || m.sp.previewOnly {
		return ""
	}
	done, total := m.inputProgress()
	if total == 0 {
		return ""
	}
	if done == total {
		return "✓ Inputs complete"
	}
	return fmt.Sprintf("Inputs %d/%d complete", done, total)
}

func (m Model) pathInputKeys() string {
	enter := "choose"
	if m.pathInputReady() {
		enter = "confirm"
		if done, total := m.inputProgress(); total > 0 {
			if done == total {
				enter = "review run"
			} else {
				enter = "next input"
			}
		}
	}
	keys := "Tab: choices  Enter: " + enter + "  Esc: back"
	if m.sp.canSkip {
		keys += "  Ctrl+K: keep interactive"
	}
	return keys
}

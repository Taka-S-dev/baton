package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
)

func TestCommandFormBlinkAcrossLayouts(t *testing.T) {
	for _, width := range []int{50, 140} {
		m := exploreProject()
		m.width = width
		// The default 530ms blink would be waited out once per field; the
		// test only needs the tick to arrive.
		m.nameInput.Cursor.BlinkSpeed = time.Millisecond
		focus := m.openExploreCommandForm("")
		for _, field := range []int{1, 0, 2, 3} {
			if field != 1 {
				focus = m.cfGotoField(field)
			}
			if focus == nil || !m.nameInput.Focused() || m.nameInput.Cursor.Mode() != cursor.CursorBlink {
				t.Fatal("field must focus blinking caret")
			}
			msg := focus()
			if _, ok := msg.(cursor.BlinkMsg); !ok {
				t.Fatalf("unexpected focus event %T", msg)
			}
			next, tick := m.Update(msg)
			m = next.(Model)
			if tick == nil {
				t.Fatalf("width %d field %d lost blink tick", width, field)
			}
			if m.nameInput.Cursor.Blink != true {
				t.Fatal("caret did not toggle")
			}
		}
		m.closeCommandForm()
		if m.nameInput.Focused() {
			t.Fatal("closed editor retains caret")
		}
	}
}

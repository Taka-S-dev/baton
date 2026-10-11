package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPickerSelectedRowFillsTheBox(t *testing.T) {
	for _, tc := range []struct {
		text  string
		width int
	}{
		{" ▶ api", 30},
		{" ▶ 日本語.go", 30},
		{" ▶ a label far longer than the box it has to fit inside", 20},
		{" ▶ x", 2},
	} {
		got := ansi.StringWidth(pickerSelectedRow(tc.text, tc.width))
		if want := max(0, tc.width-2); got != want {
			t.Errorf("%q in a box %d wide: row is %d cells, want %d", tc.text, tc.width, got, want)
		}
	}
}

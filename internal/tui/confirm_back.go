package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	mdl "github.com/Taka-S-dev/baton/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) rememberConfirmBack() {
	if m.resolve == nil || m.sp == nil {
		return
	}
	back := *m
	back.confirmBack = nil
	r := *m.resolve
	r.currentValues = copyMap(r.currentValues)
	r.itemNotes = append([]string(nil), r.itemNotes...)
	r.resolved = append([]mdl.RunItem(nil), r.resolved...)
	r.answered = make(map[string]answeredSlot, len(m.resolve.answered))
	for k, v := range m.resolve.answered {
		r.answered[k] = v
	}
	s := *m.sp
	s.picked = append([]string(nil), s.picked...)
	s.completion = nil
	back.resolve = &r
	back.sp = &s
	m.confirmBack = &back
}

func (m Model) backFromConfirm() (tea.Model, tea.Cmd) {
	if m.confirmBack != nil {
		back := *m.confirmBack
		back.width, back.height = m.width, m.height
		back.confirmBack = nil
		back.screen = ScreenSlotPick
		return back, nil
	}
	m.gotoRunHome()
	return m, nil
}

func (m Model) confirmBackLabel() string {
	if m.confirmBack != nil {
		return "Back"
	}
	return "Cancel"
}

func (m Model) confirmWorkdir(dir string) string {
	base := m.launchDir
	if base == "" {
		base, _ = os.Getwd()
	}
	if dir == "" {
		return base
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(filepath.ToSlash(dir), "/") {
		return filepath.VolumeName(base) + filepath.Clean(dir)
	}
	return filepath.Join(base, dir)
}

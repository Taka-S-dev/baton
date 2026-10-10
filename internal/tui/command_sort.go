package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var commandSortModes = []string{"name", "group", "type", "definition", "run"}

func validCommandSort(value string) bool {
	for _, mode := range commandSortModes {
		if strings.TrimSuffix(value, "-desc") == mode {
			return true
		}
	}
	return false
}

func (m *Model) loadCommandSort() {
	m.commandSort = "name"
	if m.projectDir == "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(m.projectDir, ".command_sort"))
	if err == nil && validCommandSort(strings.TrimSpace(string(data))) {
		m.commandSort = strings.TrimSpace(string(data))
	}
}

func (m Model) sortCommandIndices(indices []int) {
	if m.commandSort == "" || m.commandSort == "definition" {
		return
	}
	mode := strings.TrimSuffix(m.commandSort, "-desc")
	descending := strings.HasSuffix(m.commandSort, "-desc")
	ranks := map[int]int{}
	for i, index := range m.msSelected {
		ranks[index] = i + 1
	}
	sort.SliceStable(indices, func(i, j int) bool {
		if mode == "run" {
			a, b := ranks[indices[i]], ranks[indices[j]]
			if a == 0 && b != 0 {
				return false
			}
			if b == 0 && a != 0 {
				return true
			}
			if a != b {
				if descending {
					return a > b
				}
				return a < b
			}
		}
		a, b := m.msItems[indices[i]].cmd, m.msItems[indices[j]].cmd
		key := func(name, group, kind string) string {
			switch mode {
			case "group":
				return strings.ToLower(group)
			case "type":
				return strings.ToLower(kind)
			}
			return strings.ToLower(name)
		}
		ka, kb := key(a.Name, a.Group, commandKind(*a)), key(b.Name, b.Group, commandKind(*b))
		if ka != kb {
			if descending {
				return ka > kb
			}
			return ka < kb
		}
		if descending && mode != "run" {
			return strings.ToLower(a.Name) > strings.ToLower(b.Name)
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}

func (m *Model) cycleCommandSort() {
	next := 0
	for i, mode := range commandSortModes {
		if mode == strings.TrimSuffix(m.commandSort, "-desc") {
			next = (i + 1) % len(commandSortModes)
			break
		}
	}
	m.setCommandSort(commandSortModes[next])
}

func (m *Model) setCommandSort(mode string) {
	before := m.msFiltered()
	selected := -1
	if m.msCursor >= 0 && m.msCursor < len(before) {
		selected = before[m.msCursor]
	}
	m.commandSort = mode
	m.msCursor = 0
	for i, index := range m.msFiltered() {
		if index == selected {
			m.msCursor = i
			break
		}
	}
	if m.projectDir == "" {
		return
	}
	file, err := os.CreateTemp(m.projectDir, ".command-sort-*")
	if err == nil {
		name := file.Name()
		defer os.Remove(name)
		_, err = file.WriteString(m.commandSort + "\n")
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(name, filepath.Join(m.projectDir, ".command_sort"))
		}
	}
	if err != nil {
		m.errMsg = "Sort changed, but preference could not be saved: " + err.Error()
	}
}

func (m Model) commandSortLabel() string {
	mode := strings.TrimSuffix(m.commandSort, "-desc")
	if mode == "" || mode == "definition" {
		return "Definition"
	}
	return m.sortHeading(mode, strings.ToUpper(mode[:1])+mode[1:])
}

func (m Model) sortHeading(mode, label string) string {
	if strings.TrimSuffix(m.commandSort, "-desc") != mode {
		return label
	}
	if strings.HasSuffix(m.commandSort, "-desc") {
		return label + " ↓"
	}
	return label + " ↑"
}

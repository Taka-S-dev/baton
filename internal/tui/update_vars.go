package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taka-S-dev/baton/internal/slot"
)

// varNamePattern is what {$name} references can express: \w+ only.
var varNamePattern = regexp.MustCompile(`^\w+$`)

// ── Manage vars ───────────────────────────────────────────────────────────────

// gotoVarsMgmt opens the Manage vars submenu.
func (m *Model) gotoVarsMgmt() {
	m.screen = ScreenManageVars
	m.listItems = []string{"Create variable", "Edit variable", "Delete variable"}
	m.listCursor = 0
}

// globalVarNames returns the global ("*" row) variable names, sorted.
// Scoped "command.slot" values belong to their commands and are managed
// through Edit command → Change values, not here.
func (m *Model) globalVarNames() []string {
	var names []string
	for k := range m.vars {
		if !strings.Contains(k, ".") {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return names
}

// varRefLocations lists where {$name} is referenced, for previews and
// delete-impact display.
func (m *Model) varRefLocations(name string) []string {
	ref := "{$" + name + "}"
	var out []string
	for _, c := range m.config.AllCommands() {
		if strings.Contains(c.Cmd, ref) || strings.Contains(c.Dir, ref) {
			out = append(out, "command \""+c.Name+"\"")
		}
	}
	for _, ln := range m.sortedListNames() {
		n := 0
		for _, e := range m.lists[ln] {
			if strings.Contains(e.Value, ref) {
				n++
			}
		}
		if n > 0 {
			out = append(out, fmt.Sprintf("list \"%s\" (%d entries)", ln, n))
		}
	}
	for _, k := range sortedKeys(m.vars) {
		if strings.Contains(k, ".") && strings.Contains(m.vars[k], ref) {
			out = append(out, "saved value "+k)
		}
	}
	return out
}

// setVarPickBase fills the pick filter with "name = value" rows.
func (m *Model) setVarPickBase() {
	names := m.globalVarNames()
	labels := make([]string, len(names))
	texts := make([]string, len(names))
	for i, n := range names {
		labels[i] = n + " = " + m.vars[n]
		if refs := len(m.varRefLocations(n)); refs > 0 {
			labels[i] += fmt.Sprintf("  (%d refs)", refs)
		}
		texts[i] = n + " " + m.vars[n]
	}
	m.setPickBase(labels, texts)
	// pickBase holds display labels; keep the raw names alongside.
	m.varPickNames = names
}

func (m Model) updateVarsMgmt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Create variable":
			m.ve = &varEditState{mode: 0}
			m.nameInput.SetValue("")
			m.screen = ScreenVarForm
			return m, m.nameInput.Focus()
		case "Edit variable":
			m.screen = ScreenEditVarPick
			m.setVarPickBase()
			m.listCursor = 0
		case "Delete variable":
			m.screen = ScreenDeleteVar
			m.setVarPickBase()
			m.listCursor = 0
			m.deleteSelected = nil
		}
	case "esc":
		m.gotoMainMenu()
	}
	return m, nil
}

// ── Edit variable: pick which one ─────────────────────────────────────────────

func (m Model) updateEditVarPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		if len(m.listItems) == 0 {
			break
		}
		name := m.varPickNames[m.pickOrig(m.listCursor)]
		m.ve = &varEditState{mode: 1, name: name, oldValue: m.vars[name]}
		m.nameInput.SetValue(m.vars[name])
		m.screen = ScreenVarForm
		return m, m.nameInput.Focus()
	case "esc":
		if m.pickSearch != "" {
			m.clearPickFilter()
			break
		}
		m.gotoVarsMgmt()
	default:
		m.handlePickTyping(msg, nil)
	}
	return m, nil
}

// ── Create / edit form ────────────────────────────────────────────────────────

func (m Model) updateVarForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ve := m.ve
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.nameInput.Value())
		if ve.mode == 0 && ve.fieldIdx == 0 {
			if !varNamePattern.MatchString(val) {
				m.errMsg = "variable names are letters, digits and _ only"
				return m, nil
			}
			if _, exists := m.vars[val]; exists {
				m.errMsg = "name already in use: " + val
				return m, nil
			}
			ve.name = val
			ve.fieldIdx = 1
			m.nameInput.SetValue("")
			return m, nil
		}
		if val == "" {
			m.errMsg = "value must not be empty"
			return m, nil
		}
		return m.saveVar(val)
	case "esc":
		if ve.mode == 0 && ve.fieldIdx == 1 {
			ve.fieldIdx = 0
			m.nameInput.SetValue(ve.name)
			return m, nil
		}
		m.ve = nil
		if ve.mode == 1 {
			m.screen = ScreenEditVarPick
			m.setVarPickBase()
			m.listCursor = 0
			return m, nil
		}
		m.gotoVarsMgmt()
		return m, nil
	}
	ti, cmd := m.nameInput.Update(msg)
	m.nameInput = ti
	return m, cmd
}

func (m Model) saveVar(value string) (tea.Model, tea.Cmd) {
	ve := m.ve
	if m.vars == nil {
		m.vars = make(map[string]string)
	}
	m.vars[ve.name] = value
	if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
		m.errMsg = "failed to save vars.tsv: " + err.Error()
		return m, nil
	}
	m.ve = nil

	// A changed value may leave literals behind: offer to rebase values
	// that start with the OLD value onto the {$name} reference.
	if ve.mode == 1 && ve.oldValue != "" && ve.oldValue != value {
		if vr := m.buildVarRebase(ve.name, ve.oldValue); vr != nil {
			m.vr = vr
			m.screen = ScreenVarRebase
			return m, nil
		}
	}

	if ve.mode == 0 {
		m.successMsg = "created variable {$" + ve.name + "}"
	} else {
		m.successMsg = "updated variable {$" + ve.name + "}"
	}
	m.gotoVarsMgmt()
	return m, nil
}

// ── Delete variable ───────────────────────────────────────────────────────────

func (m Model) updateDeleteVars(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := m.varPickNames // onDelete receives original indices
	return m.updateDeleteList(msg, len(m.listItems), nil,
		m.gotoVarsMgmt,
		func(indices []int) {
			for _, i := range indices {
				delete(m.vars, names[i])
			}
			if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
				m.errMsg = "failed to save vars.tsv: " + err.Error()
				return
			}
			if len(indices) == 1 {
				m.successMsg = "deleted variable {$" + names[indices[0]] + "}"
			} else {
				m.successMsg = fmt.Sprintf("deleted %d variables", len(indices))
			}
		})
}

// ── Rebase offer ──────────────────────────────────────────────────────────────

// buildVarRebase scans for literal values that start with the variable's
// old value — scoped vars.tsv values and list entries, the files baton
// manages. The match is PREFIX-anchored, never substring: values that
// merely contain the old value elsewhere are not candidates at all.
// Prefix matches whose next character is not a path separator are shown
// but default to off (the boundary is suspicious).
func (m *Model) buildVarRebase(name, oldVal string) *varRebaseState {
	ref := "{$" + name + "}"
	cleanBoundary := func(v string) bool {
		if len(v) == len(oldVal) {
			return true
		}
		c := v[len(oldVal)]
		return c == '\\' || c == '/'
	}
	var items []varRebaseItem
	for _, k := range sortedKeys(m.vars) {
		v := m.vars[k]
		if strings.Contains(k, ".") && strings.HasPrefix(v, oldVal) {
			items = append(items, varRebaseItem{
				kind: 0, key: k, label: "saved value " + k,
				oldValue: v, newValue: ref + v[len(oldVal):], on: cleanBoundary(v),
			})
		}
	}
	for _, ln := range m.sortedListNames() {
		for i, e := range m.lists[ln] {
			if strings.HasPrefix(e.Value, oldVal) {
				items = append(items, varRebaseItem{
					kind: 1, listName: ln, entryIdx: i, label: "list " + ln,
					oldValue: e.Value, newValue: ref + e.Value[len(oldVal):], on: cleanBoundary(e.Value),
				})
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	return &varRebaseState{varName: name, items: items}
}

func (m Model) updateVarRebase(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vr := m.vr
	switch msg.String() {
	case "up":
		if vr.cursor > 0 {
			vr.cursor--
		}
	case "down":
		if vr.cursor < len(vr.items)-1 {
			vr.cursor++
		}
	case "tab":
		vr.items[vr.cursor].on = !vr.items[vr.cursor].on
	case "enter":
		return m.applyVarRebase()
	case "esc":
		m.vr = nil
		m.successMsg = "updated variable {$" + vr.varName + "} (literals left as-is)"
		m.gotoVarsMgmt()
	}
	return m, nil
}

func (m Model) applyVarRebase() (tea.Model, tea.Cmd) {
	vr := m.vr
	applied := 0
	varsChanged := false
	changedLists := make(map[string]bool)
	for _, it := range vr.items {
		if !it.on {
			continue
		}
		switch it.kind {
		case 0:
			m.vars[it.key] = it.newValue
			varsChanged = true
		case 1:
			m.lists[it.listName][it.entryIdx].Value = it.newValue
			changedLists[it.listName] = true
		}
		applied++
	}
	if varsChanged {
		if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
			m.errMsg = "failed to save vars.tsv: " + err.Error()
		}
	}
	listsDir := filepath.Join(m.projectDir, "lists")
	for ln := range changedLists {
		if err := slot.SaveList(listsDir, ln, m.lists[ln]); err != nil {
			m.errMsg = "failed to save list \"" + ln + "\": " + err.Error()
		}
	}
	name := vr.varName
	m.vr = nil
	if applied == 0 {
		m.successMsg = "updated variable {$" + name + "} (literals left as-is)"
	} else {
		m.successMsg = fmt.Sprintf("updated {$%s} and rebased %d value(s) onto it", name, applied)
	}
	m.gotoVarsMgmt()
	return m, nil
}

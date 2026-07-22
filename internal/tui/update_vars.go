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
	m.listItems = []string{"Create variable (global)", "Edit variable", "Delete variable"}
	m.listCursor = 0
}

// scopedKey splits a "command.slot" vars key at its LAST dot (command
// names may contain dots, slot names cannot). ok is false for globals.
func scopedKey(key string) (cmdName, slotName string, ok bool) {
	i := strings.LastIndex(key, ".")
	if i <= 0 {
		return "", "", false
	}
	return key[:i], key[i+1:], true
}

// syncScopedVar mirrors a scoped vars.tsv change into the in-memory
// command it belongs to and re-bakes it, so previews and runs pick the
// change up without a reload. remove un-fixes the slot (it will be
// prompted at run time again).
func (m *Model) syncScopedVar(key, value string, remove bool) {
	cmdName, slotName, ok := scopedKey(key)
	if !ok {
		return
	}
	for ci := range m.config.Commands {
		c := &m.config.Commands[ci]
		if c.Name != cmdName {
			continue
		}
		if remove {
			delete(c.Values, slotName)
		} else {
			if c.Values == nil {
				c.Values = make(map[string]string)
			}
			c.Values[slotName] = value
		}
		if baked, err := slot.MaterializeCommand(*c, m.config); err == nil {
			*c = baked
		}
		return
	}
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

// setVarPickBase fills the pick filter with every vars.tsv row: globals
// first ({$name} = value), then saved commands' fixed values
// (command.slot = value). The whole file is visible and editable here.
func (m *Model) setVarPickBase() {
	var globals, scoped []string
	for k := range m.vars {
		if strings.Contains(k, ".") {
			scoped = append(scoped, k)
		} else {
			globals = append(globals, k)
		}
	}
	sort.Strings(globals)
	sort.Strings(scoped)
	names := append(globals, scoped...)

	labels := make([]string, len(names))
	texts := make([]string, len(names))
	for i, n := range names {
		if strings.Contains(n, ".") {
			labels[i] = n + " = " + m.vars[n]
		} else {
			labels[i] = "{$" + n + "} = " + m.vars[n]
			if refs := len(m.varRefLocations(n)); refs > 0 {
				labels[i] += fmt.Sprintf("  (%d refs)", refs)
			}
		}
		texts[i] = n + " " + m.vars[n]
	}
	m.setPickBase(labels, texts)
	// pickBase holds display labels; keep the raw keys alongside.
	m.varPickNames = names
}

func (m Model) updateVarsMgmt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "down":
		m.moveListCursor(msg.String(), len(m.listItems))
	case "enter":
		switch m.listItems[m.listCursor] {
		case "Create variable (global)":
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

	if _, _, scoped := scopedKey(ve.name); scoped {
		// A saved command's fixed value: mirror it into the command and
		// re-bake. No rebase offer — that is for globals.
		m.syncScopedVar(ve.name, value, false)
		m.successMsg = "updated fixed value " + ve.name
		m.gotoVarsMgmt()
		return m, nil
	}

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
				// Deleting a fixed value un-fixes the slot: prompted at
				// run time again.
				m.syncScopedVar(names[i], "", true)
			}
			if err := slot.SaveVars(m.projectDir, m.vars); err != nil {
				m.errMsg = "failed to save vars.tsv: " + err.Error()
				return
			}
			if len(indices) == 1 {
				name := names[indices[0]]
				if _, _, scoped := scopedKey(name); scoped {
					m.successMsg = "deleted fixed value " + name + " (prompted at run time again)"
				} else {
					m.successMsg = "deleted variable {$" + name + "}"
				}
			} else {
				m.successMsg = fmt.Sprintf("deleted %d entries", len(indices))
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
			m.syncScopedVar(it.key, it.newValue, false)
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

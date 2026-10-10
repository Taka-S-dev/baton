package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// gotoExplore opens the overview with the cursors where they were left,
// so stepping out to run something and coming back does not lose the
// place on any tab.
func (m *Model) gotoExplore() {
	m.screen = ScreenExplore
	for tab := range m.exploreCursor {
		if n := len(m.exploreItems(tab)); m.exploreCursor[tab] >= n {
			m.exploreCursor[tab] = max(0, n-1)
		}
	}
}

func (m Model) updateExplore(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.exploreVisible()
	cur := &m.exploreCursor[m.exploreTab]
	if *cur >= len(items) {
		*cur = max(0, len(items)-1)
	}

	// The search takes the letters while it is open. The arrows still
	// move the cursor through what matches, so narrowing the list and
	// picking from it is one motion.
	if m.exploreSearching {
		switch msg.String() {
		case "esc":
			m.exploreSearch, m.exploreSearching = "", false
		case "enter":
			m.exploreSearching = false
		case "backspace":
			if r := []rune(m.exploreSearch); len(r) > 0 {
				m.exploreSearch = string(r[:len(r)-1])
			}
		case "up":
			*cur = max(0, *cur-1)
		case "down":
			*cur = min(max(0, len(m.exploreVisible())-1), *cur+1)
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.exploreSearch += string(msg.Runes)
				if msg.Type == tea.KeySpace {
					m.exploreSearch += " "
				}
				*cur = 0
			}
		}
		return m, nil
	}

	// The action menu takes the keys while it is open: the arrows and
	// Enter pick from it, an action's own key runs it, anything else
	// closes it. The keys also work with the menu closed; the menu is
	// how they are learnt.
	if m.exploreMenu {
		it, ok := m.exploreCurrent()
		acts := exploreActionsFor(it.kind)
		if !ok || len(acts) == 0 {
			m.exploreMenu = false
			return m, nil
		}
		switch key := msg.String(); key {
		case "up":
			m.exploreMenuCursor = max(0, m.exploreMenuCursor-1)
		case "down":
			m.exploreMenuCursor = min(len(acts)-1, m.exploreMenuCursor+1)
		case "enter":
			return m, m.runExploreAction(acts[m.exploreMenuCursor].key, it)
		case "esc", " ":
			m.exploreMenu = false
		default:
			for _, a := range acts {
				if a.key == key {
					return m, m.runExploreAction(a.key, it)
				}
			}
			m.exploreMenu = false
		}
		return m, nil
	}

	switch msg.String() {
	case "up":
		*cur = max(0, *cur-1)
	case "down":
		*cur = min(max(0, len(items)-1), *cur+1)
	case " ":
		if it, ok := m.exploreCurrent(); ok && len(exploreActionsFor(it.kind)) > 0 {
			m.exploreMenu, m.exploreMenuCursor = true, 0
		}
	case "d", "r", "x":
		if it, ok := m.exploreCurrent(); ok {
			for _, a := range exploreActionsFor(it.kind) {
				if a.key == msg.String() {
					return m, m.runExploreAction(a.key, it)
				}
			}
		}
	case "1", "2", "3", "4":
		m.switchTab(int(msg.String()[0] - '1'))
	case "left":
		m.switchTab((m.exploreTab + tabCount - 1) % tabCount)
	case "right":
		m.switchTab((m.exploreTab + 1) % tabCount)
	case "/":
		m.exploreSearching = true
	case "enter", "e":
		// Enter is "open" on every kind, and e the same key by name:
		// a command edits in the box, the rest go to their editors.
		if it, ok := m.exploreCurrent(); ok {
			return m, m.runExploreAction("e", it)
		}
	case "n":
		if m.exploreTab == tabCommands {
			return m, m.openExploreCommandForm("")
		}
	case "[":
		m.exploreSplit = m.exploreLayout(m.viewWidth()).leftW - 2
	case "]":
		m.exploreSplit = m.exploreLayout(m.viewWidth()).leftW + 2
	case "esc":
		if m.exploreSearch != "" {
			m.exploreSearch = ""
			break
		}
		m.gotoMainMenu()
	}
	return m, nil
}

// switchTab changes which kind of thing is listed. The search belongs
// to the tab it was typed over, so it is dropped on the way.
func (m *Model) switchTab(tab int) {
	if tab < 0 || tab >= tabCount || tab == m.exploreTab {
		return
	}
	m.exploreTab = tab
	m.exploreSearch, m.exploreSearching = "", false
}

// openExploreCommandForm starts a command edit. On a terminal with room
// for the right side the form takes the editor box and the list stays;
// on one without, the form has the screen to itself, which is what the
// manage menu always gave it, and comes back here when it closes.
func (m *Model) openExploreCommandForm(name string) tea.Cmd {
	cf := &commandFormState{mode: 0, editIdx: -1, returnTo: ScreenExplore}
	if name != "" {
		names, refs := m.editableCommands()
		for i, n := range names {
			if n != name {
				continue
			}
			cmd := m.editRefCommand(refs[i])
			cf = &commandFormState{
				mode: 1, returnTo: ScreenExplore, editIdx: refs[i].idx,
				tsvEdit: refs[i].tsv, origName: cmd.Name,
				fields: [5]string{cmd.Name, cmd.Cmd, cmd.Dir, cmd.Group, cmd.Shell},
			}
			break
		}
		if cf.mode == 0 {
			m.errMsg = "only TSV rows and TUI-created commands can be edited"
			return nil
		}
	}
	cf.original = cf.fields
	m.cf = cf
	if m.exploreLayout(m.viewWidth()).single {
		m.screen = ScreenCommandForm
	}
	// The cmd comes first: the name is the one piece of real thought in
	// the form, and asking for it before anything else is what made the
	// first keystroke the hardest one.
	return m.cfGotoField(1)
}

// wheelRows is how far one notch of the wheel moves the cursor. One row
// per notch reads as stuck; a page overshoots what you were aiming at.
const wheelRows = 3

// updateExploreMouse points the overview with the pointer. A click on a
// tab opens it, a click on a row selects it and a second click on the
// same row edits it, a click on a field puts the caret in that field,
// and the wheel moves the cursor. Everything it needs to know about
// where things are drawn comes from exploreLayout, the same description
// the view draws from.
func (m Model) updateExploreMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.leaveAsk != leaveNone {
		return m, nil
	}
	l := m.exploreLayout(m.viewWidth())
	cur := &m.exploreCursor[m.exploreTab]

	// A drag is only a drag while the button is still down. Anything
	// else ends it rather than being swallowed: a release that never
	// arrives would otherwise leave the grab latched, and every click
	// after it would move the divider instead of the cursor.
	if m.exploreDragging {
		if msg.Action == tea.MouseActionMotion && msg.Button == tea.MouseButtonLeft {
			m.exploreSplit = msg.X
			return m, nil
		}
		m.exploreDragging = false
		if msg.Action == tea.MouseActionRelease {
			return m, nil
		}
	}

	// With a workflow open in the box, the wheel and clicks over the
	// box go to its steps; a click on the list is a click away from
	// it, leaving the way Esc does and selecting what was clicked.
	if m.wfe != nil && !m.wfe.adding {
		e := m.wfe
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			e.cursor = max(0, e.cursor-1)
			return m, nil
		case tea.MouseButtonWheelDown:
			e.cursor = min(max(0, len(e.steps)-1), e.cursor+1)
			return m, nil
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if !l.single && msg.X >= l.rightX {
				if i := msg.Y - l.fieldY; i >= 0 && i < len(e.steps) {
					e.cursor = i
				}
				return m, nil
			}
			if msg.X < l.leftW && !m.askToLeave(leaveWorkflow) {
				m.wfe = nil
			}
		}
	}

	if m.cf == nil && m.wfe == nil {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			*cur = max(0, l.cursor-wheelRows)
			return m, nil
		case tea.MouseButtonWheelDown:
			*cur = min(max(0, len(l.items)-1), l.cursor+wheelRows)
			return m, nil
		}
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	// The column between the boxes is the handle for the split.
	if !l.single && msg.X == l.leftW {
		m.exploreDragging = true
		m.exploreSplit = msg.X
		return m, nil
	}

	// With the menu open, a click on one of its rows runs that action
	// and a click anywhere else closes it.
	if m.exploreMenu {
		m.exploreMenu = false
		if it, ok := m.exploreCurrent(); ok && !l.single && msg.X >= l.rightX {
			if i := msg.Y - l.fieldY; i >= 0 && i < len(exploreActionsFor(it.kind)) {
				return m, m.runExploreAction(exploreActionsFor(it.kind)[i].key, it)
			}
		}
		return m, nil
	}

	if msg.Y == l.tabY && m.cf == nil {
		for i, span := range l.tabX {
			if msg.X >= span[0] && msg.X < span[1] {
				m.switchTab(i)
				return m, nil
			}
		}
		return m, nil
	}

	// The fields of the editor box, whether or not the caret is in them:
	// a click on one means the same thing in both states.
	if !l.single && msg.X >= l.rightX && m.exploreTab == tabCommands {
		field := msg.Y - l.fieldY
		if field < 0 || field >= len(formOrder) {
			return m, nil
		}
		if m.cf != nil {
			m.cf.fields[m.cf.fieldIdx] = strings.TrimSpace(m.nameInput.Value())
			return m, m.cfGotoField(formFieldAt(field))
		}
		if it, ok := m.exploreCurrent(); ok {
			cmd := m.openExploreCommandForm(it.name)
			if m.cf == nil {
				return m, cmd
			}
			return m, m.cfGotoField(formFieldAt(field))
		}
		return m, nil
	}
	if msg.X >= l.leftW && !l.single {
		return m, nil
	}

	// The search line sits above the rows inside the list box.
	if !l.plain && msg.Y == l.listY-1 && m.cf == nil {
		m.exploreSearching = true
		return m, nil
	}

	row := msg.Y - l.listY
	if row < 0 || row >= l.listH {
		return m, nil
	}
	i := l.start + row
	if i < 0 || i >= len(l.items) {
		return m, nil
	}
	if m.cf != nil {
		// A click on the list is a click away from the form: it leaves,
		// exactly as Esc does, and goes on to select what was clicked —
		// unless there is something unsaved, which is asked about first.
		if m.askToLeave(leaveForm) {
			return m, nil
		}
		m.closeCommandForm()
		l = m.exploreLayout(m.viewWidth())
		i = l.start + row
		if i >= len(l.items) {
			return m, nil
		}
	}
	// A second click on the row already selected opens it, so picking
	// something and editing it is one gesture twice.
	if i == *cur {
		if it := l.items[i]; it.kind == exCommand {
			return m, m.openExploreCommandForm(it.name)
		}
	}
	*cur = i
	return m, nil
}

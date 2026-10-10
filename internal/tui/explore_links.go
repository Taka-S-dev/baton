package tui

import (
	"fmt"
	"github.com/Taka-S-dev/baton/internal/slot"
	"strings"
)

type exploreLocation struct {
	tab      int
	cursor   [tabCount]int
	search   string
	workflow *workflowEditState
}
type exploreLink struct {
	tab         int
	name, label string
}

func (m Model) commandLinks(name string) []exploreLink {
	c, ok := m.config.FindCommand(name)
	if !ok {
		return nil
	}
	var links []exploreLink
	seen := map[string]bool{}
	for _, s := range slot.GetSlots(c) {
		if _, ok := m.lists[s.ListName]; ok && !seen[s.ListName] {
			links = append(links, exploreLink{tabLists, s.ListName, "Open list · " + s.ListName})
			seen[s.ListName] = true
		}
	}
	for _, key := range sortedKeys(m.vars) {
		if !strings.Contains(key, ".") && strings.Contains(c.Cmd+" "+c.Dir, "{$"+key+"}") {
			links = append(links, exploreLink{tabVars, key, "Open variable · " + key})
		}
	}
	return links
}
func (m *Model) openExploreLink(tab int, name string) {
	m.exploreHistory = append(m.exploreHistory, exploreLocation{m.exploreTab, m.exploreCursor, m.exploreSearch, m.wfe})
	m.wfe = nil
	m.exploreTab = tab
	m.exploreSearch = ""
	m.exploreSearching = false
	for i, it := range m.exploreItems(tab) {
		if it.name == name {
			m.exploreCursor[tab] = i
			break
		}
	}
}
func (m *Model) backExploreLink() bool {
	n := len(m.exploreHistory)
	if n == 0 {
		return false
	}
	p := m.exploreHistory[n-1]
	m.exploreHistory = m.exploreHistory[:n-1]
	m.exploreTab, m.exploreCursor, m.exploreSearch, m.wfe = p.tab, p.cursor, p.search, p.workflow
	return true
}
func linkKey(i int) string { return fmt.Sprintf("go%d", i+1) }

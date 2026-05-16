package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

const sidebarWidth = 22

type sidebarSection int

const (
	sectionAll         sidebarSection = iota
	sectionTypesHeader                // non-selectable divider
	sectionByType                     // one per EntryType
	sectionTagsHeader                 // non-selectable divider
	sectionByTag                      // one per unique tag
)

type sidebarItem struct {
	section sidebarSection
	label   string
	filter  string // EntryType string or tag value
}

// Sidebar manages the left-hand navigation panel.
type Sidebar struct {
	items    []sidebarItem
	cursor   int
	focused  bool
}

func NewSidebar() Sidebar {
	return Sidebar{}
}

// Rebuild rebuilds the sidebar items from the current set of entries.
func (s *Sidebar) Rebuild(entries []data.Entry) {
	items := []sidebarItem{
		{section: sectionAll, label: "  All"},
		{section: sectionTypesHeader, label: "  Types"},
	}
	for _, t := range data.AllEntryTypes {
		items = append(items, sidebarItem{
			section: sectionByType,
			label:   "  " + data.EntryTypeLabel(t),
			filter:  string(t),
		})
	}

	// Collect unique tags
	tagSet := map[string]struct{}{}
	for _, e := range entries {
		for _, tag := range e.Tags {
			tagSet[tag] = struct{}{}
		}
	}
	if len(tagSet) > 0 {
		items = append(items, sidebarItem{section: sectionTagsHeader, label: "  Tags"})
		tags := make([]string, 0, len(tagSet))
		for t := range tagSet {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		for _, t := range tags {
			items = append(items, sidebarItem{section: sectionByTag, label: "  #" + t, filter: t})
		}
	}

	// Preserve cursor position if possible
	if s.cursor >= len(items) {
		s.cursor = 0
	}
	s.items = items
}

func (s *Sidebar) MoveUp() {
	for {
		if s.cursor > 0 {
			s.cursor--
		}
		if s.items[s.cursor].section != sectionTagsHeader &&
			s.items[s.cursor].section != sectionTypesHeader {
			break
		}
	}
}

func (s *Sidebar) MoveDown() {
	for {
		if s.cursor < len(s.items)-1 {
			s.cursor++
		}
		if s.items[s.cursor].section != sectionTagsHeader &&
			s.items[s.cursor].section != sectionTypesHeader {
			break
		}
	}
}

// ActiveFilter returns the current type filter (empty = all) and tag filter.
func (s *Sidebar) ActiveFilter() (typeFilter string, tagFilter string) {
	if s.cursor >= len(s.items) {
		return "", ""
	}
	item := s.items[s.cursor]
	switch item.section {
	case sectionAll:
		return "", ""
	case sectionByType:
		return item.filter, ""
	case sectionByTag:
		return "", item.filter
	}
	return "", ""
}

func (s *Sidebar) View(height int) string {
	style := SidebarStyle
	if s.focused {
		style = SidebarFocusBorderStyle
	}

	var sb strings.Builder
	for i, item := range s.items {
		var line string
		switch item.section {
		case sectionTagsHeader, sectionTypesHeader:
			line = SidebarSectionStyle.Render(item.label)
		default:
			if i == s.cursor {
				line = SidebarActiveStyle.Render(fmt.Sprintf("▸ %s", strings.TrimLeft(item.label, " ")))
			} else {
				line = SidebarItemStyle.Render(item.label)
			}
		}
		sb.WriteString(line + "\n")
	}

	inner := strings.TrimRight(sb.String(), "\n")
	return style.Height(height).Render(inner)
}

func (s *Sidebar) SetFocused(f bool) { s.focused = f }

// CountByType returns a map of entry counts per type string, used for optional display.
func countByType(entries []data.Entry) map[string]int {
	m := map[string]int{}
	for _, e := range entries {
		m[string(e.Type)]++
	}
	return m
}

// RenderTagBadges renders a comma-separated row of tags for the detail view.
func RenderTagBadges(tags []string) string {
	if len(tags) == 0 {
		return DetailMaskedStyle.Render("—")
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorText).
			Padding(0, 1).
			Render("#" + t)
	}
	return strings.Join(parts, " ")
}

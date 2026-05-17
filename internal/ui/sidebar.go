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
	sectionFavorites                  // ★ Favorites filter
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
	items        []sidebarItem
	cursor       int
	scrollOffset int
	focused      bool
	height       int
}

func NewSidebar() Sidebar {
	return Sidebar{}
}

// Rebuild rebuilds the sidebar items from the current set of entries.
// showCounts controls whether entry counts are shown next to each label.
func (s *Sidebar) Rebuild(entries []data.Entry, showCounts bool) {
	// Count per type and per tag
	typeCounts := map[string]int{}
	tagCounts := map[string]int{}
	for _, e := range entries {
		typeCounts[string(e.Type)]++
		for _, tag := range e.Tags {
			tagCounts[tag]++
		}
	}

	allLabel := "  All"
	if showCounts {
		allLabel = fmt.Sprintf("  All [%d]", len(entries))
	}
	favCount := 0
	for _, e := range entries {
		if e.Favorite {
			favCount++
		}
	}
	favLabel := "  ★ Favorites"
	if showCounts {
		favLabel = fmt.Sprintf("  ★ Favorites [%d]", favCount)
	}
	items := []sidebarItem{
		{section: sectionAll, label: allLabel},
		{section: sectionFavorites, label: favLabel},
		{section: sectionTypesHeader, label: "  Types"},
	}
	for _, t := range data.AllEntryTypes {
		label := "  " + data.EntryTypeLabel(t)
		if showCounts {
			label = fmt.Sprintf("  %s [%d]", data.EntryTypeLabel(t), typeCounts[string(t)])
		}
		items = append(items, sidebarItem{
			section: sectionByType,
			label:   label,
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
			label := "  #" + t
			if showCounts {
				label = fmt.Sprintf("  #%s [%d]", t, tagCounts[t])
			}
			items = append(items, sidebarItem{
				section: sectionByTag,
				label:   label,
				filter:  t,
			})
		}
	}

	// Preserve cursor position if possible
	if s.cursor >= len(items) {
		s.cursor = 0
	}
	s.items = items
}

func (s *Sidebar) visibleRows() int {
	v := s.height - 4 // 2 border + 2 padding
	if v < 1 {
		v = 1
	}
	return v
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
	if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
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
	visible := s.visibleRows()
	if s.cursor >= s.scrollOffset+visible {
		s.scrollOffset = s.cursor - visible + 1
	}
}

// ActiveFilter returns the current type filter, tag filter, and favorites-only flag.
func (s *Sidebar) ActiveFilter() (typeFilter string, tagFilter string, favoritesOnly bool) {
	if s.cursor >= len(s.items) {
		return "", "", false
	}
	item := s.items[s.cursor]
	switch item.section {
	case sectionAll:
		return "", "", false
	case sectionFavorites:
		return "", "", true
	case sectionByType:
		return item.filter, "", false
	case sectionByTag:
		return "", item.filter, false
	}
	return "", "", false
}

func (s *Sidebar) View(height int) string {
	if s.height > 0 {
		height = s.height
	}
	style := SidebarStyle
	if s.focused {
		style = SidebarFocusBorderStyle
	}

	total := len(s.items)
	visible := s.visibleRows()

	// Clamp scrollOffset defensively
	scrollOffset := s.scrollOffset
	if scrollOffset > total-visible {
		scrollOffset = total - visible
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + visible
	if end > total {
		end = total
	}

	var sb strings.Builder

	// ▲ scroll-up indicator
	if scrollOffset > 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render("  ▲ scroll") + "\n")
	}

	for i, item := range s.items[scrollOffset:end] {
		absIdx := i + scrollOffset
		var line string
		switch item.section {
		case sectionTagsHeader, sectionTypesHeader:
			line = SidebarSectionStyle.Render(item.label)
		default:
			if absIdx == s.cursor {
				line = SidebarActiveStyle.Render(fmt.Sprintf("▸ %s", strings.TrimLeft(item.label, " ")))
			} else {
				line = SidebarItemStyle.Render(item.label)
			}
		}
		sb.WriteString(line + "\n")
	}

	// ▼ scroll-down indicator
	if end < total {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render("  ▼ more") + "\n")
	}

	inner := strings.TrimRight(sb.String(), "\n")
	return style.Height(height).Render(inner)
}

func (s *Sidebar) SetFocused(f bool) { s.focused = f }
func (s *Sidebar) SetHeight(h int)   { s.height = h }

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

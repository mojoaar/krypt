package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

// ListView renders the table of vault entries.
type ListView struct {
	entries  []data.Entry // filtered+sorted subset
	cursor   int
	focused  bool
	width    int
	height   int
}

func NewListView() ListView { return ListView{} }

func (l *ListView) SetEntries(entries []data.Entry) {
	// Preserve cursor
	var selectedID string
	if l.cursor < len(l.entries) {
		selectedID = l.entries[l.cursor].ID
	}
	l.entries = entries
	if selectedID != "" {
		for i, e := range l.entries {
			if e.ID == selectedID {
				l.cursor = i
				return
			}
		}
	}
	if l.cursor >= len(l.entries) {
		l.cursor = max(0, len(l.entries)-1)
	}
}

func (l *ListView) SetSize(w, h int) { l.width = w; l.height = h }
func (l *ListView) SetFocused(f bool) { l.focused = f }
func (l *ListView) MoveUp()   { if l.cursor > 0 { l.cursor-- } }
func (l *ListView) MoveDown() { if l.cursor < len(l.entries)-1 { l.cursor++ } }

// Selected returns the currently highlighted entry, or nil if list is empty.
func (l *ListView) Selected() *data.Entry {
	if len(l.entries) == 0 {
		return nil
	}
	e := l.entries[l.cursor]
	return &e
}

func (l ListView) View() string {
	if l.width == 0 {
		return ""
	}

	// Choose border style based on focused state
	baseStyle := ListStyle
	_ = l.focused // focus shown via ▸ cursor indicator only

	// "▸ " prefix on cursor row; account for 2 visual chars
	const cursorPfx = "▸ "
	const noCursorPfx = "  "
	usable := l.width - 4 // Padding(1,2) → 4 chars horizontal

	// Column widths: BADGE(7 visual) + cursor prefix(2) + NAME + DETAIL + UPDATED
	badgeContentW := 5
	badgeVisualW := badgeContentW + 2 // +2 for Padding(0,1)
	updatedW := 12
	nameW := (usable - badgeVisualW - updatedW - 6 - 2) * 35 / 100
	detailW := usable - badgeVisualW - nameW - updatedW - 6 - 2

	// Header row (no cursor prefix on header)
	typeHeader := HeaderStyle.Render(pad("TYPE", badgeVisualW))
	header := noCursorPfx + renderListRow(
		typeHeader,
		HeaderStyle.Render(pad("NAME", nameW)),
		HeaderStyle.Render(pad("DETAIL", detailW)),
		HeaderStyle.Render("UPDATED"),
	)

	if len(l.entries) == 0 {
		empty := lipgloss.NewStyle().Foreground(colorMuted).Render("no entries — press 'a' to add one")
		return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, empty))
	}

	rows := make([]string, len(l.entries))
	for i, e := range l.entries {
		badge := BadgeStyle(string(e.Type)).Render(data.EntryTypeBadge(e.Type))
		name := truncate(e.Name, nameW)
		detail := truncate(e.DetailLine(), detailW)
		updated := e.UpdatedAt.Format("Jan 02 2006")
		if e.UpdatedAt.IsZero() {
			updated = "—"
		}

		pfx := noCursorPfx
		if i == l.cursor {
			pfx = cursorPfx
		}

		row := pfx + renderListRow(
			badge,
			pad(name, nameW),
			pad(detail, detailW),
			updated,
		)

		// ANSI-safe width fill for the selection highlight
		vw := lipgloss.Width(row)
		if vw < usable {
			row = row + strings.Repeat(" ", usable-vw)
		}

		if i == l.cursor && l.focused {
			rows[i] = ListSelectedStyle.Render(row)
		} else if i == l.cursor {
			rows[i] = ListSelectedStyle.UnsetBackground().Render(row)
		} else {
			rows[i] = ListItemStyle.Render(row)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, rows...)...)
	return baseStyle.Render(content)
}

func renderListRow(badge, name, detail, updated string) string {
	return fmt.Sprintf("%-7s %-*s  %-*s  %s",
		badge,
		0, name,
		0, detail,
		updated,
	)
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FilterAndSort returns entries filtered by type/tag/search and sorted by name.
func FilterAndSort(entries []data.Entry, typeFilter, tagFilter, search string) []data.Entry {
	search = strings.ToLower(strings.TrimSpace(search))
	out := make([]data.Entry, 0, len(entries))
	for _, e := range entries {
		if typeFilter != "" && string(e.Type) != typeFilter {
			continue
		}
		if tagFilter != "" && !e.HasTag(tagFilter) {
			continue
		}
		if search != "" {
			nameMatch := strings.Contains(strings.ToLower(e.Name), search)
			detailMatch := strings.Contains(strings.ToLower(e.DetailLine()), search)
			tagMatch := false
			for _, t := range e.Tags {
				if strings.Contains(strings.ToLower(t), search) {
					tagMatch = true
					break
				}
			}
			if !nameMatch && !detailMatch && !tagMatch {
				continue
			}
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// age returns a short human-readable age string.
func age(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return t.Format("Jan 02")
	}
}

var _ = age // suppress unused warning; used optionally

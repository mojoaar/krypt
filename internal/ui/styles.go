package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Palette
	colorPrimary  = lipgloss.Color("#7C3AED") // violet
	colorAccent   = lipgloss.Color("#A78BFA") // light violet
	colorMuted    = lipgloss.Color("#6B7280") // gray
	colorText     = lipgloss.Color("#F9FAFB") // near-white
	colorSubtle   = lipgloss.Color("#D1D5DB") // light gray
	colorBorder   = lipgloss.Color("#374151") // dark gray border
	colorSelected = lipgloss.Color("#2D1B69") // dark violet bg (contrast-safe)
	colorDanger   = lipgloss.Color("#EF4444") // red for delete
	colorSuccess  = lipgloss.Color("#10B981") // green for success
	colorWarning  = lipgloss.Color("#F59E0B") // amber for warnings

	// Entry type badge colours
	colorBadgeLogin    = lipgloss.Color("#7C3AED") // violet  (matches primary)
	colorBadgeNote     = lipgloss.Color("#A78BFA") // light violet
	colorBadgeCard     = lipgloss.Color("#5B21B6") // deep violet
	colorBadgeIdentity = lipgloss.Color("#9333EA") // purple
	colorBadgeSSH      = lipgloss.Color("#C4B5FD") // pale violet

	// App chrome
	TitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	StatusOkStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Padding(0, 1)

	StatusErrStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Padding(0, 1)

	// Sidebar
	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), false, true, false, false).
			BorderForeground(colorBorder).
			Padding(1, 1).
			Width(20)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(colorSubtle).
				Padding(0, 1)

	SidebarActiveStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Padding(0, 1)

	SidebarFocusBorderStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder(), false, true, false, false).
					BorderForeground(colorPrimary).
					Padding(1, 1).
					Width(20)

	SidebarSectionStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true).
				Padding(0, 1)

	// List
	ListStyle = lipgloss.NewStyle().
			Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			Foreground(colorText)

	ListSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorAccent).
				Bold(true)

	// Column headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Bold(true)

	// Entry type badges
	BadgeLoginStyle = lipgloss.NewStyle().
			Background(colorBadgeLogin).
			Foreground(colorText).
			Bold(true).
			Width(5).
			Align(lipgloss.Center).
			Padding(0, 1)

	BadgeNoteStyle = lipgloss.NewStyle().
			Background(colorBadgeNote).
			Foreground(colorText).
			Bold(true).
			Width(5).
			Align(lipgloss.Center).
			Padding(0, 1)

	BadgeCardStyle = lipgloss.NewStyle().
			Background(colorBadgeCard).
			Foreground(colorText).
			Bold(true).
			Width(5).
			Align(lipgloss.Center).
			Padding(0, 1)

	BadgeIdentityStyle = lipgloss.NewStyle().
				Background(colorBadgeIdentity).
				Foreground(colorText).
				Bold(true).
				Width(5).
				Align(lipgloss.Center).
				Padding(0, 1)

	BadgeSSHStyle = lipgloss.NewStyle().
			Background(colorBadgeSSH).
			Foreground(lipgloss.Color("#1E1B4B")). // dark on pale violet for contrast
			Bold(true).
			Width(5).
			Align(lipgloss.Center).
			Padding(0, 1)

	// Form
	FormLabelStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	FormActiveInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	FormInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	FormTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginBottom(1)

	// Confirm dialog
	ConfirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDanger).
			Padding(1, 3).
			Align(lipgloss.Center)

	ConfirmTitleStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true).
				MarginBottom(1)

	ConfirmYesStyle = lipgloss.NewStyle().
			Background(colorDanger).
			Foreground(colorText).
			Bold(true).
			Padding(0, 2)

	ConfirmNoStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 2)

	// Help bar
	HelpKeyStyle  = lipgloss.NewStyle().Foreground(colorAccent)
	HelpDescStyle = lipgloss.NewStyle().Foreground(colorMuted)
	HelpSepStyle  = lipgloss.NewStyle().Foreground(colorBorder)

	// Search
	SearchPromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	SearchStyle       = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// Unlock screen
	UnlockBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 3).
			Align(lipgloss.Center)

	UnlockLabelStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				MarginBottom(1)

	UnlockErrorStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)

	UnlockHintStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Detail view
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true).
				Width(14)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	DetailMaskedStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				MarginBottom(1)

	DetailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)
)

// BadgeStyle returns the correct badge style for an entry type.
func BadgeStyle(t string) lipgloss.Style {
	switch t {
	case "login":
		return BadgeLoginStyle
	case "note":
		return BadgeNoteStyle
	case "card":
		return BadgeCardStyle
	case "identity":
		return BadgeIdentityStyle
	case "sshkey":
		return BadgeSSHStyle
	}
	return BadgeLoginStyle
}

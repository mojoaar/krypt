package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmYesMsg is emitted when the user confirms.
type ConfirmYesMsg struct{ ID string }

// ConfirmNoMsg is emitted when the user cancels.
type ConfirmNoMsg struct{}

type confirmChoice int

const (
	confirmNo  confirmChoice = iota
	confirmYes
)

// Confirm is the delete-confirmation dialog.
type Confirm struct {
	choice  confirmChoice
	entryID string
	name    string
	width   int
	height  int
}

func NewConfirm() Confirm { return Confirm{} }

func (c *Confirm) Open(id, name string) {
	c.entryID = id
	c.name = name
	c.choice = confirmNo
}

func (c *Confirm) SetSize(w, h int) { c.width = w; c.height = h }

func (c Confirm) Update(msg tea.Msg) (Confirm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			c.choice = confirmNo
		case "right", "l":
			c.choice = confirmYes
		case "tab":
			if c.choice == confirmNo {
				c.choice = confirmYes
			} else {
				c.choice = confirmNo
			}
		case "enter":
			if c.choice == confirmYes {
				id := c.entryID
				return c, func() tea.Msg { return ConfirmYesMsg{ID: id} }
			}
			return c, func() tea.Msg { return ConfirmNoMsg{} }
		case "esc", "n", "q":
			return c, func() tea.Msg { return ConfirmNoMsg{} }
		case "y":
			id := c.entryID
			return c, func() tea.Msg { return ConfirmYesMsg{ID: id} }
		}
	}
	return c, nil
}

func (c Confirm) View() string {
	title := ConfirmTitleStyle.Render("Delete entry?")
	msg := DetailValueStyle.Render(`"` + c.name + `" will be permanently removed.`)

	var yesBtn, noBtn string
	if c.choice == confirmYes {
		yesBtn = ConfirmYesStyle.Render("  Yes, delete  ")
		noBtn = ConfirmNoStyle.Render("  No  ")
	} else {
		yesBtn = lipgloss.NewStyle().Foreground(colorDanger).Padding(0, 2).Render("Yes, delete")
		noBtn = ConfirmYesStyle.
			Background(colorPrimary).
			Render("  No  ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, noBtn, "   ", yesBtn)
	inner := lipgloss.JoinVertical(lipgloss.Center, title, msg, "", buttons)
	hint := HelpDescStyle.Render("←/→ or tab to switch  •  enter to confirm  •  esc to cancel")
	box := ConfirmBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Center, inner, "", hint))

	if c.width > 0 && c.height > 0 {
		return lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

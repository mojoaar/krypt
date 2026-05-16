package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

type twoFAStep int

const (
	twoFAStepShow   twoFAStep = iota // display generated secret, wait for user to add to app
	twoFAStepVerify                  // enter 6-digit code to confirm setup
	twoFAStepDisable                 // confirm disabling
)

// TwoFADoneMsg signals the parent that 2FA setup/disable completed.
type TwoFADoneMsg struct {
	Enabled bool
	Err     error
}

// TwoFAModel handles the in-app 2FA setup / disable flow.
type TwoFAModel struct {
	step           twoFAStep
	secret         string
	codeInput      textinput.Model
	masterPassword string
	has2FA         bool
	errMsg         string
	width          int
	height         int
}

func NewTwoFAModel(masterPassword string, has2FA bool) (TwoFAModel, error) {
	ci := textinput.New()
	ci.Placeholder = "123456"
	ci.CharLimit = 6

	m := TwoFAModel{
		masterPassword: masterPassword,
		has2FA:         has2FA,
		codeInput:      ci,
	}

	if has2FA {
		m.step = twoFAStepDisable
	} else {
		secret, err := data.GenerateTOTPSecret()
		if err != nil {
			return m, err
		}
		m.secret = secret
		m.step = twoFAStepShow
	}
	return m, nil
}

func (m TwoFAModel) Init() tea.Cmd { return textinput.Blink }

func (m TwoFAModel) Update(msg tea.Msg) (TwoFAModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.errMsg = ""
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return TwoFADoneMsg{Enabled: m.has2FA} }

		case "enter":
			switch m.step {
			case twoFAStepShow:
				// Move to verification step
				m.step = twoFAStepVerify
				m.codeInput.Focus()
				return m, textinput.Blink

			case twoFAStepVerify:
				code := strings.TrimSpace(m.codeInput.Value())
				if len(code) != 6 {
					m.errMsg = "enter the 6-digit code from your authenticator app"
					return m, nil
				}
				// Verify the code matches the secret
				expected, _, err := data.GenerateTOTP(m.secret)
				if err != nil || code != expected {
					m.errMsg = "code does not match — check your authenticator and try again"
					m.codeInput.SetValue("")
					return m, nil
				}
				// Save the secret
				if err := data.SaveTwoFASecret(m.secret, m.masterPassword); err != nil {
					return m, func() tea.Msg { return TwoFADoneMsg{Err: err} }
				}
				return m, func() tea.Msg { return TwoFADoneMsg{Enabled: true} }

			case twoFAStepDisable:
				if err := data.DisableTwoFA(); err != nil {
					return m, func() tea.Msg { return TwoFADoneMsg{Err: err} }
				}
				return m, func() tea.Msg { return TwoFADoneMsg{Enabled: false} }
			}
		}
	}

	if m.step == twoFAStepVerify {
		var cmd tea.Cmd
		m.codeInput, cmd = m.codeInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m TwoFAModel) View() string {
	var inner string
	switch m.step {
	case twoFAStepShow:
		inner = m.viewShow()
	case twoFAStepVerify:
		inner = m.viewVerify()
	case twoFAStepDisable:
		inner = m.viewDisable()
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Render(inner)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m TwoFAModel) viewShow() string {
	// Format secret in groups of 4 for readability: ABCD EFGH IJKL …
	grouped := formatSecret(m.secret)

	lines := []string{
		FormTitleStyle.Render("Enable Two-Factor Authentication"),
		"",
		HelpDescStyle.Render("A secret key has been generated for your vault."),
		HelpDescStyle.Render("Add it to your authenticator app before continuing."),
		"",
		FormLabelStyle.Render("Your TOTP secret:"),
		"",
		lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 2).
			Render(grouped),
		"",
		HelpDescStyle.Render("Enter this key manually in:"),
		HelpKeyStyle.Render("  Google Authenticator") + HelpDescStyle.Render("  (+ → Enter a setup key)"),
		HelpKeyStyle.Render("  Authy") + HelpDescStyle.Render("             (Add Account → Enter key manually)"),
		HelpKeyStyle.Render("  1Password") + HelpDescStyle.Render("         (new item → One-Time Password field)"),
		"",
		HelpDescStyle.Render("Use account name: ") + HelpKeyStyle.Render("krypt"),
		"",
		UnlockHintStyle.Render("Once added, press enter to verify with a 6-digit code."),
		UnlockHintStyle.Render("Press esc to cancel."),
	}

	if m.errMsg != "" {
		lines = append(lines, "", UnlockErrorStyle.Render("✗ "+m.errMsg))
	}
	return strings.Join(lines, "\n")
}

func (m TwoFAModel) viewVerify() string {
	lines := []string{
		FormTitleStyle.Render("Verify 2FA Code"),
		"",
		HelpDescStyle.Render("Enter the 6-digit code from your authenticator app"),
		HelpDescStyle.Render("to confirm the secret was added correctly."),
		"",
		FormLabelStyle.Render("6-digit code"),
		FormActiveInputStyle.Width(20).Render(m.codeInput.View()),
		"",
		UnlockHintStyle.Render("Press enter to save  •  esc to go back"),
	}

	if m.errMsg != "" {
		lines = append(lines, "", UnlockErrorStyle.Render("✗ "+m.errMsg))
	}
	return strings.Join(lines, "\n")
}

func (m TwoFAModel) viewDisable() string {
	lines := []string{
		ConfirmTitleStyle.Render("Disable Two-Factor Authentication?"),
		"",
		HelpDescStyle.Render("This will remove your 2FA secret. The vault will"),
		HelpDescStyle.Render("only require the master password to unlock."),
		"",
		UnlockHintStyle.Render("Press enter to disable 2FA  •  esc to cancel"),
	}

	if m.errMsg != "" {
		lines = append(lines, "", UnlockErrorStyle.Render("✗ "+m.errMsg))
	}
	return strings.Join(lines, "\n")
}

// formatSecret splits a base32 secret into space-separated groups of 4.
func formatSecret(secret string) string {
	var groups []string
	for i := 0; i < len(secret); i += 4 {
		end := i + 4
		if end > len(secret) {
			end = len(secret)
		}
		groups = append(groups, secret[i:end])
	}
	return fmt.Sprintf("  %s  ", strings.Join(groups, " "))
}

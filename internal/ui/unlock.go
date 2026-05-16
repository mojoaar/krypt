package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// unlockStep tracks which step of the unlock flow we're on.
type unlockStep int

const (
	unlockStepPassword unlockStep = iota
	unlockStepTOTP
	unlockStepSetup2FA // shown after first vault creation to offer 2FA setup
	unlockStepReset    // danger zone: type "delete" to wipe all config files
)

// UnlockDoneMsg is emitted when the user successfully authenticates.
type UnlockDoneMsg struct {
	MasterPassword string
	IsNewVault     bool
}

// Unlock2FASetupMsg is emitted when the user opts to set up 2FA.
type Unlock2FASetupMsg struct {
	MasterPassword string
}

// UnlockResetMsg is emitted when the user confirms a full vault reset.
type UnlockResetMsg struct{}

// UnlockModel handles the master-password (+ optional TOTP) unlock screen.
type UnlockModel struct {
	step        unlockStep
	password    textinput.Model
	totp        textinput.Model
	resetInput  textinput.Model
	twoFASecret string // base32 secret shown during setup
	has2FA      bool
	isNewVault  bool
	errMsg      string
	width       int
	height      int
}

func NewUnlockModel(has2FA, isNewVault bool) UnlockModel {
	pw := textinput.New()
	pw.Placeholder = "master password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.CharLimit = 256
	pw.Focus()

	tp := textinput.New()
	tp.Placeholder = "123456"
	tp.CharLimit = 6

	ri := textinput.New()
	ri.Placeholder = `type "delete" to confirm`
	ri.CharLimit = 10

	return UnlockModel{
		step:       unlockStepPassword,
		password:   pw,
		totp:       tp,
		resetInput: ri,
		has2FA:     has2FA,
		isNewVault: isNewVault,
	}
}

func (m UnlockModel) Init() tea.Cmd { return textinput.Blink }

func (m UnlockModel) Update(msg tea.Msg) (UnlockModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r":
			if m.step == unlockStepPassword {
				m.step = unlockStepReset
				m.errMsg = ""
				m.password.Blur()
				m.resetInput.SetValue("")
				m.resetInput.Focus()
				return m, textinput.Blink
			}
		case "enter":
			m.errMsg = "" // clear error only on submit
			switch m.step {
			case unlockStepPassword:
				if strings.TrimSpace(m.password.Value()) == "" {
					m.errMsg = "password cannot be empty"
					return m, nil
				}
				if m.has2FA {
					m.step = unlockStepTOTP
					m.password.Blur()
					m.totp.Focus()
					return m, textinput.Blink
				}
				return m, func() tea.Msg {
					return UnlockDoneMsg{MasterPassword: m.password.Value(), IsNewVault: m.isNewVault}
				}
			case unlockStepTOTP:
				code := strings.TrimSpace(m.totp.Value())
				if len(code) != 6 {
					m.errMsg = "enter the 6-digit code from your authenticator"
					return m, nil
				}
				return m, func() tea.Msg {
					return unlockVerify2FAMsg{password: m.password.Value(), code: code, isNewVault: m.isNewVault}
				}
			case unlockStepReset:
				if strings.TrimSpace(m.resetInput.Value()) != "delete" {
					m.errMsg = `type the word "delete" to confirm`
					return m, nil
				}
				return m, func() tea.Msg { return UnlockResetMsg{} }
			}
		case "esc":
			if m.step == unlockStepTOTP {
				m.errMsg = ""
				m.step = unlockStepPassword
				m.totp.Blur()
				m.totp.SetValue("")
				m.password.Focus()
				return m, textinput.Blink
			}
			if m.step == unlockStepReset {
				m.errMsg = ""
				m.step = unlockStepPassword
				m.resetInput.Blur()
				m.resetInput.SetValue("")
				m.password.Focus()
				return m, textinput.Blink
			}
		}
	}

	var cmd tea.Cmd
	switch m.step {
	case unlockStepPassword:
		m.password, cmd = m.password.Update(msg)
	case unlockStepTOTP:
		m.totp, cmd = m.totp.Update(msg)
	case unlockStepReset:
		m.resetInput, cmd = m.resetInput.Update(msg)
	}
	return m, cmd
}

func (m UnlockModel) View() string {
	banner := viewUnlockBanner()

	var body string
	switch m.step {
	case unlockStepPassword:
		label := UnlockLabelStyle.Render("Enter master password")
		hint := UnlockHintStyle.Render("press enter to unlock")
		if m.isNewVault {
			hint = UnlockHintStyle.Render("new vault — choose a strong master password")
		}
		pw := FormActiveInputStyle.Width(36).Render(m.password.View())
		body = lipgloss.JoinVertical(lipgloss.Center, label, pw, hint)
	case unlockStepTOTP:
		label := UnlockLabelStyle.Render("Enter 2FA code")
		hint := UnlockHintStyle.Render("6-digit code from your authenticator app  •  esc to go back")
		tp := FormActiveInputStyle.Width(20).Render(m.totp.View())
		body = lipgloss.JoinVertical(lipgloss.Center, label, tp, hint)
	case unlockStepReset:
		warning := UnlockErrorStyle.Render("⚠  DANGER ZONE")
		desc := UnlockHintStyle.Render("This will permanently delete your vault and all krypt data.")
		desc2 := UnlockHintStyle.Render("This cannot be undone.")
		ri := FormActiveInputStyle.Width(36).Render(m.resetInput.View())
		body = lipgloss.JoinVertical(lipgloss.Center, warning, desc, desc2, ri)
	}

	if m.errMsg != "" {
		body = lipgloss.JoinVertical(lipgloss.Center, body, UnlockErrorStyle.Render("✗ "+m.errMsg))
	}

	box := UnlockBoxStyle.Render(body)

	var helpLine string
	switch m.step {
	case unlockStepReset:
		helpLine = HelpKeyStyle.Render("enter") + HelpSepStyle.Render("  confirm  •  ") +
			HelpKeyStyle.Render("esc") + HelpSepStyle.Render("  cancel  •  ") +
			HelpKeyStyle.Render("ctrl+c") + HelpSepStyle.Render("  quit")
	default:
		helpLine = HelpKeyStyle.Render("ctrl+r") + HelpSepStyle.Render("  reset vault  •  ") +
			HelpKeyStyle.Render("ctrl+c") + HelpSepStyle.Render("  quit")
	}

	content := lipgloss.JoinVertical(lipgloss.Center, banner, box, helpLine)

	if m.width > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func viewUnlockBanner() string {
	art := []string{
		`    __                    __ `,
		`   / /_________  ______  / /_`,
		`  / //_/ ___/ / / / __ \/ __/`,
		` / ,< / /  / /_/ / /_/ / /_  `,
		`/_/|_/_/   \__, / .___/\__/  `,
		`          /____/_/           `,
	}
	lines := make([]string, len(art))
	for i, l := range art {
		lines[i] = TitleStyle.Render(l)
	}
	return strings.Join(lines, "\n")
}

// unlockVerify2FAMsg carries the password + code for verification in the parent app.
type unlockVerify2FAMsg struct {
	password   string
	code       string
	isNewVault bool
}

// SetError sets an error message (called from parent when TOTP verification fails).
func (m *UnlockModel) SetError(msg string) { m.errMsg = msg }

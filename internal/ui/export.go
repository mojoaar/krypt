package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

type exportStep int

const (
	exportStepFormat     exportStep = iota // choose plaintext or encrypted
	exportStepPath                         // edit output path
	exportStepPassphrase                   // passphrase (encrypted only)
	exportStepConfirm                      // final warning before writing
)

// ExportDoneMsg is sent when the export completes successfully.
type ExportDoneMsg struct{ Path string }

// ExportCancelMsg is sent when the user cancels.
type ExportCancelMsg struct{}

// exportFormat mirrors the user's format choice.
type exportFormat int

const (
	exportFormatPlain exportFormat = iota
	exportFormatEncrypted
)

// ExportModel is the multi-step export overlay.
type ExportModel struct {
	step       exportStep
	format     exportFormat
	pathInput  textinput.Model
	passInput  textinput.Model
	pass2Input textinput.Model // passphrase confirm
	err        string
	width      int
	height     int
}

func NewExportModel() ExportModel {
	pi := textinput.New()
	pi.CharLimit = 512
	pi.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	pi.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	pw := textinput.New()
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.Placeholder = "passphrase…"
	pw.CharLimit = 128
	pw.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	pw.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	pw2 := textinput.New()
	pw2.EchoMode = textinput.EchoPassword
	pw2.EchoCharacter = '•'
	pw2.Placeholder = "confirm passphrase…"
	pw2.CharLimit = 128
	pw2.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	pw2.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	return ExportModel{
		step:       exportStepFormat,
		format:     exportFormatPlain,
		pathInput:  pi,
		passInput:  pw,
		pass2Input: pw2,
	}
}

func (m *ExportModel) Open() {
	m.step = exportStepFormat
	m.format = exportFormatPlain
	m.err = ""
	m.pathInput.SetValue(m.defaultPath(exportFormatPlain))
	m.passInput.SetValue("")
	m.pass2Input.SetValue("")
}

func (m *ExportModel) SetSize(w, h int) { m.width = w; m.height = h }

func (m ExportModel) defaultPath(f exportFormat) string {
	home, _ := os.UserHomeDir()
	date := time.Now().Format("2006-01-02")
	if f == exportFormatEncrypted {
		return fmt.Sprintf("%s/krypt-export-%s.enc.json", home, date)
	}
	return fmt.Sprintf("%s/krypt-export-%s.json", home, date)
}

func (m ExportModel) Update(msg tea.Msg) (ExportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case exportStepFormat:
			return m.updateFormat(msg)
		case exportStepPath:
			return m.updatePath(msg)
		case exportStepPassphrase:
			return m.updatePassphrase(msg)
		case exportStepConfirm:
			return m.updateConfirm(msg)
		}
	}
	return m, nil
}

func (m ExportModel) updateFormat(msg tea.KeyMsg) (ExportModel, tea.Cmd) {
	switch msg.String() {
	case "p":
		m.format = exportFormatPlain
		m.pathInput.SetValue(m.defaultPath(exportFormatPlain))
		m.step = exportStepPath
		m.pathInput.Focus()
	case "e":
		m.format = exportFormatEncrypted
		m.pathInput.SetValue(m.defaultPath(exportFormatEncrypted))
		m.step = exportStepPath
		m.pathInput.Focus()
	case "esc", "q":
		return m, func() tea.Msg { return ExportCancelMsg{} }
	}
	return m, nil
}

func (m ExportModel) updatePath(msg tea.KeyMsg) (ExportModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "enter":
		m.pathInput.Blur()
		if m.format == exportFormatEncrypted {
			m.step = exportStepPassphrase
			m.passInput.Focus()
		} else {
			m.step = exportStepConfirm
		}
	case "esc":
		m.pathInput.Blur()
		m.step = exportStepFormat
	default:
		m.pathInput, cmd = m.pathInput.Update(msg)
	}
	return m, cmd
}

func (m ExportModel) updatePassphrase(msg tea.KeyMsg) (ExportModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "enter":
		if m.passInput.Focused() {
			m.passInput.Blur()
			m.pass2Input.Focus()
			return m, nil
		}
		// on pass2 — validate
		if m.passInput.Value() == "" {
			m.err = "passphrase cannot be empty"
			m.passInput.Focus()
			m.pass2Input.Blur()
			return m, nil
		}
		if m.passInput.Value() != m.pass2Input.Value() {
			m.err = "passphrases do not match"
			m.pass2Input.SetValue("")
			m.passInput.SetValue("")
			m.pass2Input.Blur()
			m.passInput.Focus()
			return m, nil
		}
		m.err = ""
		m.pass2Input.Blur()
		m.step = exportStepConfirm
	case "esc":
		m.passInput.Blur()
		m.pass2Input.Blur()
		m.step = exportStepPath
		m.pathInput.Focus()
	case "tab":
		if m.passInput.Focused() {
			m.passInput.Blur()
			m.pass2Input.Focus()
		} else {
			m.pass2Input.Blur()
			m.passInput.Focus()
		}
	default:
		if m.passInput.Focused() {
			m.passInput, cmd = m.passInput.Update(msg)
		} else {
			m.pass2Input, cmd = m.pass2Input.Update(msg)
		}
	}
	return m, cmd
}

func (m ExportModel) updateConfirm(msg tea.KeyMsg) (ExportModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "y":
		path := strings.TrimSpace(m.pathInput.Value())
		if path == "" {
			path = m.defaultPath(m.format)
		}
		// expand ~ manually
		if strings.HasPrefix(path, "~/") {
			home, _ := os.UserHomeDir()
			path = home + path[1:]
		}
		var err error
		// entries are passed via ExportDoMsg — we do the write here via a Cmd
		passphrase := m.passInput.Value()
		exportPath := path
		exportFormat := m.format
		return m, func() tea.Msg {
			return exportWriteMsg{path: exportPath, format: exportFormat, passphrase: passphrase}
		}
		_ = err
	case "esc", "n":
		m.step = exportStepFormat
		return m, func() tea.Msg { return ExportCancelMsg{} }
	}
	return m, nil
}

// exportWriteMsg is an internal message that triggers the actual file write.
type exportWriteMsg struct {
	path       string
	format     exportFormat
	passphrase string
}

func (m ExportModel) View() string {
	var content string
	switch m.step {
	case exportStepFormat:
		content = m.viewFormat()
	case exportStepPath:
		content = m.viewPath()
	case exportStepPassphrase:
		content = m.viewPassphrase()
	case exportStepConfirm:
		content = m.viewConfirm()
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Padding(1, 3).
		Align(lipgloss.Center).
		Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m ExportModel) viewFormat() string {
	title := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("Export Vault")
	sub := HelpDescStyle.Render("Choose export format")

	pKey := HelpKeyStyle.Render("p")
	pDesc := HelpDescStyle.Render("  plaintext JSON  (human-readable, no password required)")
	eKey := HelpKeyStyle.Render("e")
	eDesc := HelpDescStyle.Render("  encrypted JSON  (AES-256-GCM, requires a passphrase)")

	warn := lipgloss.NewStyle().Foreground(colorWarning).Render("⚠  plaintext export contains unencrypted secrets")
	hint := HelpDescStyle.Render("esc cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, sub, "",
		pKey+pDesc,
		eKey+eDesc,
		"", warn, "", hint,
	)
}

func (m ExportModel) viewPath() string {
	fmtLabel := "plaintext"
	if m.format == exportFormatEncrypted {
		fmtLabel = "encrypted"
	}
	title := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).
		Render(fmt.Sprintf("Export — %s", fmtLabel))
	sub := HelpDescStyle.Render("Output path  (enter to confirm, esc to go back)")

	inputW := 60
	if m.width > 0 && m.width-14 < inputW {
		inputW = m.width - 14
	}
	m.pathInput.Width = inputW

	inputBox := FormActiveInputStyle.Render(m.pathInput.View())
	return lipgloss.JoinVertical(lipgloss.Left, title, sub, "", inputBox)
}

func (m ExportModel) viewPassphrase() string {
	title := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("Export — passphrase")
	sub := HelpDescStyle.Render("Set a passphrase to encrypt the export file")

	inputW := 40
	pwLabel := FormLabelStyle.Render("Passphrase")
	pw2Label := FormLabelStyle.Render("Confirm   ")

	m.passInput.Width = inputW
	m.pass2Input.Width = inputW

	var pw1Box, pw2Box string
	if m.passInput.Focused() {
		pw1Box = FormActiveInputStyle.Render(m.passInput.View())
		pw2Box = FormInputStyle.Render(m.pass2Input.View())
	} else {
		pw1Box = FormInputStyle.Render(m.passInput.View())
		pw2Box = FormActiveInputStyle.Render(m.pass2Input.View())
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Center, pwLabel+"  ", pw1Box)
	row2 := lipgloss.JoinHorizontal(lipgloss.Center, pw2Label+"  ", pw2Box)

	hint := HelpDescStyle.Render("tab switch  •  enter confirm  •  esc back")
	parts := []string{title, sub, "", row1, row2, ""}
	if m.err != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorDanger).Render("✗ "+m.err))
	}
	parts = append(parts, hint)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m ExportModel) viewConfirm() string {
	path := strings.TrimSpace(m.pathInput.Value())
	if path == "" {
		path = m.defaultPath(m.format)
	}

	fmtLabel := "plaintext JSON"
	fmtColor := colorWarning
	if m.format == exportFormatEncrypted {
		fmtLabel = "encrypted JSON"
		fmtColor = colorSuccess
	}

	title := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("Confirm export")
	fmtLine := HelpDescStyle.Render("Format  ") +
		lipgloss.NewStyle().Foreground(fmtColor).Bold(true).Render(fmtLabel)
	pathLine := HelpDescStyle.Render("Path    ") + DetailValueStyle.Render(path)

	var warn string
	if m.format == exportFormatPlain {
		warn = lipgloss.NewStyle().Foreground(colorWarning).Render("⚠  file will contain unencrypted secrets")
	} else {
		warn = lipgloss.NewStyle().Foreground(colorSuccess).Render("✓  file will be AES-256-GCM encrypted")
	}

	hint := HelpDescStyle.Render("enter / y  confirm  •  esc / n  cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", fmtLine, pathLine, "", warn, "", hint,
	)
}

// DoExport performs the actual write given vault entries. Called from app.go.
func DoExport(msg exportWriteMsg, entries []data.Entry) tea.Cmd {
	return func() tea.Msg {
		var err error
		if msg.format == exportFormatEncrypted {
			err = data.ExportEncryptedJSON(entries, msg.passphrase, msg.path)
		} else {
			err = data.ExportJSON(entries, msg.path)
		}
		if err != nil {
			return exportErrorMsg(err.Error())
		}
		return ExportDoneMsg{Path: msg.path}
	}
}

type exportErrorMsg string

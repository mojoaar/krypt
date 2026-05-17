package ui

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

// generatePassword returns a cryptographically random password using the given config.
func generatePassword(cfg data.PasswordGenConfig) string {
	charset := cfg.Charset()
	length := cfg.EffectiveLength()
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// FormSubmitMsg is emitted when the user submits the form.
type FormSubmitMsg struct {
	Entry data.Entry
}

// FormCancelMsg is emitted when the user cancels.
type FormCancelMsg struct{}

type formField int

// Login fields
const (
	fLoginName formField = iota
	fLoginURL
	fLoginUsername
	fLoginPassword
	fLoginNotes
	fLoginTags
	fLoginCount
)

// Note fields
const (
	fNoteName formField = iota
	fNoteContent
	fNoteTags
	fNoteSecure // toggle field — not a textinput
	fNoteCount
)

// Card fields
const (
	fCardName formField = iota
	fCardHolder
	fCardNumber
	fCardExpiry
	fCardCVC
	fCardPIN
	fCardBank
	fCardNotes
	fCardTags
	fCardCount
)

// Identity fields
const (
	fIdentityName formField = iota
	fIdentityFirst
	fIdentityLast
	fIdentityEmail
	fIdentityPhone
	fIdentityAddress
	fIdentityCompany
	fIdentitySSN
	fIdentityDriversLicense
	fIdentityPassport
	fIdentityNotes
	fIdentityNotesSecure
	fIdentityTags
	fIdentityCount
)

// SSH fields
const (
	fSSHName formField = iota
	fSSHHost
	fSSHPublicKey
	fSSHPrivateKey
	fSSHPassphrase
	fSSHTags
	fSSHCount
)

// typePickerMode is the initial state for a new entry where the user picks a type.
type typePickerMode bool

// Form is the add/edit overlay.
type Form struct {
	entryType           data.EntryType
	typePicker          bool // true when picking type for a new entry
	pickerIdx           int
	inputs              []textinput.Model
	noteArea            textarea.Model // multiline content for Note entries
	noteSecure          bool           // secure toggle for Note entries
	identityNoteArea    textarea.Model // multiline notes for Identity entries
	identityNotesSecure bool           // secure toggle for Identity notes
	cardNoteArea        textarea.Model // multiline notes for Card entries
	sshPrivKeyArea      textarea.Model // multiline private key for SSH entries
	sshPrivKeyRevealed  bool           // whether private key textarea is shown
	labels              []string
	masked              []bool // whether a field is masked by default
	showMasked          []bool // current reveal state per field
	focus               int
	scrollOffset        int    // first visible field index
	editID              string // non-empty when editing an existing entry
	width               int
	height              int
	pwGenCfg            data.PasswordGenConfig
	favorite            bool // shared across all entry types
}

func NewForm() Form { return Form{} }

// OpenNew starts the type-picker for a new entry.
func (f *Form) OpenNew() {
	f.typePicker = true
	f.pickerIdx = 0
	f.editID = ""
	f.scrollOffset = 0
	f.favorite = false
}

// OpenEdit pre-fills the form from an existing entry.
func (f *Form) OpenEdit(e data.Entry) {
	f.typePicker = false
	f.editID = e.ID
	f.entryType = e.Type
	f.scrollOffset = 0
	f.buildInputs()

	switch e.Type {
	case data.EntryTypeLogin:
		f.inputs[fLoginName].SetValue(e.Name)
		f.inputs[fLoginURL].SetValue(e.URL)
		f.inputs[fLoginUsername].SetValue(e.Username)
		f.inputs[fLoginPassword].SetValue(e.Password)
		f.inputs[fLoginNotes].SetValue(e.Notes)
		f.inputs[fLoginTags].SetValue(strings.Join(e.Tags, ", "))
	case data.EntryTypeNote:
		f.inputs[fNoteName].SetValue(e.Name)
		f.noteArea.SetValue(e.Content)
		f.noteSecure = e.Secure
		f.inputs[fNoteTags].SetValue(strings.Join(e.Tags, ", "))
	case data.EntryTypeCard:
		f.inputs[fCardName].SetValue(e.Name)
		f.inputs[fCardHolder].SetValue(e.CardHolder)
		f.inputs[fCardNumber].SetValue(e.CardNumber)
		f.inputs[fCardExpiry].SetValue(e.Expiry)
		f.inputs[fCardCVC].SetValue(e.CVV)
		f.inputs[fCardPIN].SetValue(e.PIN)
		f.inputs[fCardBank].SetValue(e.Bank)
		f.cardNoteArea.SetValue(e.CardNotes)
		f.inputs[fCardTags].SetValue(strings.Join(e.Tags, ", "))
	case data.EntryTypeIdentity:
		f.inputs[fIdentityName].SetValue(e.Name)
		f.inputs[fIdentityFirst].SetValue(e.FirstName)
		f.inputs[fIdentityLast].SetValue(e.LastName)
		f.inputs[fIdentityEmail].SetValue(e.Email)
		f.inputs[fIdentityPhone].SetValue(e.Phone)
		f.inputs[fIdentityAddress].SetValue(e.Address)
		f.inputs[fIdentityCompany].SetValue(e.Company)
		f.inputs[fIdentitySSN].SetValue(e.SSN)
		f.inputs[fIdentityDriversLicense].SetValue(e.DriversLicense)
		f.inputs[fIdentityPassport].SetValue(e.PassportNumber)
		f.identityNoteArea.SetValue(e.IdentityNotes)
		f.identityNotesSecure = e.IdentityNotesSecure
		f.inputs[fIdentityTags].SetValue(strings.Join(e.Tags, ", "))
	case data.EntryTypeSSHKey:
		f.inputs[fSSHName].SetValue(e.Name)
		f.inputs[fSSHHost].SetValue(e.Host)
		f.inputs[fSSHPublicKey].SetValue(e.PublicKey)
		f.sshPrivKeyArea.SetValue(e.PrivateKey)
		f.inputs[fSSHPassphrase].SetValue(e.Passphrase)
		f.inputs[fSSHTags].SetValue(strings.Join(e.Tags, ", "))
	}

	f.focus = 0
	f.focusAt(0)
	f.favorite = e.Favorite
}

func (f *Form) buildInputs() {
	switch f.entryType {
	case data.EntryTypeLogin:
		f.labels = []string{"Name", "URL", "Username", "Password", "Notes", "Tags"}
		f.masked = []bool{false, false, false, true, false, false}
		f.inputs = makeInputs(int(fLoginCount), f.masked)
		f.inputs[fLoginPassword].Placeholder = "••••••••"
		f.inputs[fLoginTags].Placeholder = "work, personal"
	case data.EntryTypeNote:
		f.labels = []string{"Name", "Content", "Tags", "Secure"}
		f.masked = []bool{false, false, false, false}
		f.inputs = makeInputs(int(fNoteCount), f.masked)
		f.inputs[fNoteTags].Placeholder = "work, personal"
		f.noteArea = newNoteArea()
		f.noteSecure = false
	case data.EntryTypeCard:
		f.labels = []string{"Name", "Card Holder", "Card Number", "Expiry", "CVC", "PIN", "Bank", "Notes", "Tags"}
		f.masked = []bool{false, false, true, false, true, true, false, false, false}
		f.inputs = makeInputs(int(fCardCount), f.masked)
		f.inputs[fCardExpiry].Placeholder = "MM/YY"
		f.inputs[fCardCVC].Placeholder = "123"
		f.inputs[fCardPIN].Placeholder = "1234"
		f.inputs[fCardTags].Placeholder = "work, personal"
		f.cardNoteArea = newNoteArea()
	case data.EntryTypeIdentity:
		f.labels = []string{"Name", "First Name", "Last Name", "Email", "Phone", "Address", "Company", "SSN (Social Security Number)", "Drivers License", "Passport Number", "Notes", "Secure Notes", "Tags"}
		f.masked = []bool{false, false, false, false, false, false, false, true, true, true, false, false, false}
		f.inputs = makeInputs(int(fIdentityCount), f.masked)
		f.identityNoteArea = newNoteArea()
		f.inputs[fIdentityTags].Placeholder = "work, personal"
	case data.EntryTypeSSHKey:
		f.labels = []string{"Name", "Host", "Public Key", "Private Key", "Passphrase", "Tags"}
		f.masked = []bool{false, false, false, false, true, false}
		f.inputs = makeInputs(int(fSSHCount), f.masked)
		f.sshPrivKeyArea = newSSHKeyArea()
		f.sshPrivKeyRevealed = false
		f.inputs[fSSHPublicKey].Placeholder = "ssh-ed25519 AAAA…"
		f.inputs[fSSHTags].Placeholder = "work, personal"
	}
	f.showMasked = make([]bool, len(f.masked))
	f.focus = 0
	f.focusAt(0)
}

func makeInputs(n int, masked []bool) []textinput.Model {
	inputs := make([]textinput.Model, n)
	for i := range inputs {
		t := textinput.New()
		t.CharLimit = 512
		if i < len(masked) && masked[i] {
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		inputs[i] = t
	}
	return inputs
}

func newNoteArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "write your note here…"
	ta.SetHeight(5)
	ta.SetWidth(60) // will be resized in viewFields
	ta.CharLimit = 8192
	ta.ShowLineNumbers = false
	// Style to match the form palette
	ta.FocusedStyle.Base = FormActiveInputStyle.Copy()
	ta.BlurredStyle.Base = FormInputStyle.Copy()
	return ta
}

func newSSHKeyArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "-----BEGIN OPENSSH PRIVATE KEY-----\n…\n-----END OPENSSH PRIVATE KEY-----"
	ta.SetHeight(8)
	ta.SetWidth(60) // will be resized in viewFields
	ta.CharLimit = 16384
	ta.ShowLineNumbers = false
	ta.FocusedStyle.Base = FormActiveInputStyle.Copy()
	ta.BlurredStyle.Base = FormInputStyle.Copy()
	return ta
}

func (f *Form) focusAt(i int) {
	for j := range f.inputs {
		if j == i {
			f.inputs[j].Focus()
		} else {
			f.inputs[j].Blur()
		}
	}
	// For Note content: focus/blur the textarea instead of the dummy input
	if f.entryType == data.EntryTypeNote {
		if i == int(fNoteContent) {
			f.noteArea.Focus()
			f.inputs[fNoteContent].Blur()
		} else {
			f.noteArea.Blur()
		}
		if i == int(fNoteSecure) {
			f.inputs[fNoteSecure].Blur()
		}
	}
	// For Identity notes textarea and secure toggle
	if f.entryType == data.EntryTypeIdentity {
		if i == int(fIdentityNotes) {
			f.identityNoteArea.Focus()
			f.inputs[fIdentityNotes].Blur()
		} else {
			f.identityNoteArea.Blur()
		}
		if i == int(fIdentityNotesSecure) {
			f.inputs[fIdentityNotesSecure].Blur()
		}
	}
	// For Card notes textarea
	if f.entryType == data.EntryTypeCard {
		if i == int(fCardNotes) {
			f.cardNoteArea.Focus()
			f.inputs[fCardNotes].Blur()
		} else {
			f.cardNoteArea.Blur()
		}
	}
	// For SSH private key textarea
	if f.entryType == data.EntryTypeSSHKey {
		if i == int(fSSHPrivateKey) && f.sshPrivKeyRevealed {
			f.sshPrivKeyArea.Focus()
			f.inputs[fSSHPrivateKey].Blur()
		} else {
			f.sshPrivKeyArea.Blur()
		}
	}
	f.scrollTo(i)
}

// scrollTo adjusts scrollOffset so that field i is visible.
// Each field renders as 4 rows (label + bordered input). Box overhead = 8 rows.
func (f *Form) scrollTo(i int) {
	const rowsPerField = 4
	const overhead = 8

	total := len(f.inputs)
	if total*rowsPerField+overhead <= f.height {
		return // everything fits, no scrolling needed
	}
	// Reserve 2 rows for ↑/↓ indicators
	visibleFields := (f.height - overhead - 2) / rowsPerField
	if visibleFields < 1 {
		visibleFields = 1
	}
	if i < f.scrollOffset {
		f.scrollOffset = i
	} else if i >= f.scrollOffset+visibleFields {
		f.scrollOffset = i - visibleFields + 1
	}
	maxOffset := total - visibleFields
	if maxOffset < 0 {
		maxOffset = 0
	}
	if f.scrollOffset > maxOffset {
		f.scrollOffset = maxOffset
	}
	if f.scrollOffset < 0 {
		f.scrollOffset = 0
	}
}

func (f *Form) SetSize(w, h int) { f.width = w; f.height = h }

func (f Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+r":
			if !f.typePicker && f.focus < len(f.masked) && f.masked[f.focus] {
				f.showMasked[f.focus] = !f.showMasked[f.focus]
				if f.showMasked[f.focus] {
					f.inputs[f.focus].EchoMode = textinput.EchoNormal
				} else {
					f.inputs[f.focus].EchoMode = textinput.EchoPassword
					f.inputs[f.focus].EchoCharacter = '•'
				}
			}
			// Toggle SSH private key textarea reveal
			if !f.typePicker && f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) {
				f.sshPrivKeyRevealed = !f.sshPrivKeyRevealed
				if f.sshPrivKeyRevealed {
					f.sshPrivKeyArea.Focus()
					f.inputs[fSSHPrivateKey].Blur()
				} else {
					f.sshPrivKeyArea.Blur()
				}
			}
			return f, nil

		case "ctrl+g":
			// Generate a password for Login password or SSH passphrase.
			isLoginPw := f.entryType == data.EntryTypeLogin && f.focus == int(fLoginPassword)
			isSSHPass := f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPassphrase)
			if !f.typePicker && (isLoginPw || isSSHPass) {
				pw := generatePassword(f.pwGenCfg)
				f.inputs[f.focus].SetValue(pw)
				// Reveal so user can see the generated password.
				f.showMasked[f.focus] = true
				f.inputs[f.focus].EchoMode = textinput.EchoNormal
			}
			return f, nil

		case "esc":
			return f, func() tea.Msg { return FormCancelMsg{} }

		case " ":
			// Toggle the secure field when focused on it
			if f.entryType == data.EntryTypeNote && f.focus == int(fNoteSecure) {
				f.noteSecure = !f.noteSecure
				return f, nil
			}
			if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotesSecure) {
				f.identityNotesSecure = !f.identityNotesSecure
				return f, nil
			}
			// Favorite toggle
			if !f.typePicker && f.focus == len(f.inputs) {
				f.favorite = !f.favorite
				return f, nil
			}

		case "ctrl+s", "enter":
			if f.typePicker {
				f.entryType = data.AllEntryTypes[f.pickerIdx]
				f.typePicker = false
				f.buildInputs()
				return f, textinput.Blink
			}
			// Favorite toggle field: enter toggles, ctrl+s submits
			if !f.typePicker && f.focus == len(f.inputs) {
				if msg.String() == "enter" {
					f.favorite = !f.favorite
					return f, nil
				}
				// ctrl+s falls through to submit
				return f, func() tea.Msg { return FormSubmitMsg{Entry: f.buildEntry()} }
			}
			// On secure toggle field, enter also toggles
			if f.entryType == data.EntryTypeNote && f.focus == int(fNoteSecure) && msg.String() == "enter" {
				f.noteSecure = !f.noteSecure
				return f, nil
			}
			if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotesSecure) && msg.String() == "enter" {
				f.identityNotesSecure = !f.identityNotesSecure
				return f, nil
			}
			// In textareas, enter inserts a newline — let it fall through
			if f.entryType == data.EntryTypeNote && f.focus == int(fNoteContent) && msg.String() == "enter" {
				break
			}
			if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotes) && msg.String() == "enter" {
				break
			}
			if f.entryType == data.EntryTypeCard && f.focus == int(fCardNotes) && msg.String() == "enter" {
				break
			}
			if f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) && f.sshPrivKeyRevealed && msg.String() == "enter" {
				break
			}
			if msg.String() == "enter" && f.focus < len(f.inputs) {
				f.focus++
				f.focusAt(f.focus)
				return f, textinput.Blink
			}
			if msg.String() == "ctrl+s" || f.focus == len(f.inputs)-1 {
				return f, func() tea.Msg { return FormSubmitMsg{Entry: f.buildEntry()} }
			}

		case "shift+tab", "up":
			if f.typePicker {
				if f.pickerIdx > 0 {
					f.pickerIdx--
				}
				return f, nil
			}
			// From favorite field, go back to last input
			if !f.typePicker && f.focus == len(f.inputs) {
				f.focus = len(f.inputs) - 1
				f.focusAt(f.focus)
				return f, textinput.Blink
			}
			// Don't let "up" steal from textareas
			if f.entryType == data.EntryTypeNote && f.focus == int(fNoteContent) && msg.String() == "up" {
				break
			}
			if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotes) && msg.String() == "up" {
				break
			}
			if f.entryType == data.EntryTypeCard && f.focus == int(fCardNotes) && msg.String() == "up" {
				break
			}
			if f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) && f.sshPrivKeyRevealed && msg.String() == "up" {
				break
			}
			if f.focus > 0 {
				f.focus--
				f.focusAt(f.focus)
			}
			return f, textinput.Blink

		case "tab", "down":
			if f.typePicker {
				if f.pickerIdx < len(data.AllEntryTypes)-1 {
					f.pickerIdx++
				}
				return f, nil
			}
			// Don't let "down" steal from textareas
			if f.entryType == data.EntryTypeNote && f.focus == int(fNoteContent) && msg.String() == "down" {
				break
			}
			if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotes) && msg.String() == "down" {
				break
			}
			if f.entryType == data.EntryTypeCard && f.focus == int(fCardNotes) && msg.String() == "down" {
				break
			}
			if f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) && f.sshPrivKeyRevealed && msg.String() == "down" {
				break
			}
			if f.focus < len(f.inputs) {
				f.focus++
				f.focusAt(f.focus)
			}
			return f, textinput.Blink
		}
	}

	if f.typePicker || f.focus > len(f.inputs) {
		return f, nil
	}
	// Favorite toggle: don't forward to textinput
	if !f.typePicker && f.focus == len(f.inputs) {
		return f, nil
	}
	// Note secure toggle: don't forward to textinput
	if f.entryType == data.EntryTypeNote && f.focus == int(fNoteSecure) {
		return f, nil
	}
	// Note content field: route to textarea
	if f.entryType == data.EntryTypeNote && f.focus == int(fNoteContent) {
		var cmd tea.Cmd
		f.noteArea, cmd = f.noteArea.Update(msg)
		return f, cmd
	}
	// Identity notes secure toggle: don't forward to textinput
	if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotesSecure) {
		return f, nil
	}
	// Identity notes field: route to textarea
	if f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotes) {
		var cmd tea.Cmd
		f.identityNoteArea, cmd = f.identityNoteArea.Update(msg)
		return f, cmd
	}
	// Card notes field: route to textarea
	if f.entryType == data.EntryTypeCard && f.focus == int(fCardNotes) {
		var cmd tea.Cmd
		f.cardNoteArea, cmd = f.cardNoteArea.Update(msg)
		return f, cmd
	}
	// SSH private key: route to textarea when revealed
	if f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) && f.sshPrivKeyRevealed {
		var cmd tea.Cmd
		f.sshPrivKeyArea, cmd = f.sshPrivKeyArea.Update(msg)
		return f, cmd
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return f, cmd
}

func (f Form) buildEntry() data.Entry {
	parseTags := func(raw string) []string {
		parts := strings.Split(raw, ",")
		tags := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				tags = append(tags, t)
			}
		}
		return tags
	}

	e := data.Entry{ID: f.editID, Type: f.entryType, Favorite: f.favorite}
	switch f.entryType {
	case data.EntryTypeLogin:
		e.Name = f.inputs[fLoginName].Value()
		e.URL = f.inputs[fLoginURL].Value()
		e.Username = f.inputs[fLoginUsername].Value()
		e.Password = f.inputs[fLoginPassword].Value()
		e.Notes = f.inputs[fLoginNotes].Value()
		e.Tags = parseTags(f.inputs[fLoginTags].Value())
	case data.EntryTypeNote:
		e.Name = f.inputs[fNoteName].Value()
		e.Content = f.noteArea.Value()
		e.Secure = f.noteSecure
		e.Tags = parseTags(f.inputs[fNoteTags].Value())
	case data.EntryTypeCard:
		e.Name = f.inputs[fCardName].Value()
		e.CardHolder = f.inputs[fCardHolder].Value()
		e.CardNumber = f.inputs[fCardNumber].Value()
		e.Expiry = f.inputs[fCardExpiry].Value()
		e.CVV = f.inputs[fCardCVC].Value()
		e.PIN = f.inputs[fCardPIN].Value()
		e.Bank = f.inputs[fCardBank].Value()
		e.CardNotes = f.cardNoteArea.Value()
		e.Tags = parseTags(f.inputs[fCardTags].Value())
	case data.EntryTypeIdentity:
		e.Name = f.inputs[fIdentityName].Value()
		e.FirstName = f.inputs[fIdentityFirst].Value()
		e.LastName = f.inputs[fIdentityLast].Value()
		e.Email = f.inputs[fIdentityEmail].Value()
		e.Phone = f.inputs[fIdentityPhone].Value()
		e.Address = f.inputs[fIdentityAddress].Value()
		e.Company = f.inputs[fIdentityCompany].Value()
		e.SSN = f.inputs[fIdentitySSN].Value()
		e.DriversLicense = f.inputs[fIdentityDriversLicense].Value()
		e.PassportNumber = f.inputs[fIdentityPassport].Value()
		e.IdentityNotes = f.identityNoteArea.Value()
		e.IdentityNotesSecure = f.identityNotesSecure
		e.Tags = parseTags(f.inputs[fIdentityTags].Value())
	case data.EntryTypeSSHKey:
		e.Name = f.inputs[fSSHName].Value()
		e.Host = f.inputs[fSSHHost].Value()
		e.PublicKey = f.inputs[fSSHPublicKey].Value()
		e.PrivateKey = f.sshPrivKeyArea.Value()
		e.Passphrase = f.inputs[fSSHPassphrase].Value()
		e.Tags = parseTags(f.inputs[fSSHTags].Value())
	}
	return e
}

func (f Form) View() string {
	var title string
	if f.editID != "" {
		title = FormTitleStyle.Render("Edit entry")
	} else {
		title = FormTitleStyle.Render("New entry")
	}

	var body string
	if f.typePicker {
		body = f.viewTypePicker()
	} else {
		body = f.viewFields()
	}

	hint := HelpKeyStyle.Render("tab/↑↓") + HelpDescStyle.Render(" navigate  ") +
		HelpKeyStyle.Render("ctrl+s") + HelpDescStyle.Render(" save  ") +
		HelpKeyStyle.Render("esc") + HelpDescStyle.Render(" cancel")

	// If the focused field is masked, add a reveal hint
	if !f.typePicker && f.focus < len(f.masked) && f.masked[f.focus] {
		revealLabel := "show"
		if f.focus < len(f.showMasked) && f.showMasked[f.focus] {
			revealLabel = "hide"
		}
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("ctrl+r") + HelpDescStyle.Render(" "+revealLabel)
	}
	// Password / passphrase generator hint
	isLoginPwFocused := !f.typePicker && f.entryType == data.EntryTypeLogin && f.focus == int(fLoginPassword)
	isSSHPassFocused := !f.typePicker && f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPassphrase)
	if isLoginPwFocused || isSSHPassFocused {
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("ctrl+g") + HelpDescStyle.Render(" generate")
	}
	// SSH private key reveal hint
	if !f.typePicker && f.entryType == data.EntryTypeSSHKey && f.focus == int(fSSHPrivateKey) {
		revealLabel := "show private key"
		if f.sshPrivKeyRevealed {
			revealLabel = "hide private key"
		}
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("ctrl+r") + HelpDescStyle.Render(" "+revealLabel)
	}
	// Favorite toggle hint
	if !f.typePicker && f.focus == len(f.inputs) {
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("space/enter") + HelpDescStyle.Render(" toggle")
	}
	// Secure toggle hints
	if !f.typePicker && f.entryType == data.EntryTypeNote && f.focus == int(fNoteSecure) {
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("space/enter") + HelpDescStyle.Render(" toggle")
	}
	if !f.typePicker && f.entryType == data.EntryTypeIdentity && f.focus == int(fIdentityNotesSecure) {
		hint += HelpDescStyle.Render("  ") +
			HelpKeyStyle.Render("space/enter") + HelpDescStyle.Render(" toggle")
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, title, body, "", hint)
	maxW := f.width - 8
	if maxW < 40 {
		maxW = 40
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Width(maxW).
		Render(inner)

	if f.width > 0 && f.height > 0 {
		return lipgloss.Place(f.width, f.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (f Form) viewTypePicker() string {
	lines := []string{
		FormLabelStyle.Render("Select entry type") + "\n",
	}
	for i, t := range data.AllEntryTypes {
		label := "  " + data.EntryTypeLabel(t)
		if i == f.pickerIdx {
			lines = append(lines, ListSelectedStyle.Render("▸ "+data.EntryTypeLabel(t)))
		} else {
			lines = append(lines, SidebarItemStyle.Render(label))
		}
	}
	lines = append(lines, "", HelpDescStyle.Render("↑↓ or tab to move  •  enter to select"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (f Form) viewFields() string {
	inputW := f.width - 20
	if inputW < 36 {
		inputW = 36
	}

	const rowsPerField = 4
	const overhead = 8

	total := len(f.inputs)
	var visibleFields int
	if total*rowsPerField+overhead <= f.height {
		visibleFields = total // all fields fit
	} else {
		visibleFields = (f.height - overhead - 2) / rowsPerField
		if visibleFields < 1 {
			visibleFields = 1
		}
	}

	start := f.scrollOffset
	end := start + visibleFields
	if end > total {
		end = total
	}

	var rows []string

	// "more above" indicator
	if start > 0 {
		rows = append(rows, HelpDescStyle.Render("  ↑ more fields above"))
	}

	for i := start; i < end; i++ {
		inp := f.inputs[i]
		label := f.labels[i]

		// SSH private key: show textarea when revealed, masked placeholder when not
		if f.entryType == data.EntryTypeSSHKey && i == int(fSSHPrivateKey) {
			rows = append(rows, FormLabelStyle.Render(label))
			if f.sshPrivKeyRevealed {
				f.sshPrivKeyArea.SetWidth(inputW)
				rows = append(rows, f.sshPrivKeyArea.View())
			} else {
				hint := "•••• (ctrl+r to reveal and edit)"
				rows = append(rows, FormInputStyle.Width(inputW).Render(hint))
			}
			continue
		}

		// Identity notes textarea
		if f.entryType == data.EntryTypeIdentity && i == int(fIdentityNotes) {
			f.identityNoteArea.SetWidth(inputW)
			rows = append(rows, FormLabelStyle.Render(label))
			rows = append(rows, f.identityNoteArea.View())
			continue
		}

		// Card notes textarea
		if f.entryType == data.EntryTypeCard && i == int(fCardNotes) {
			f.cardNoteArea.SetWidth(inputW)
			rows = append(rows, FormLabelStyle.Render(label))
			rows = append(rows, f.cardNoteArea.View())
			continue
		}

		// Identity notes secure toggle
		if f.entryType == data.EntryTypeIdentity && i == int(fIdentityNotesSecure) {
			checkmark := "[ ] no"
			if f.identityNotesSecure {
				checkmark = "[✓] yes"
			}
			var rendered string
			if i == f.focus {
				rendered = FormActiveInputStyle.Width(inputW).Render(checkmark)
			} else {
				rendered = FormInputStyle.Width(inputW).Render(checkmark)
			}
			rows = append(rows, FormLabelStyle.Render(label))
			rows = append(rows, rendered)
			continue
		}

		// Note secure toggle
		if f.entryType == data.EntryTypeNote && i == int(fNoteSecure) {
			checkmark := "[ ] no"
			if f.noteSecure {
				checkmark = "[✓] yes"
			}
			var rendered string
			if i == f.focus {
				rendered = FormActiveInputStyle.Width(inputW).Render(checkmark)
			} else {
				rendered = FormInputStyle.Width(inputW).Render(checkmark)
			}
			rows = append(rows, FormLabelStyle.Render(label))
			rows = append(rows, rendered)
			continue
		}

		// Note content: render textarea instead of textinput
		if f.entryType == data.EntryTypeNote && i == int(fNoteContent) {
			f.noteArea.SetWidth(inputW)
			rows = append(rows, FormLabelStyle.Render(label))
			rows = append(rows, f.noteArea.View())
			continue
		}

		var renderedInput string
		if i == f.focus {
			renderedInput = FormActiveInputStyle.Width(inputW).Render(inp.View())
		} else {
			renderedInput = FormInputStyle.Width(inputW).Render(inp.View())
		}

		// For password / passphrase fields, show a right-aligned [ctrl+g] generate hint on the label row.
		isLoginPw := f.entryType == data.EntryTypeLogin && i == int(fLoginPassword)
		isSSHPass := f.entryType == data.EntryTypeSSHKey && i == int(fSSHPassphrase)
		if (isLoginPw || isSSHPass) && i == f.focus {
			genHint := HelpKeyStyle.Render("ctrl+g") + HelpDescStyle.Render(" generate")
			labelW := lipgloss.Width(renderedInput) // match exact rendered width of input box
			labelText := FormLabelStyle.Render(label)
			gap := labelW - lipgloss.Width(labelText) - lipgloss.Width(genHint)
			if gap < 1 {
				gap = 1
			}
			rows = append(rows, labelText+strings.Repeat(" ", gap)+genHint)
		} else {
			rows = append(rows, FormLabelStyle.Render(label))
		}
		rows = append(rows, renderedInput)
	}

	// Favorite toggle — always shown as the last field for all entry types
	if end >= total {
		favLabel := FormLabelStyle.Render("Favorite")
		favCheck := "[ ] no"
		if f.favorite {
			favCheck = "[★] yes"
		}
		var favRendered string
		if f.focus == len(f.inputs) {
			favRendered = FormActiveInputStyle.Width(inputW).Render(favCheck)
		} else {
			favRendered = FormInputStyle.Width(inputW).Render(favCheck)
		}
		rows = append(rows, favLabel)
		rows = append(rows, favRendered)
	}

	// "more below" indicator
	if end < total {
		rows = append(rows, HelpDescStyle.Render("  ↓ more fields below"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

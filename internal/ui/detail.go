package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

// DetailView renders the full detail of a selected entry.
type DetailView struct {
	entry          data.Entry
	showPassword   bool
	showCVC        bool
	showCardNumber bool
	showContent    bool
	showIdentity   bool // reveal SSN + drivers license on Identity
	showPrivKey    bool
	showPass       bool
	width          int
	height         int
}

func NewDetailView() DetailView { return DetailView{} }

func (d *DetailView) SetEntry(e data.Entry) {
	d.entry = e
	d.showPassword = false
	d.showCVC = false
	d.showCardNumber = false
	d.showContent = false
	d.showIdentity = false
	d.showPrivKey = false
	d.showPass = false
}

func (d *DetailView) SetSize(w, h int) { d.width = w; d.height = h }

func (d *DetailView) TogglePassword()   { d.showPassword = !d.showPassword }
func (d *DetailView) ToggleCVC()        { d.showCVC = !d.showCVC }
func (d *DetailView) ToggleCardNumber() { d.showCardNumber = !d.showCardNumber }
func (d *DetailView) ToggleContent()    { d.showContent = !d.showContent }
func (d *DetailView) ToggleIdentity()   { d.showIdentity = !d.showIdentity }
func (d *DetailView) TogglePrivKey()    { d.showPrivKey = !d.showPrivKey }

func (d DetailView) View() string {
	e := d.entry
	badge := BadgeStyle(string(e.Type)).Render(data.EntryTypeBadge(e.Type))
	title := DetailTitleStyle.Render(badge + "  " + e.Name)

	var rows []string
	rows = append(rows, title)

	switch e.Type {
	case data.EntryTypeLogin:
		if e.URL != "" {
			rows = append(rows, d.rowLink("URL", e.URL))
		}
		rows = append(rows, d.row("Username", e.Username))
		if d.showPassword {
			rows = append(rows, d.row("Password", e.Password))
		} else {
			rows = append(rows, d.row("Password", masked(e.Password)))
		}
		if e.Notes != "" {
			rows = append(rows, d.row("Notes", e.Notes))
		}

	case data.EntryTypeNote:
		if e.Secure && !d.showContent {
			rows = append(rows, d.row("Content", masked(e.Content)))
		} else {
			rows = append(rows, d.rowMultiline("Content", e.Content))
		}

	case data.EntryTypeCard:
		rows = append(rows, d.row("Card Holder", e.CardHolder))
		rows = append(rows, d.row("Bank", e.Bank))
		if d.showCardNumber {
			rows = append(rows, d.row("Card Number", formatCardNumber(e.CardNumber)))
		} else if len(e.CardNumber) >= 4 {
			rows = append(rows, d.row("Card Number", "•••• •••• •••• "+e.CardNumber[len(e.CardNumber)-4:]))
		}
		rows = append(rows, d.row("Expiry", e.Expiry))
		if d.showCVC {
			rows = append(rows, d.row("CVC", e.CVV))
		} else {
			rows = append(rows, d.row("CVC", "•••"))
		}
		if e.PIN != "" {
			if d.showCVC {
				rows = append(rows, d.row("PIN", e.PIN))
			} else {
				rows = append(rows, d.row("PIN", "••••"))
			}
		}
		if e.CardNotes != "" {
			rows = append(rows, d.rowMultiline("Notes", e.CardNotes))
		}

	case data.EntryTypeIdentity:
		rows = append(rows, d.row("Name", e.FirstName+" "+e.LastName))
		rows = append(rows, d.row("Email", e.Email))
		rows = append(rows, d.row("Phone", e.Phone))
		rows = append(rows, d.row("Company", e.Company))
		rows = append(rows, d.row("Address", e.Address))
		if e.SSN != "" {
			if d.showIdentity {
				rows = append(rows, d.row("SSN", e.SSN))
			} else {
				rows = append(rows, d.row("SSN", masked(e.SSN)))
			}
		}
		if e.DriversLicense != "" {
			if d.showIdentity {
				rows = append(rows, d.row("Drivers License", e.DriversLicense))
			} else {
				rows = append(rows, d.row("Drivers License", masked(e.DriversLicense)))
			}
		}
		if e.PassportNumber != "" {
			if d.showIdentity {
				rows = append(rows, d.row("Passport Number", e.PassportNumber))
			} else {
				rows = append(rows, d.row("Passport Number", masked(e.PassportNumber)))
			}
		}
		if e.IdentityNotes != "" {
			if e.IdentityNotesSecure && !d.showIdentity {
				rows = append(rows, d.row("Notes", masked(e.IdentityNotes)))
			} else {
				rows = append(rows, d.rowMultiline("Notes", e.IdentityNotes))
			}
		}

	case data.EntryTypeSSHKey:
		rows = append(rows, d.row("Host", e.Host))
		rows = append(rows, d.row("Public Key", truncateKey(e.PublicKey, 48)))
		if d.showPrivKey {
			rows = append(rows, d.rowMultiline("Private Key", e.PrivateKey))
		} else {
			rows = append(rows, d.row("Private Key", masked(e.PrivateKey)))
		}
		if e.Passphrase != "" {
			rows = append(rows, d.row("Passphrase", masked(e.Passphrase)))
		}
	}

	// Tags
	rows = append(rows, "")
	rows = append(rows, DetailLabelStyle.Render("Tags")+"  "+RenderTagBadges(e.Tags))

	// Help hints
	rows = append(rows, "")
	rows = append(rows, d.helpBar())

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	box := DetailBoxStyle.Width(d.width - 4).Render(content)
	return ListStyle.Render(box)
}

func (d DetailView) row(label, value string) string {
	lbl := DetailLabelStyle.Render(fmt.Sprintf("%-14s", label))
	val := DetailValueStyle.Render(value)
	return lbl + val
}

// rowLink renders a row where the value is a terminal OSC 8 hyperlink.
// Falls back to plain text in terminals that don't support hyperlinks.
func (d DetailView) rowLink(label, url string) string {
	lbl := DetailLabelStyle.Render(fmt.Sprintf("%-14s", label))
	// OSC 8 hyperlink: \e]8;;URL\e\\TEXT\e]8;;\e\\
	link := "\033]8;;" + url + "\033\\" + url + "\033]8;;\033\\"
	val := DetailValueStyle.Render(link)
	return lbl + val
}

func (d DetailView) rowMultiline(label, value string) string {
	lbl := DetailLabelStyle.Render(fmt.Sprintf("%-14s", label))
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return lbl + DetailValueStyle.Render("—")
	}
	first := lbl + DetailValueStyle.Render(lines[0])
	rest := make([]string, len(lines)-1)
	for i, l := range lines[1:] {
		rest[i] = strings.Repeat(" ", 14) + DetailValueStyle.Render(l)
	}
	return strings.Join(append([]string{first}, rest...), "\n")
}

func (d DetailView) helpBar() string {
	parts := []string{
		HelpKeyStyle.Render("esc") + HelpSepStyle.Render(" back"),
		HelpKeyStyle.Render("e") + HelpSepStyle.Render(" edit"),
		HelpKeyStyle.Render("d") + HelpSepStyle.Render(" delete"),
	}
	switch d.entry.Type {
	case data.EntryTypeNote:
		parts = append(parts,
			HelpKeyStyle.Render("c")+HelpSepStyle.Render(" copy content"),
		)
		if d.entry.Secure {
			revealLabel := " reveal content"
			if d.showContent {
				revealLabel = " hide content"
			}
			parts = append(parts,
				HelpKeyStyle.Render("space")+HelpSepStyle.Render(revealLabel),
			)
		}
	case data.EntryTypeLogin:
		parts = append(parts,
			HelpKeyStyle.Render("u") + HelpSepStyle.Render(" copy username"),
			HelpKeyStyle.Render("p") + HelpSepStyle.Render(" copy password"),
			HelpKeyStyle.Render("space") + HelpSepStyle.Render(" reveal password"),
		)
	case data.EntryTypeCard:
		parts = append(parts,
			HelpKeyStyle.Render("n")+HelpSepStyle.Render(" copy number"),
			HelpKeyStyle.Render("x")+HelpSepStyle.Render(" copy expiry"),
			HelpKeyStyle.Render("c")+HelpSepStyle.Render(" copy cvc"),
			HelpKeyStyle.Render("i")+HelpSepStyle.Render(" copy pin"),
			HelpKeyStyle.Render("space")+HelpSepStyle.Render(" reveal all"),
		)
	case data.EntryTypeIdentity:
		parts = append(parts,
			HelpKeyStyle.Render("u")+HelpSepStyle.Render(" copy email"),
			HelpKeyStyle.Render("s")+HelpSepStyle.Render(" copy ssn"),
			HelpKeyStyle.Render("l")+HelpSepStyle.Render(" copy license"),
			HelpKeyStyle.Render("b")+HelpSepStyle.Render(" copy passport"),
			HelpKeyStyle.Render("o")+HelpSepStyle.Render(" copy notes"),
			HelpKeyStyle.Render("space")+HelpSepStyle.Render(" reveal sensitive fields"),
		)
	case data.EntryTypeSSHKey:
		parts = append(parts,
			HelpKeyStyle.Render("k") + HelpSepStyle.Render(" copy pub key"),
			HelpKeyStyle.Render("p") + HelpSepStyle.Render(" copy priv key"),
			HelpKeyStyle.Render("space") + HelpSepStyle.Render(" reveal priv key"),
		)
	}
	return HelpDescStyle.Render(strings.Join(parts, HelpDescStyle.Render("  •  ")))
}

func masked(s string) string {
	if s == "" {
		return DetailMaskedStyle.Render("—")
	}
	return DetailMaskedStyle.Render(strings.Repeat("•", min(len(s), 12)))
}

func truncateKey(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// formatCardNumber inserts a space every 4 digits: "4012888888881881" → "4012 8888 8888 1881"
func formatCardNumber(n string) string {
	// Strip existing spaces first
	clean := strings.ReplaceAll(n, " ", "")
	var b strings.Builder
	for i, ch := range clean {
		if i > 0 && i%4 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

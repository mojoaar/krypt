package data

import (
	"strings"
	"time"
)

// EntryType identifies what kind of secret an entry holds.
type EntryType string

const (
	EntryTypeLogin    EntryType = "login"
	EntryTypeNote     EntryType = "note"
	EntryTypeCard     EntryType = "card"
	EntryTypeIdentity EntryType = "identity"
	EntryTypeSSHKey   EntryType = "sshkey"
)

// AllEntryTypes lists every type in display order.
var AllEntryTypes = []EntryType{
	EntryTypeLogin,
	EntryTypeNote,
	EntryTypeCard,
	EntryTypeIdentity,
	EntryTypeSSHKey,
}

// EntryTypeLabel returns a human-readable label for the sidebar.
func EntryTypeLabel(t EntryType) string {
	switch t {
	case EntryTypeLogin:
		return "Login"
	case EntryTypeNote:
		return "Note"
	case EntryTypeCard:
		return "Card"
	case EntryTypeIdentity:
		return "Identity"
	case EntryTypeSSHKey:
		return "SSH Key"
	}
	return string(t)
}

// EntryTypeBadge returns the readable label shown in the list TYPE column.
// All labels are 5 characters wide so columns align consistently.
func EntryTypeBadge(t EntryType) string {
	switch t {
	case EntryTypeLogin:
		return "Login"
	case EntryTypeNote:
		return "Note "
	case EntryTypeCard:
		return "Card "
	case EntryTypeIdentity:
		return "Ident"
	case EntryTypeSSHKey:
		return "SSH  "
	}
	return "?????"
}

// Entry holds all data for a single vault item. Type-specific fields are
// included inline; unused fields are empty strings.
type Entry struct {
	ID        string    `json:"id"`
	Type      EntryType `json:"type"`
	Name      string    `json:"name"`
	Tags      []string  `json:"tags"`
	Favorite  bool      `json:"favorite"`
	UpdatedAt time.Time `json:"updated_at"`

	// Login
	URL      string `json:"url,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Notes    string `json:"notes,omitempty"`

	// Note
	Content string `json:"content,omitempty"`
	Secure  bool   `json:"secure,omitempty"`

	// Card
	CardHolder string `json:"card_holder,omitempty"`
	CardNumber string `json:"card_number,omitempty"`
	Expiry     string `json:"expiry,omitempty"`
	CVV        string `json:"cvc,omitempty"`
	Bank       string `json:"bank,omitempty"`

	// Identity
	FirstName      string `json:"first_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
	Email          string `json:"email,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Address        string `json:"address,omitempty"`
	Company        string `json:"company,omitempty"`
	SSN            string `json:"ssn,omitempty"`
	DriversLicense string `json:"drivers_license,omitempty"`
	PassportNumber string `json:"passport_number,omitempty"`
	IdentityNotes       string `json:"identity_notes,omitempty"`
	IdentityNotesSecure bool   `json:"identity_notes_secure,omitempty"`

	// SSH Key
	PublicKey  string `json:"public_key,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Host       string `json:"host,omitempty"`
}

// DetailLine returns the contextual value shown in the list's detail column.
func (e Entry) DetailLine() string {
	switch e.Type {
	case EntryTypeLogin:
		return e.Username
	case EntryTypeNote:
		if e.Secure {
			return "••••••••••"
		}
		// Show first non-empty line only — content may contain newlines from textarea
		first := e.Content
		if i := strings.IndexByte(first, '\n'); i >= 0 {
			first = strings.TrimSpace(first[:i])
			if first == "" {
				// first line was blank, try next
				rest := strings.TrimSpace(e.Content[i+1:])
				if j := strings.IndexByte(rest, '\n'); j >= 0 {
					rest = rest[:j]
				}
				first = rest
			}
		}
		if len([]rune(first)) > 40 {
			return string([]rune(first)[:40]) + "…"
		}
		return first
	case EntryTypeCard:
		if len(e.CardNumber) >= 4 {
			return "•••• " + e.CardNumber[len(e.CardNumber)-4:]
		}
		return e.CardHolder
	case EntryTypeIdentity:
		full := e.FirstName + " " + e.LastName
		if full == " " {
			return e.Email
		}
		return full
	case EntryTypeSSHKey:
		return e.Host
	}
	return ""
}

// HasTag reports whether the entry carries the given tag.
func (e Entry) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// VaultData is the plaintext payload that gets encrypted on disk.
type VaultData struct {
	Entries []Entry `json:"entries"`
}

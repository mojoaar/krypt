// Package cli implements the krypt command-line interface for non-interactive
// secret retrieval. 2FA is intentionally skipped — the vault is protected by
// AES-256-GCM; the CLI requires the master password which is the primary key.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/mojoaar/krypt/internal/data"
	"golang.org/x/term"
)

// Run dispatches the CLI subcommand. Returns true if a CLI command was handled
// (so main.go knows not to launch the TUI).
func Run(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "get":
		runGet(args[1:])
		return true
	case "list":
		runList(args[1:])
		return true
	case "help", "--help", "-h":
		printUsage()
		return true
	}
	return false
}

// ── get ──────────────────────────────────────────────────────────────────────

func runGet(args []string) {
	// krypt get <name> <field> [--copy]
	if len(args) < 2 {
		fatalf("usage: krypt get <name> <field> [--copy]\n\nFields by type:\n%s", fieldHelp())
	}
	name := args[0]
	field := strings.ToLower(args[1])
	doCopy := false
	for _, a := range args[2:] {
		if a == "--copy" || a == "-c" {
			doCopy = true
		}
	}

	store := mustLoad()
	entry := findEntry(store, name)

	value := fieldValue(entry, field)
	if value == "" {
		fatalf("field %q is empty or not available for entry type %q", field, entry.Type)
	}

	if doCopy {
		if err := clipboard.WriteAll(value); err != nil {
			fatalf("clipboard write failed: %v", err)
		}
		fmt.Fprintf(os.Stderr, "copied %s for %q\n", field, entry.Name)
	} else {
		fmt.Println(value)
	}
}

// ── list ─────────────────────────────────────────────────────────────────────

func runList(args []string) {
	// krypt list [--type=<type>]
	typeFilter := ""
	for _, a := range args {
		if strings.HasPrefix(a, "--type=") {
			typeFilter = strings.ToLower(strings.TrimPrefix(a, "--type="))
		}
	}

	store := mustLoad()
	entries := store.Entries()

	for _, e := range entries {
		t := string(e.Type)
		if typeFilter != "" && t != typeFilter {
			continue
		}
		fmt.Printf("[%s] %s\n", t, e.Name)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustLoad() *data.Store {
	pw := os.Getenv("KRYPT_MASTER_PASSWORD")
	if pw == "" {
		pw = promptPassword()
	}
	store, err := data.Load(pw)
	if err != nil {
		fatalf("could not unlock vault: %v", err)
	}
	return store
}

func promptPassword() string {
	fmt.Fprint(os.Stderr, "master password: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		fatalf("could not read password: %v", err)
	}
	return string(b)
}

func findEntry(store *data.Store, name string) data.Entry {
	lower := strings.ToLower(name)
	for _, e := range store.Entries() {
		if strings.ToLower(e.Name) == lower {
			return e
		}
	}
	// fuzzy: contains match
	for _, e := range store.Entries() {
		if strings.Contains(strings.ToLower(e.Name), lower) {
			return e
		}
	}
	fatalf("no entry found matching %q", name)
	return data.Entry{}
}

func fieldValue(e data.Entry, field string) string {
	switch field {
	// ── Login ──────────────────────────────────────────────────────────────
	case "password", "pass", "pw":
		return e.Password
	case "username", "user", "u":
		return e.Username
	case "url":
		return e.URL
	case "notes", "note":
		if e.Type == data.EntryTypeLogin {
			return e.Notes
		}
		if e.Type == data.EntryTypeCard {
			return e.CardNotes
		}
		// Note entry
		return e.Content
	case "content":
		return e.Content
	// ── Card ───────────────────────────────────────────────────────────────
	case "number", "cardnumber", "card":
		return e.CardNumber
	case "expiry", "exp":
		return e.Expiry
	case "cvc", "cvv":
		return e.CVV
	case "pin":
		return e.PIN
	case "holder", "cardholder":
		return e.CardHolder
	case "bank":
		return e.Bank
	case "cardnotes":
		return e.CardNotes
	// ── Identity ───────────────────────────────────────────────────────────
	case "email":
		return e.Email
	case "phone":
		return e.Phone
	case "address":
		return e.Address
	case "company":
		return e.Company
	case "ssn":
		return e.SSN
	case "license", "drivers", "dl":
		return e.DriversLicense
	case "passport":
		return e.PassportNumber
	case "firstname", "first":
		return e.FirstName
	case "lastname", "last":
		return e.LastName
	// ── SSH ────────────────────────────────────────────────────────────────
	case "pubkey", "public", "pub":
		return e.PublicKey
	case "privkey", "private", "priv":
		return e.PrivateKey
	case "passphrase":
		return e.Passphrase
	case "host":
		return e.Host
	}
	fatalf("unknown field %q — run 'krypt help' to see available fields", field)
	return ""
}

func fieldHelp() string {
	return `  login    : password  username  url  notes
  note     : content
  card     : number  expiry  cvc  holder  bank
  identity : email  phone  address  company  ssn  license  passport  firstname  lastname
  ssh      : pubkey  privkey  passphrase  host`
}

func printUsage() {
	fmt.Print(`krypt — terminal password manager

Usage:
  krypt                              launch TUI
  krypt get <name> <field> [--copy]  retrieve a secret
  krypt list [--type=<type>]         list entries
  krypt --version                    print version

Fields by entry type:
` + fieldHelp() + `

Master password:
  Set KRYPT_MASTER_PASSWORD env var for non-interactive use,
  or krypt will prompt securely (no echo).

Note:
  2FA is skipped for CLI commands. The vault is still AES-256-GCM
  encrypted and requires the master password.

Examples:
  krypt get "iCloud" password
  krypt get "iCloud" password --copy
  krypt get "GitHub SSH" pubkey
  KRYPT_MASTER_PASSWORD=xxx krypt list --type=login
`)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "krypt: "+format+"\n", args...)
	os.Exit(1)
}

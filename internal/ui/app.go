package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mojoaar/krypt/internal/data"
)

// ── Modes ──────────────────────────────────────────────────────────────────

type appMode int

const (
	modeUnlock  appMode = iota // master password (+ optional 2FA) prompt
	modeNav                    // browse sidebar + list
	modeDetail                 // full entry detail
	modeForm                   // add/edit overlay
	modeConfirm                // delete confirmation
	modeSearch                 // live search input
	modeHelp                   // help overlay
	modeSync                   // sync in progress
	mode2FA                    // 2FA setup / disable
	modeExport                 // export overlay
	modeMenu                   // actions menu overlay
)

type focusTarget int

const (
	focusSidebar focusTarget = iota
	focusList
)

const bannerLines = 8 // 1 padding + 6 art lines + 1 padding

// ── Tea commands / messages ─────────────────────────────────────────────────

type clearStatusMsg struct{}
type syncDoneMsg struct{ err error }
type statusMsg string

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// ── App ─────────────────────────────────────────────────────────────────────

// App is the root Bubble Tea model.
type App struct {
	version string
	mode    appMode
	focus   focusTarget

	// sub-models
	unlock     UnlockModel
	sidebar    Sidebar
	list       ListView
	detail     DetailView
	form       Form
	confirm    Confirm
	export     ExportModel
	search     textinput.Model
	helpVP     viewport.Model
	twoFA      TwoFAModel

	// vault (nil until unlocked)
	store          *data.Store
	masterPassword string

	// sync config
	syncCfg *data.Config

	// display
	width     int
	height    int
	statusMsg string
	statusErr bool
}

// NewApp constructs the initial app in unlock mode.
func NewApp(version string) App {
	has2FA, _ := data.TwoFAEnabled()
	vaultExists, _ := data.VaultExists()

	sb := NewSidebar()
	lv := NewListView()
	lv.SetFocused(true) // initial focus is on the list
	dv := NewDetailView()
	si := textinput.New()
	si.Placeholder = "search…"
	si.CharLimit = 128
	si.TextStyle = lipgloss.NewStyle().Background(colorSelected).Foreground(colorText)
	si.PlaceholderStyle = lipgloss.NewStyle().Background(colorSelected).Foreground(colorMuted)
	si.PromptStyle = lipgloss.NewStyle().Background(colorSelected).Foreground(colorAccent)
	si.Cursor.Style = lipgloss.NewStyle().Background(colorAccent).Foreground(colorSelected)

	syncCfg, _ := data.LoadConfig()
	if syncCfg == nil {
		syncCfg = &data.Config{}
	}

	return App{
		version: version,
		mode:    modeUnlock,
		unlock:  NewUnlockModel(has2FA, !vaultExists),
		sidebar: sb,
		list:    lv,
		detail:  dv,
		form:    NewForm(),
		confirm: NewConfirm(),
		export:  NewExportModel(),
		search:  si,
		syncCfg: syncCfg,
		focus:   focusList,
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func (a App) Init() tea.Cmd { return textinput.Blink }

// ── Update ───────────────────────────────────────────────────────────────────

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global exit — works in every mode including unlock
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
	case tea.QuitMsg:
		return a, tea.Quit
	}

	switch msg := msg.(type) {

	// ── Window resize ───────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		listW := a.width - sidebarWidth - 2
		listH := a.height - bannerLines - 2
		a.list.SetSize(listW, listH)
		a.detail.SetSize(listW, listH)
		a.form.SetSize(a.width, a.height)
		a.confirm.SetSize(a.width, a.height)
		a.export.SetSize(a.width, a.height)
		a.unlock.width = a.width
		a.unlock.height = a.height
		// Resize help viewport if it's open
		if a.mode == modeHelp {
			a.helpVP.Width = helpVPWidth(a.width)
			a.helpVP.Height = helpVPHeight(a.height)
		}
		return a, nil

	// ── Status clear ────────────────────────────────────────────────────────
	case clearStatusMsg:
		a.statusMsg = ""
		a.statusErr = false
		return a, nil

	// ── Unlock flow ─────────────────────────────────────────────────────────
	case UnlockDoneMsg:
		return a.handleUnlockDone(msg)

	case unlockVerify2FAMsg:
		return a.handleVerify2FA(msg)

	case UnlockResetMsg:
		_ = data.ResetAll()
		return a, tea.Quit

	// ── Form events ─────────────────────────────────────────────────────────
	case FormSubmitMsg:
		a.mode = modeNav
		e := msg.Entry
		e.UpdatedAt = time.Now()
		if e.ID == "" {
			a.store.Add(e)
		} else {
			a.store.Update(e)
		}
		if err := a.store.Save(a.masterPassword); err != nil {
			a.setStatus("save failed: "+err.Error(), true)
		} else {
			a.setStatus("saved ✓", false)
		}
		a.rebuildList()
		return a, clearStatusAfter(2 * time.Second)

	case FormCancelMsg:
		a.mode = modeNav
		return a, nil

	// ── Confirm events ───────────────────────────────────────────────────────
	case ConfirmYesMsg:
		a.store.Delete(msg.ID)
		if err := a.store.Save(a.masterPassword); err != nil {
			a.setStatus("delete failed: "+err.Error(), true)
		} else {
			a.setStatus("deleted ✓", false)
		}
		a.mode = modeNav
		a.rebuildList()
		return a, clearStatusAfter(2 * time.Second)

	case ConfirmNoMsg:
		a.mode = modeNav
		return a, nil

	// ── Sync done ────────────────────────────────────────────────────────────
	case syncDoneMsg:
		a.mode = modeNav
		if msg.err != nil {
			a.setStatus("sync failed: "+msg.err.Error(), true)
		} else {
			a.setStatus("synced ✓", false)
		}
		return a, clearStatusAfter(3 * time.Second)

	case TwoFADoneMsg:
		a.mode = modeNav
		if msg.Err != nil {
			a.setStatus("2FA error: "+msg.Err.Error(), true)
		} else if msg.Enabled {
			a.setStatus("2FA enabled ✓ — you will need your authenticator on next unlock", false)
		} else {
			a.setStatus("2FA disabled — vault opens with master password only", false)
		}
		return a, clearStatusAfter(5 * time.Second)

	// ── Export events ────────────────────────────────────────────────────────
	case exportWriteMsg:
		entries := []data.Entry{}
		if a.store != nil {
			entries = a.store.Entries()
		}
		return a, DoExport(msg, entries)

	case ExportDoneMsg:
		a.mode = modeNav
		a.setStatus("exported to "+msg.Path, false)
		return a, clearStatusAfter(5 * time.Second)

	case exportErrorMsg:
		a.mode = modeNav
		a.setStatus("export failed: "+string(msg), true)
		return a, clearStatusAfter(5 * time.Second)

	case ExportCancelMsg:
		a.mode = modeNav
		return a, nil
	}

	// ── Per-mode routing ─────────────────────────────────────────────────────
	switch a.mode {
	case modeUnlock:
		return a.updateUnlock(msg)
	case modeForm:
		return a.updateForm(msg)
	case modeConfirm:
		return a.updateConfirm(msg)
	case mode2FA:
		return a.update2FA(msg)
	case modeExport:
		return a.updateExport(msg)
	case modeMenu:
		return a.updateMenu(msg)
	case modeNav:
		return a.updateNav(msg)
	case modeDetail:
		return a.updateDetail(msg)
	case modeSearch:
		return a.updateSearch(msg)
	case modeHelp:
		return a.updateHelp(msg)
	}
	return a, nil
}

// ── Unlock handlers ──────────────────────────────────────────────────────────

func (a App) handleUnlockDone(msg UnlockDoneMsg) (tea.Model, tea.Cmd) {
	store, err := data.Load(msg.MasterPassword)
	if err != nil {
		n := data.IncrementAttempts()
		remaining := data.MaxAttempts() - n
		if remaining <= 0 {
			data.DestroyVault()
			a.unlock.SetError("too many failed attempts — vault destroyed")
			return a, tea.Quit
		}
		a.unlock.SetError(fmt.Sprintf("wrong password — %d attempt(s) remaining", remaining))
		return a, nil
	}
	data.ResetAttempts()
	a.store = store
	a.masterPassword = msg.MasterPassword
	a.mode = modeNav
	a.rebuildList()
	if msg.IsNewVault {
		// Write the empty vault to disk immediately so the master password is
		// persisted and subsequent opens don't treat it as a new vault.
		if err := a.store.Save(a.masterPassword); err != nil {
			a.setStatus("could not create vault: "+err.Error(), true)
		} else {
			a.setStatus("new vault created — your data is encrypted  (~/.config/krypt/vault.enc)", false)
		}
		return a, clearStatusAfter(5 * time.Second)
	}
	return a, nil
}

func (a App) handleVerify2FA(msg unlockVerify2FAMsg) (tea.Model, tea.Cmd) {
	secret, err := data.LoadTwoFASecret(msg.password)
	if err != nil {
		n := data.IncrementAttempts()
		remaining := data.MaxAttempts() - n
		if remaining <= 0 {
			data.DestroyVault()
			a.unlock.SetError("too many failed attempts — vault destroyed")
			return a, tea.Quit
		}
		a.unlock.SetError(fmt.Sprintf("wrong password — %d attempt(s) remaining", remaining))
		return a, nil
	}
	code, _, err := data.GenerateTOTP(secret)
	if err != nil || code != msg.code {
		n := data.IncrementAttempts()
		remaining := data.MaxAttempts() - n
		if remaining <= 0 {
			data.DestroyVault()
			a.unlock.SetError("too many failed attempts — vault destroyed")
			return a, tea.Quit
		}
		a.unlock.SetError(fmt.Sprintf("invalid 2FA code — %d attempt(s) remaining", remaining))
		return a, nil
	}
	return a.handleUnlockDone(UnlockDoneMsg{MasterPassword: msg.password, IsNewVault: msg.isNewVault})
}

// ── Mode updates ─────────────────────────────────────────────────────────────

func (a App) updateUnlock(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.unlock, cmd = a.unlock.Update(msg)
	return a, cmd
}

func (a App) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.form, cmd = a.form.Update(msg)
	return a, cmd
}

func (a App) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.confirm, cmd = a.confirm.Update(msg)
	return a, cmd
}

func (a App) update2FA(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.twoFA, cmd = a.twoFA.Update(msg)
	return a, cmd
}

func (a App) updateExport(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.export, cmd = a.export.Update(msg)
	return a, cmd
}

func (a App) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch k.String() {
	case "esc", "m", "q":
		a.mode = modeNav
	case "g":
		a.mode = modeNav
		pw := generatePassword()
		if err := clipboard.WriteAll(pw); err != nil {
			a.setStatus("generate failed: "+err.Error(), true)
		} else {
			a.setStatus("strong password generated and copied", false)
		}
		return a, clearStatusAfter(5 * time.Second)
	case "t":
		a.mode = modeNav
		return a.open2FASetup()
	case "x":
		a.mode = modeNav
		a.export.Open()
		a.mode = modeExport
	}
	return a, nil
}

func (a App) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "?", "esc", "q":
			a.mode = modeNav
			return a, nil
		}
	}
	var cmd tea.Cmd
	a.helpVP, cmd = a.helpVP.Update(msg)
	return a, cmd
}

func (a App) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			a.mode = modeNav
			a.search.Blur()
			return a, nil
		}
	}
	var cmd tea.Cmd
	a.search, cmd = a.search.Update(msg)
	a.rebuildList()
	return a, cmd
}

func (a App) updateNav(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always keep visual focus in sync with logical focus
	a.list.SetFocused(a.focus == focusList)
	a.sidebar.SetFocused(a.focus == focusSidebar)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "?":
			a.mode = modeHelp
			w := helpVPWidth(a.width)
			h := helpVPHeight(a.height)
			a.helpVP = viewport.New(w, h)
			a.helpVP.SetContent(a.buildHelpContent(w))
			return a, nil
		case "/":
			a.mode = modeSearch
			a.search.Focus()
			return a, textinput.Blink
		case "tab":
			if a.focus == focusSidebar {
				a.focus = focusList
			} else {
				a.focus = focusSidebar
			}
			a.sidebar.SetFocused(a.focus == focusSidebar)
			a.list.SetFocused(a.focus == focusList)
			return a, nil
		case "j", "down":
			if a.focus == focusSidebar {
				a.sidebar.MoveDown()
				a.rebuildList()
			} else {
				a.list.MoveDown()
			}
		case "k", "up":
			if a.focus == focusSidebar {
				a.sidebar.MoveUp()
				a.rebuildList()
			} else {
				a.list.MoveUp()
			}
		case "enter":
			if a.focus == focusList {
				if e := a.list.Selected(); e != nil {
					a.detail.SetEntry(*e)
					a.mode = modeDetail
				}
			}
		case "a":
			a.form.OpenNew()
			a.mode = modeForm
		case "e":
			if e := a.list.Selected(); e != nil {
				a.form.OpenEdit(*e)
				a.mode = modeForm
			}
		case "d":
			if e := a.list.Selected(); e != nil {
				a.confirm.Open(e.ID, e.Name)
				a.mode = modeConfirm
			}
		case "u", "p":
			return a.handleCopyNav(msg.String())
		case "s":
			return a.handleSync()
		case "t":
			return a.open2FASetup()
		case "g":
			pw := generatePassword()
			if err := clipboard.WriteAll(pw); err != nil {
				a.setStatus("generate failed: "+err.Error(), true)
			} else {
				a.setStatus("strong password generated and copied", false)
			}
			return a, clearStatusAfter(5 * time.Second)
		case "x":
			a.export.Open()
			a.mode = modeExport
			return a, nil
		case "m":
			a.mode = modeMenu
			return a, nil
		}
	}
	return a, nil
}

func (a App) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			a.mode = modeNav
		case "?":
			a.mode = modeHelp
			w := helpVPWidth(a.width)
			h := helpVPHeight(a.height)
			a.helpVP = viewport.New(w, h)
			a.helpVP.SetContent(a.buildHelpContent(w))
			return a, nil
		case "e":
			if e := a.list.Selected(); e != nil {
				a.form.OpenEdit(*e)
				a.mode = modeForm
			}
		case "d":
			if e := a.list.Selected(); e != nil {
				a.confirm.Open(e.ID, e.Name)
				a.mode = modeConfirm
			}
		case " ":
			entry := a.detail.entry
			switch entry.Type {
			case data.EntryTypeLogin:
				a.detail.TogglePassword()
			case data.EntryTypeNote:
				if entry.Secure {
					a.detail.ToggleContent()
				}
			case data.EntryTypeCard:
				a.detail.ToggleCardNumber()
				a.detail.ToggleCVC()
			case data.EntryTypeIdentity:
				a.detail.ToggleIdentity()
			case data.EntryTypeSSHKey:
				a.detail.TogglePrivKey()
			}
		case "u":
			return a.handleCopyDetail("u")
		case "p":
			return a.handleCopyDetail("p")
		case "n":
			return a.handleCopyDetail("n")
		case "k":
			return a.handleCopyDetail("k")
		case "c":
			return a.handleCopyDetail("c")
		case "x":
			return a.handleCopyDetail("x")
		case "s":
			return a.handleCopyDetail("s")
		case "l":
			return a.handleCopyDetail("l")
		case "b":
			return a.handleCopyDetail("b")
		case "o":
			return a.handleCopyDetail("o")
		case "i":
			return a.handleCopyDetail("i")
		}
	}
	return a, nil
}

// ── Copy helpers ─────────────────────────────────────────────────────────────

func (a App) handleCopyNav(key string) (tea.Model, tea.Cmd) {
	e := a.list.Selected()
	if e == nil {
		return a, nil
	}
	return a.copyField(*e, key)
}

func (a App) handleCopyDetail(key string) (tea.Model, tea.Cmd) {
	return a.copyField(a.detail.entry, key)
}

func (a App) copyField(e data.Entry, key string) (tea.Model, tea.Cmd) {
	var value, label string
	switch e.Type {
	case data.EntryTypeLogin:
		switch key {
		case "u":
			value, label = e.Username, "username"
		case "p":
			value, label = e.Password, "password"
		}
	case data.EntryTypeNote:
		if key == "c" {
			value, label = e.Content, "note content"
		}
	case data.EntryTypeIdentity:
		switch key {
		case "u":
			value, label = e.Email, "email"
		case "s":
			value, label = e.SSN, "ssn"
		case "l":
			value, label = e.DriversLicense, "drivers license"
		case "b":
			value, label = e.PassportNumber, "passport number"
		case "o":
			value, label = e.IdentityNotes, "notes"
		}
	case data.EntryTypeCard:
		switch key {
		case "n":
			value, label = e.CardNumber, "card number"
		case "x":
			value, label = e.Expiry, "expiry"
		case "c":
			value, label = e.CVV, "cvc"
		case "i":
			value, label = e.PIN, "pin"
		}
	case data.EntryTypeSSHKey:
		switch key {
		case "k":
			value, label = e.PublicKey, "public key"
		case "p":
			value, label = e.PrivateKey, "private key"
		}
	}
	if value == "" {
		return a, nil
	}
	if err := clipboard.WriteAll(value); err != nil {
		a.setStatus("clipboard error: "+err.Error(), true)
	} else {
		a.setStatus(fmt.Sprintf("copied %s ✓", label), false)
	}
	return a, clearStatusAfter(2 * time.Second)
}

// ── Sync ─────────────────────────────────────────────────────────────────────

func (a App) handleSync() (tea.Model, tea.Cmd) {
	tokenInEnv := os.Getenv("KRYPT_GITHUB_TOKEN") != ""
	if !a.syncCfg.SyncEnabled && !tokenInEnv && a.syncCfg.Token == "" {
		a.setStatus("sync not configured — set KRYPT_GITHUB_TOKEN to enable", true)
		return a, clearStatusAfter(4 * time.Second)
	}
	a.mode = modeSync
	a.setStatus("syncing…", false)
	cfg := a.syncCfg
	mp := a.masterPassword
	store := a.store
	return a, func() tea.Msg {
		// Push current vault
		blob, err := store.EncryptedBlob(mp)
		if err != nil {
			return syncDoneMsg{err: err}
		}
		if err := data.SyncPush(cfg, blob, mp); err != nil {
			return syncDoneMsg{err: err}
		}
		cfg.SyncEnabled = true
		_ = data.SaveConfig(cfg)
		return syncDoneMsg{}
	}
}

func (a App) open2FASetup() (tea.Model, tea.Cmd) {
	has2FA, _ := data.TwoFAEnabled()
	m, err := NewTwoFAModel(a.masterPassword, has2FA)
	if err != nil {
		a.setStatus("failed to generate 2FA secret: "+err.Error(), true)
		return a, clearStatusAfter(4 * time.Second)
	}
	m.width = a.width
	m.height = a.height
	a.twoFA = m
	a.mode = mode2FA
	return a, a.twoFA.Init()
}

// ── List rebuild ─────────────────────────────────────────────────────────────

func (a *App) rebuildList() {
	if a.store == nil {
		return
	}
	entries := a.store.Entries()
	a.sidebar.Rebuild(entries)
	typeF, tagF := a.sidebar.ActiveFilter()
	filtered := FilterAndSort(entries, typeF, tagF, a.search.Value())
	a.list.SetEntries(filtered)
	a.list.SetFocused(a.focus == focusList)
	a.sidebar.SetFocused(a.focus == focusSidebar)
}

func (a *App) setStatus(msg string, isErr bool) {
	a.statusMsg = msg
	a.statusErr = isErr
}

// ── View ─────────────────────────────────────────────────────────────────────

func (a App) View() string {
	switch a.mode {
	case modeUnlock:
		return a.unlock.View()
	case modeForm:
		return a.form.View()
	case modeConfirm:
		return a.confirm.View()
	case modeHelp:
		return a.viewHelp()
	case mode2FA:
		return a.twoFA.View()
	case modeExport:
		return a.export.View()
	case modeMenu:
		return a.viewMenu()
	}
	return a.viewMain()
}

func (a App) viewMain() string {
	if a.width < 80 || a.height < 24 {
		msg := "↕ Terminal too small — resize to continue"
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#B49FD8")).Render(msg))
	}
	banner := a.viewBanner()
	sidebar := a.sidebar.View(a.height - bannerLines - 2)

	var content string
	switch a.mode {
	case modeDetail:
		content = a.detail.View()
	default:
		content = a.list.View()
	}

	middle := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	helpBar := a.viewHelpBar()
	statusBar := a.viewStatusBar()

	if a.mode == modeSearch {
		searchBar := lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(colorAccent).
			Width(a.width).
			Padding(0, 1).
			Render(SearchPromptStyle.Render("/") + " " + a.search.View())
		return lipgloss.JoinVertical(lipgloss.Left, banner, middle, searchBar, statusBar, helpBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, banner, middle, statusBar, helpBar)
}

func (a App) viewMenu() string {
	type item struct{ key, desc string }
	items := []item{
		{"g", "generate password"},
		{"t", "2FA setup / disable"},
		{"x", "export vault"},
	}

	title := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("⚡ Actions")

	var rows []string
	rows = append(rows, title, "")
	for _, it := range items {
		k := HelpKeyStyle.Width(4).Render(it.key)
		d := HelpDescStyle.Render(it.desc)
		rows = append(rows, "  "+k+d)
	}
	rows = append(rows, "")
	rows = append(rows, HelpDescStyle.Render("esc  close"))

	inner := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Render(inner)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

func (a App) viewBanner() string {
	art := []string{
		`    __                    __ `,
		`   / /_________  ______  / /_`,
		`  / //_/ ___/ / / / __ \/ __/`,
		` / ,< / /  / /_/ / /_/ / /_  `,
		`/_/|_/_/   \__, / .___/\__/  `,
		`          /____/_/           `,
	}

	artWidth := 30
	leftPad := (a.width - artWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	pad := strings.Repeat(" ", leftPad)

	hint := HelpDescStyle.Render("[?] help  [q] quit")
	hintPad := a.width - lipgloss.Width(hint) - 1
	if hintPad < 0 {
		hintPad = 0
	}

	var rows []string
	rows = append(rows, strings.Repeat(" ", hintPad)+hint)
	for _, line := range art {
		rows = append(rows, pad+TitleStyle.Render(line))
	}
	rows = append(rows, "")
	return strings.Join(rows, "\n")
}

func (a App) viewStatusBar() string {
	if a.statusMsg == "" {
		return ""
	}
	if a.statusErr {
		return StatusErrStyle.Render("✗ " + a.statusMsg)
	}
	return StatusOkStyle.Render("✓ " + a.statusMsg)
}

func (a App) viewHelpBar() string {
	type hint struct{ key, desc string }
	var hints []hint

	switch a.mode {
	case modeNav:
		hints = []hint{
			{"j/k", "navigate"},
			{"tab", "switch panel"},
			{"a", "add"},
			{"e", "edit"},
			{"d", "delete"},
			{"enter", "open"},
			{"/", "search"},
			{"s", "sync"},
			{"m", "menu"},
		}
	case modeDetail:
		hints = []hint{
			{"esc", "back"},
			{"e", "edit"},
			{"d", "delete"},
		}
		switch a.detail.entry.Type {
		case data.EntryTypeNote:
			hints = append(hints, hint{"c", "copy content"})
			if a.detail.entry.Secure {
				hints = append(hints, hint{"space", "reveal"})
			}
		case data.EntryTypeLogin:
			hints = append(hints, hint{"u", "copy user"}, hint{"p", "copy pass"}, hint{"space", "reveal"})
		case data.EntryTypeCard:
			hints = append(hints,
				hint{"n", "copy number"},
				hint{"x", "copy expiry"},
				hint{"c", "copy cvc"},
				hint{"space", "reveal all"},
			)
		case data.EntryTypeIdentity:
			hints = append(hints,
				hint{"u", "copy email"},
				hint{"s", "copy ssn"},
				hint{"l", "copy license"},
				hint{"b", "copy passport"},
				hint{"o", "copy notes"},
				hint{"space", "reveal"},
			)
		case data.EntryTypeSSHKey:
			hints = append(hints, hint{"k", "copy pub"}, hint{"p", "copy priv"}, hint{"space", "reveal"})
		}
	case modeSearch:
		hints = []hint{{"enter/esc", "done"}, {"type", "filter"}}
	case modeSync:
		hints = []hint{{"syncing…", ""}}
	}

	parts := make([]string, len(hints))
	for i, h := range hints {
		if h.desc == "" {
			parts[i] = HelpKeyStyle.Render(h.key)
		} else {
			parts[i] = HelpKeyStyle.Render(h.key) + HelpSepStyle.Render(" ") + HelpDescStyle.Render(h.desc)
		}
	}

	bar := strings.Join(parts, HelpSepStyle.Render("  ·  "))
	return StatusBarStyle.Render(bar)
}

func helpVPWidth(termW int) int {
	w := termW - 12 // box border + padding
	if w > 80 {
		w = 80
	}
	if w < 40 {
		w = 40
	}
	return w
}

func helpVPHeight(termH int) int {
	h := termH - 8
	if h < 10 {
		h = 10
	}
	return h
}

// buildHelpContent renders the full help text at the given width.
func (a App) buildHelpContent(w int) string {
	keybindings := []struct {
		title string
		rows  [][2]string
	}{
		{"Navigation", [][2]string{
			{"j / k  or  ↑ / ↓", "move up / down"},
			{"tab", "switch focus: sidebar ↔ list"},
			{"enter", "open entry detail"},
			{"esc", "go back"},
		}},
		{"Entries", [][2]string{
			{"a", "add new entry"},
			{"e", "edit selected"},
			{"d", "delete selected"},
			{"/", "search by name, detail or tag"},
		}},
		{"Copy (in list or detail)", [][2]string{
			{"c", "copy note content / card cvc"},
			{"u", "copy username / email"},
			{"p", "copy password / private key"},
			{"n", "copy card number"},
			{"x", "copy card expiry"},
			{"k", "copy SSH public key"},
			{"s", "copy SSN (identity)"},
			{"l", "copy drivers license (identity)"},
			{"b", "copy passport number (identity)"},
			{"o", "copy identity notes"},
			{"space", "reveal card number + cvc / password / SSH private key / identity sensitive fields"},
		}},
		{"Login URLs", [][2]string{
			{"", "URLs are shown as clickable hyperlinks"},
			{"", "cmd+click (macOS) or ctrl+click to open in browser"},
			{"", "supported in: iTerm2, WezTerm, kitty, Ghostty"},
			{"", "falls back to plain text in unsupported terminals"},
		}},
		{"App", [][2]string{
			{"m", "open actions menu (generate pw, 2FA, export)"},
			{"g", "generate password and copy to clipboard"},
			{"t", "2FA setup / disable"},
			{"x", "export vault (plaintext or encrypted JSON)"},
			{"s", "sync vault to GitHub Gist"},
			{"?", "toggle this help"},
			{"q  or  ctrl+c", "quit"},
		}},
	}

	ver := a.version
	if ver == "dev" {
		ver = "dev build"
	}

	sep := HelpSepStyle.Render(strings.Repeat("─", w))
	note := func(s string) string { return HelpDescStyle.Render(s) }
	key := func(s string) string { return HelpKeyStyle.Render(s) }
	head := func(s string) string { return FormTitleStyle.Render(s) }

	var lines []string
	lines = append(lines, TitleStyle.Render(fmt.Sprintf("krypt %s  —  help", ver)))
	lines = append(lines, "")

	// ── Keybindings ──────────────────────────────────────────────────────────
	for _, sec := range keybindings {
		lines = append(lines, head(sec.title))
		for _, row := range sec.rows {
			k := HelpKeyStyle.Width(28).Render(row[0])
			d := HelpDescStyle.Render(row[1])
			lines = append(lines, "  "+k+d)
		}
		lines = append(lines, "")
	}

	lines = append(lines, note("press ? or esc to close  •  j/k or ↑/↓ to scroll"))
	lines = append(lines, "")
	lines = append(lines, sep)
	lines = append(lines, "")

	// ── GitHub Sync ──────────────────────────────────────────────────────────
	lines = append(lines, head("GitHub Sync  (optional)"))
	lines = append(lines, note("Your vault is pushed as an encrypted blob to a private GitHub Gist."))
	lines = append(lines, note("GitHub never sees plaintext — only the AES-256-GCM ciphertext."))
	lines = append(lines, "")
	lines = append(lines, note("  1. Create a GitHub token with ")+key("gist")+note(" scope:"))
	lines = append(lines, note("       ")+key("https://github.com/settings/tokens"))
	lines = append(lines, "")
	lines = append(lines, note("  2. Export the token in your shell profile:"))
	lines = append(lines, note("       ")+key("export KRYPT_GITHUB_TOKEN=ghp_..."))
	lines = append(lines, "")
	lines = append(lines, note("  3. Press ")+key("s")+note(" inside krypt to push. The Gist ID is saved"))
	lines = append(lines, note("     automatically to ")+key("~/.config/krypt/config.json")+note(" after the first push."))
	lines = append(lines, "")
	lines = append(lines, note("  4. On another machine: set the same token, then press ")+key("s")+note(" to pull"))
	lines = append(lines, note("     and decrypt using your master password."))
	lines = append(lines, "")
	lines = append(lines, sep)
	lines = append(lines, "")

	// ── 2FA ─────────────────────────────────────────────────────────────────
	lines = append(lines, head("Two-Factor Authentication  (optional)"))
	lines = append(lines, note("Adds a TOTP second factor to the krypt unlock screen itself."))
	lines = append(lines, note("Works with any RFC 6238 app: Google Authenticator, Authy, 1Password, etc."))
	lines = append(lines, "")
	lines = append(lines, note("How it works:"))
	lines = append(lines, note("  • Master password alone → vault opens  (2FA disabled)"))
	lines = append(lines, note("  • Master password + 6-digit code → vault opens  (2FA enabled)"))
	lines = append(lines, "")
	lines = append(lines, note("To enable 2FA, press ")+key("t")+note(" from the main screen."))
	lines = append(lines, note("krypt will generate a secret, show it for you to add to your authenticator,"))
	lines = append(lines, note("then ask for a verification code to confirm setup."))
	lines = append(lines, "")
	lines = append(lines, note("To disable 2FA, press ")+key("t")+note(" again. Confirm to remove the 2FA secret."))
	lines = append(lines, "")
	lines = append(lines, note("On next launch after enabling, krypt will prompt:"))
	lines = append(lines, note("  master password → 6-digit code from your app → vault opens."))
	lines = append(lines, "")
	lines = append(lines, sep)
	lines = append(lines, "")

	// ── About ────────────────────────────────────────────────────────────────
	verLabel := a.version
	if verLabel == "dev" {
		verLabel = "dev build"
	}
	lines = append(lines, note("Version ")+key(verLabel))
	if a.store != nil {
		lines = append(lines, note("Data    ")+key(a.store.Path()))
	}
	lines = append(lines, note("Author  ")+key("Morten Johansen")+HelpSepStyle.Render("  |  ")+note("johansen.foo"))
	lines = append(lines, note("Repo    ")+key("github.com/mojoaar/krypt"))

	return strings.Join(lines, "\n")
}

func (a App) viewHelp() string {
	// Scrollbar
	totalLines := strings.Count(a.helpVP.View(), "\n") + 1
	vpH := a.helpVP.Height
	scrollbar := renderScrollbar(a.helpVP.YOffset, a.helpVP.TotalLineCount(), vpH)

	vpRendered := lipgloss.JoinHorizontal(lipgloss.Top,
		a.helpVP.View(),
		scrollbar,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Render(vpRendered)

	hint := HelpDescStyle.Render("j/k scroll  •  esc close")
	content := lipgloss.JoinVertical(lipgloss.Left, box, hint)

	_ = totalLines
	if a.width > 0 && a.height > 0 {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

// renderScrollbar returns a simple single-column scrollbar string.
func renderScrollbar(offset, total, visible int) string {
	if total <= visible {
		return strings.Repeat(" \n", visible)
	}
	thumbH := max(1, visible*visible/total)
	thumbPos := (offset * (visible - thumbH)) / max(1, total-visible)

	var sb strings.Builder
	for i := 0; i < visible; i++ {
		if i >= thumbPos && i < thumbPos+thumbH {
			sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Render("┃"))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(colorBorder).Render("│"))
		}
		if i < visible-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}


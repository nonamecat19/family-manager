package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"connectrpc.com/connect"
	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"

	"github.com/nnc/family-manager/apps/tui/internal/session"
)

type NotesClient interface {
	ListNotes(context.Context, *connect.Request[notesv1.ListNotesRequest]) (*connect.Response[notesv1.ListNotesResponse], error)
	GetNote(context.Context, *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error)
}

type Auth interface {
	LoggedIn() (bool, error)
	StartDeviceLogin(context.Context) (session.Grant, error)
	Poll(context.Context, *session.Grant) (bool, error)
}

type screen int

const (
	screenStarting screen = iota
	screenLogin
	screenNotes
	screenNote
)

type (
	loggedInMsg struct {
		ok  bool
		err error
	}
	grantMsg struct {
		grant session.Grant
		err   error
	}
	pollMsg struct {
		grant session.Grant
		done  bool
		err   error
	}
	notesMsg struct {
		notes []*notesv1.Note
		err   error
	}
	noteMsg struct {
		note *notesv1.Note
		err  error
	}
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cursorStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
	codeStyle   = lipgloss.NewStyle().Bold(true).Padding(0, 2).Border(lipgloss.RoundedBorder())
	quoteStyle  = lipgloss.NewStyle().Italic(true)
	codeBlock   = lipgloss.NewStyle().Faint(true)
)

type Model struct {
	ctx    context.Context
	auth   Auth
	notes  NotesClient
	screen screen
	width  int
	height int
	err    error
	status string

	grant   session.Grant
	waiting bool

	list   []*notesv1.Note
	cursor int

	note   *notesv1.Note
	scroll int
}

func New(ctx context.Context, auth Auth, notes NotesClient) Model {
	return Model{ctx: ctx, auth: auth, notes: notes, screen: screenStarting, width: 80, height: 24}
}

func Run(ctx context.Context, auth Auth, notes NotesClient) error {
	_, err := tea.NewProgram(New(ctx, auth, notes), tea.WithContext(ctx)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("terminal ui: %w", err)
	}
	return nil
}

func (m Model) Init() tea.Cmd {
	return m.checkLoggedIn
}

func (m Model) checkLoggedIn() tea.Msg {
	ok, err := m.auth.LoggedIn()
	return loggedInMsg{ok: ok, err: err}
}

func (m Model) startLogin() tea.Msg {
	g, err := m.auth.StartDeviceLogin(m.ctx)
	return grantMsg{grant: g, err: err}
}

func (m Model) poll(g session.Grant) tea.Cmd {
	return func() tea.Msg {
		if err := session.Sleep(m.ctx, g.Interval); err != nil {
			return pollMsg{grant: g, err: err}
		}
		done, err := m.auth.Poll(m.ctx, &g)
		return pollMsg{grant: g, done: done, err: err}
	}
}

func (m Model) loadNotes() tea.Msg {
	resp, err := m.notes.ListNotes(m.ctx, connect.NewRequest(&notesv1.ListNotesRequest{Sort: notesv1.NoteSort_NOTE_SORT_UPDATED}))
	if err != nil {
		return notesMsg{err: fmt.Errorf("list notes: %w", err)}
	}
	return notesMsg{notes: resp.Msg.GetNotes()}
}

func (m Model) loadNote(id string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.notes.GetNote(m.ctx, connect.NewRequest(&notesv1.GetNoteRequest{NoteId: id}))
		if err != nil {
			return noteMsg{err: fmt.Errorf("open note: %w", err)}
		}
		return noteMsg{note: resp.Msg.GetNote()}
	}
}

func (m Model) toLogin() (Model, tea.Cmd) {
	m.screen = screenLogin
	m.grant = session.Grant{}
	m.waiting = false
	m.err = nil
	m.status = "Requesting a code…"
	return m, m.startLogin
}

func needsLogin(err error) bool {
	return errors.Is(err, session.ErrLoggedOut) || connect.CodeOf(err) == connect.CodeUnauthenticated
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	case loggedInMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if !msg.ok {
			return m.toLogin()
		}
		m.screen = screenNotes
		m.status = "Loading notes…"
		return m, m.loadNotes
	case grantMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = ""
			return m, nil
		}
		m.grant = msg.grant
		m.waiting = true
		m.status = "Waiting for approval…"
		return m, m.poll(m.grant)
	case pollMsg:
		if m.screen != screenLogin || !m.waiting || msg.grant.DeviceCode != m.grant.DeviceCode {
			return m, nil
		}
		m.grant = msg.grant
		switch {
		case msg.err != nil:
			m.waiting = false
			m.err = msg.err
			m.status = ""
			return m, nil
		case msg.done:
			m.waiting = false
			m.screen = screenNotes
			m.status = "Loading notes…"
			return m, m.loadNotes
		default:
			return m, m.poll(m.grant)
		}
	case notesMsg:
		if msg.err != nil {
			if needsLogin(msg.err) {
				return m.toLogin()
			}
			m.err = msg.err
			m.status = ""
			return m, nil
		}
		m.err = nil
		m.status = ""
		m.list = msg.notes
		m.cursor = min(m.cursor, max(len(m.list)-1, 0))
		return m, nil
	case noteMsg:
		if msg.err != nil {
			if needsLogin(msg.err) {
				return m.toLogin()
			}
			m.err = msg.err
			m.status = ""
			return m, nil
		}
		m.err = nil
		m.status = ""
		m.note = msg.note
		m.scroll = 0
		m.screen = screenNote
		return m, nil
	}
	return m, nil
}

func (m Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}
	switch m.screen {
	case screenLogin:
		if msg.String() == "r" && !m.waiting {
			return m.toLogin()
		}
	case screenNotes:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.list)-1 {
				m.cursor++
			}
		case "r":
			m.err = nil
			m.status = "Loading notes…"
			return m, m.loadNotes
		case "enter":
			if m.cursor < len(m.list) {
				m.err = nil
				m.status = "Opening…"
				return m, m.loadNote(m.list[m.cursor].GetId())
			}
		}
	case screenNote:
		switch msg.String() {
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			if m.scroll < len(m.noteLines())-1 {
				m.scroll++
			}
		case "esc", "backspace", "h", "left":
			m.screen = screenNotes
			m.note = nil
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m Model) render() string {
	var b strings.Builder
	switch m.screen {
	case screenStarting:
		b.WriteString(dimStyle.Render("Starting…"))
	case screenLogin:
		b.WriteString(m.loginView())
	case screenNotes:
		b.WriteString(m.notesView())
	case screenNote:
		b.WriteString(m.noteView())
	}
	if m.status != "" {
		b.WriteString("\n" + dimStyle.Render(m.status))
	}
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render(describe(m.err)))
	}
	b.WriteString("\n\n" + dimStyle.Render(m.help()))
	return b.String()
}

func (m Model) help() string {
	switch m.screen {
	case screenLogin:
		if !m.waiting {
			return "r new code · q quit"
		}
	case screenNotes:
		return "↑/↓ move · enter open · r reload · q quit"
	case screenNote:
		return "↑/↓ scroll · esc back · q quit"
	}
	return "q quit"
}

func (m Model) loginView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sign in to family-manager") + "\n\n")
	if m.grant.UserCode == "" {
		return b.String()
	}
	b.WriteString(codeStyle.Render(m.grant.UserCode) + "\n\n")
	b.WriteString(session.ApproveHint + ".\n")
	if !m.grant.ExpiresAt.IsZero() {
		b.WriteString(dimStyle.Render("The code expires at "+m.grant.ExpiresAt.Local().Format(time.Kitchen)+".") + "\n")
	}
	return b.String()
}

func (m Model) notesView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Notes") + "\n\n")
	if len(m.list) == 0 {
		if m.status == "" && m.err == nil {
			b.WriteString(dimStyle.Render("No notes yet.") + "\n")
		}
		return b.String()
	}
	rows := max(m.height-6, 3)
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := min(start+rows, len(m.list))
	for i := start; i < end; i++ {
		n := m.list[i]
		title := noteTitle(n)
		if n.GetStarred() {
			title = "★ " + title
		}
		row := "  " + title
		if i == m.cursor {
			row = cursorStyle.Render("› " + title)
		}
		if p := strings.TrimSpace(n.GetPreview()); p != "" {
			row += "  " + dimStyle.Render(truncate(firstLine(p), max(m.width-len([]rune(title))-6, 10)))
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}

func (m Model) noteView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(noteTitle(m.note)) + "\n\n")
	lines := m.noteLines()
	rows := max(m.height-6, 3)
	start := min(m.scroll, max(len(lines)-1, 0))
	end := min(start+rows, len(lines))
	for _, l := range lines[start:end] {
		b.WriteString(l + "\n")
	}
	return b.String()
}

func (m Model) noteLines() []string {
	if m.note == nil {
		return nil
	}
	width := max(m.width-2, 20)
	return strings.Split(RenderBlocks(m.note.GetBlocks(), width), "\n")
}

func RenderBlocks(blocks []*notesv1.Block, width int) string {
	wrap := lipgloss.NewStyle().Width(width)
	var out []string
	number := 0
	for _, bl := range blocks {
		if bl.GetType() != notesv1.BlockType_BLOCK_TYPE_NUMBERED {
			number = 0
		}
		text := bl.GetText()
		switch bl.GetType() {
		case notesv1.BlockType_BLOCK_TYPE_HEADING:
			out = append(out, titleStyle.Width(width).Render(text))
		case notesv1.BlockType_BLOCK_TYPE_TODO:
			box := "[ ] "
			if bl.GetChecked() {
				box = "[x] "
			}
			out = append(out, wrap.Render(box+text))
		case notesv1.BlockType_BLOCK_TYPE_BULLET:
			out = append(out, wrap.Render("• "+text))
		case notesv1.BlockType_BLOCK_TYPE_NUMBERED:
			number++
			out = append(out, wrap.Render(fmt.Sprintf("%d. %s", number, text)))
		case notesv1.BlockType_BLOCK_TYPE_QUOTE:
			out = append(out, quoteStyle.Width(width).Render("│ "+text))
		case notesv1.BlockType_BLOCK_TYPE_CODE:
			out = append(out, codeBlock.Render(text))
		case notesv1.BlockType_BLOCK_TYPE_DIVIDER:
			out = append(out, dimStyle.Render(strings.Repeat("─", min(width, 40))))
		case notesv1.BlockType_BLOCK_TYPE_IMAGE:
			continue
		default:
			out = append(out, wrap.Render(text))
		}
	}
	return strings.Join(out, "\n")
}

func noteTitle(n *notesv1.Note) string {
	if t := strings.TrimSpace(n.GetTitle()); t != "" {
		return t
	}
	return "Untitled"
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func describe(err error) string {
	switch {
	case errors.Is(err, session.ErrDenied):
		return "The sign-in was denied."
	case errors.Is(err, session.ErrExpired):
		return "The code expired before it was approved."
	default:
		return err.Error()
	}
}

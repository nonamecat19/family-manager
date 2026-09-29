package notes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"
	"github.com/nnc/family-manager/sdk/go/notes/v1/notesv1connect"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

const (
	intro        = string(i18n.NotesIntro)
	listLimit    = 8
	searchLimit  = 8
	maxTitleRune = 60
	stateCapture = "notes:capture"
	stateSearch  = "notes:search"
)

type client struct {
	rpc notesv1connect.NotesServiceClient
}

func Bot(httpClient *http.Client, addr string) bot.Options {
	c := &client{rpc: notesv1connect.NewNotesServiceClient(httpClient, addr)}

	return bot.Options{
		Intro: intro,
		Home:  c.home,
		Commands: []bot.Command{
			{Name: "notes", Help: string(i18n.NotesRecent), Run: c.showNotes},
			{Name: "note", Args: "<text>", Help: string(i18n.NotesNew), Run: c.note},
			{Name: "find", Args: "<query>", Help: string(i18n.NotesSearch), Run: c.find},
			{Name: "notebooks", Help: string(i18n.NotesBooks), Run: c.showNotebooks},
		},
		Callbacks: []bot.Callback{
			{Prefix: "list", Run: c.showNotes},
			{Prefix: "books", Run: c.showNotebooks},
			{Prefix: "new", Run: c.askCapture},
			{Prefix: "search", Run: c.askSearch},
			{Prefix: "open", Run: c.open},
		},
		OnText: c.onText,
	}
}

func (c *client) home(ctx context.Context, cc *bot.Context) (string, bot.Keyboard, error) {
	text := bot.Lines(
		bot.Bold(cc.T(i18n.NotesTitle)),
		"",
		bot.Italic(cc.T(i18n.NotesHint)),
	)
	keyboard := bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.NotesNew), "new"), bot.Data(cc.T(i18n.NotesSearch), "search")),
		bot.Row(bot.Data(cc.T(i18n.NotesRecent), "list"), bot.Data(cc.T(i18n.NotesBooks), "books")),
	}
	return text, keyboard, nil
}

func (c *client) askCapture(ctx context.Context, cc *bot.Context) error {
	if err := cc.SetState(ctx, stateCapture, nil); err != nil {
		return err
	}
	return cc.Show(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.NotesNew)),
		"",
		bot.Esc(cc.T(i18n.NotesAskText)),
	), cancelOnly(cc))
}

func (c *client) askSearch(ctx context.Context, cc *bot.Context) error {
	if err := cc.SetState(ctx, stateSearch, nil); err != nil {
		return err
	}
	return cc.Show(ctx, bot.Lines(
		bot.Bold(cc.T(i18n.NotesSearch)),
		"",
		bot.Esc(cc.T(i18n.NotesAskQuery)),
	), cancelOnly(cc))
}

func (c *client) onText(ctx context.Context, cc *bot.Context) error {
	state, ok := cc.State(ctx)
	if !ok {
		return cc.Send(ctx, bot.Italic(cc.T(i18n.OnlyButtons)), nil)
	}

	cc.ClearState(ctx)
	switch state.Kind {
	case stateCapture:
		return c.create(ctx, cc, cc.Args)
	case stateSearch:
		return c.search(ctx, cc, cc.Args)
	default:
		return c.showNotes(ctx, cc)
	}
}

func (c *client) note(ctx context.Context, cc *bot.Context) error {
	body := strings.TrimSpace(cc.Args)
	if body == "" {
		return c.askCapture(ctx, cc)
	}
	return c.create(ctx, cc, body)
}

func (c *client) create(ctx context.Context, cc *bot.Context, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return bot.Invalid("%s", cc.T(i18n.NotesNeedText, "buy a new kettle"))
	}

	notebookID, err := c.firstNotebook(ctx, cc)
	if err != nil {
		return err
	}

	head, rest, _ := strings.Cut(body, "\n")
	req := connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: notebookID,
		Title:      truncate(head, maxTitleRune),
		Blocks: []*notesv1.Block{{
			Type: notesv1.BlockType_BLOCK_TYPE_PARAGRAPH,
			Text: body,
		}},
	})
	cc.Authorize(req)

	res, err := c.rpc.CreateNote(ctx, req)
	if err != nil {
		return err
	}

	lines := []string{bot.Bold(cc.T(i18n.NotesSaved)), "",
		bot.Bold(bot.Esc(title(cc, res.Msg.GetNote())))}
	if strings.TrimSpace(rest) != "" {
		lines = append(lines, bot.Italic(cc.T(i18n.NotesRestInBody)))
	}

	return cc.Send(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.NotesAnother), "new"), bot.Data(cc.T(i18n.NotesRecent), "list")),
		bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
	})
}

func (c *client) find(ctx context.Context, cc *bot.Context) error {
	query := strings.TrimSpace(cc.Args)
	if query == "" {
		return c.askSearch(ctx, cc)
	}
	return c.search(ctx, cc, query)
}

func (c *client) search(ctx context.Context, cc *bot.Context, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return bot.Invalid("%s", cc.T(i18n.NotesAskQuery))
	}

	req := connect.NewRequest(&notesv1.SearchRequest{
		Query: query,
		Facet: notesv1.SearchFacet_SEARCH_FACET_ALL,
		Limit: searchLimit,
	})
	cc.Authorize(req)

	res, err := c.rpc.Search(ctx, req)
	if err != nil {
		return err
	}
	if len(res.Msg.GetHits()) == 0 {
		return cc.Send(ctx, bot.Lines(bot.Bold(cc.T(i18n.NotesSearch)), "",
			cc.T(i18n.NotesNoMatch, bot.Code(query))), backOnly(cc))
	}

	var buttons []bot.Button
	lines := []string{bot.Bold(cc.T(i18n.NotesMatches, bot.Esc(query))), ""}
	for _, h := range res.Msg.GetHits() {
		name := fallback(h.GetTitle(), cc.T(i18n.NotesUntitled))
		lines = append(lines, "• "+bot.Bold(bot.Esc(name)))
		if snippet := strings.TrimSpace(h.GetSnippet()); snippet != "" {
			lines = append(lines, "      "+bot.Italic(bot.Esc(snippet)))
		}
		buttons = append(buttons, bot.Data(bot.Esc(name), "open:"+h.GetNoteId()))
	}

	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.Menu), "home")))
	return cc.Send(ctx, strings.Join(lines, "\n"), keyboard)
}

func (c *client) showNotes(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&notesv1.ListNotesRequest{
		Sort:     notesv1.NoteSort_NOTE_SORT_UPDATED,
		PageSize: listLimit,
	})
	cc.Authorize(req)

	res, err := c.rpc.ListNotes(ctx, req)
	if err != nil {
		return err
	}
	if len(res.Msg.GetNotes()) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.NotesRecent)), "",
			bot.Italic(cc.T(i18n.NotesEmpty))), bot.Keyboard{
			bot.Row(bot.Data(cc.T(i18n.NotesNew), "new")),
			bot.Row(bot.Data(cc.T(i18n.Menu), "home")),
		})
	}

	var buttons []bot.Button
	lines := []string{bot.Bold(cc.T(i18n.NotesRecent)), ""}
	for _, n := range res.Msg.GetNotes() {
		lines = append(lines, "• "+bot.Bold(bot.Esc(title(cc, n))))
		if preview := strings.TrimSpace(n.GetPreview()); preview != "" {
			lines = append(lines, "      "+bot.Italic(bot.Esc(preview)))
		}
		if n.GetTaskTotal() > 0 {
			lines = append(lines, fmt.Sprintf("      ☑ %d/%d", n.GetTaskDone(), n.GetTaskTotal()))
		}
		buttons = append(buttons, bot.Data(bot.Esc(title(cc, n)), "open:"+n.GetId()))
	}

	keyboard := bot.Keyboard{}.Grid(buttons, 1)
	keyboard = append(keyboard, bot.Row(bot.Data(cc.T(i18n.NotesNew), "new"),
		bot.Data(cc.T(i18n.Menu), "home")))
	return cc.Show(ctx, strings.Join(lines, "\n"), keyboard)
}

func (c *client) open(ctx context.Context, cc *bot.Context) error {
	req := connect.NewRequest(&notesv1.GetNoteRequest{NoteId: cc.Payload()})
	cc.Authorize(req)

	res, err := c.rpc.GetNote(ctx, req)
	if err != nil {
		return err
	}
	n := res.Msg.GetNote()
	if n == nil {
		return bot.Invalid("%s", cc.T(i18n.NotesGone))
	}

	lines := []string{bot.Bold(bot.Esc(title(cc, n))), ""}
	for _, b := range n.GetBlocks() {
		text := strings.TrimSpace(b.GetText())
		if text == "" {
			continue
		}
		switch b.GetType() {
		case notesv1.BlockType_BLOCK_TYPE_TODO:
			box := "☐"
			if b.GetChecked() {
				box = "☑"
			}
			lines = append(lines, box+" "+bot.Esc(text))
		case notesv1.BlockType_BLOCK_TYPE_HEADING:
			lines = append(lines, bot.Bold(bot.Esc(text)))
		case notesv1.BlockType_BLOCK_TYPE_BULLET:
			lines = append(lines, "• "+bot.Esc(text))
		default:
			lines = append(lines, bot.Esc(text))
		}
	}

	return cc.Show(ctx, strings.Join(lines, "\n"), bot.Keyboard{
		bot.Row(bot.Data(cc.T(i18n.NotesRecent), "list"), bot.Data(cc.T(i18n.Menu), "home")),
	})
}

func (c *client) showNotebooks(ctx context.Context, cc *bot.Context) error {
	books, err := c.notebooks(ctx, cc)
	if err != nil {
		return err
	}
	if len(books) == 0 {
		return cc.Show(ctx, bot.Lines(bot.Bold(cc.T(i18n.NotesBooks)), "",
			bot.Italic(cc.T(i18n.NotesNoBooks))), backOnly(cc))
	}

	lines := []string{bot.Bold(cc.T(i18n.NotesBooks)), ""}
	for _, n := range books {
		lines = append(lines, fmt.Sprintf("• %s  %s",
			bot.Bold(bot.Esc(n.GetName())), bot.Italic(cc.T(i18n.NotesCount, n.GetNoteCount()))))
	}
	return cc.Show(ctx, strings.Join(lines, "\n"), backOnly(cc))
}

func (c *client) notebooks(ctx context.Context, cc *bot.Context) ([]*notesv1.Notebook, error) {
	req := connect.NewRequest(&notesv1.ListNotebooksRequest{})
	cc.Authorize(req)

	res, err := c.rpc.ListNotebooks(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Msg.GetNotebooks(), nil
}

func (c *client) firstNotebook(ctx context.Context, cc *bot.Context) (string, error) {
	books, err := c.notebooks(ctx, cc)
	if err != nil {
		return "", err
	}
	if len(books) == 0 {
		return "", nil
	}
	return books[0].GetId(), nil
}

func backOnly(cc *bot.Context) bot.Keyboard {
	return bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Menu), "home"))}
}

func cancelOnly(cc *bot.Context) bot.Keyboard {
	return bot.Keyboard{bot.Row(bot.Data(cc.T(i18n.Cancel), "home"))}
}

func title(cc *bot.Context, n *notesv1.Note) string {
	return fallback(strings.TrimSpace(n.GetTitle()), cc.T(i18n.NotesUntitled))
}

func fallback(value, alt string) string {
	if value == "" {
		return alt
	}
	return value
}

func truncate(s string, limit int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

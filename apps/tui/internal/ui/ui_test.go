package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"

	"github.com/nnc/family-manager/apps/tui/internal/session"
)

type fakeAuth struct {
	loggedIn bool
	grant    session.Grant
}

func (f *fakeAuth) LoggedIn() (bool, error) { return f.loggedIn, nil }

func (f *fakeAuth) StartDeviceLogin(context.Context) (session.Grant, error) { return f.grant, nil }

func (f *fakeAuth) Poll(context.Context, *session.Grant) (bool, error) { return false, nil }

type fakeNotes struct {
	notes []*notesv1.Note
	err   error
}

func (f *fakeNotes) ListNotes(context.Context, *connect.Request[notesv1.ListNotesRequest]) (*connect.Response[notesv1.ListNotesResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.ListNotesResponse{Notes: f.notes}), nil
}

func (f *fakeNotes) GetNote(_ context.Context, req *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error) {
	for _, n := range f.notes {
		if n.GetId() == req.Msg.GetNoteId() {
			return connect.NewResponse(&notesv1.GetNoteResponse{Note: n}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("no note"))
}

func step(t *testing.T, m tea.Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func TestLoggedOutShowsDeviceCode(t *testing.T) {
	auth := &fakeAuth{grant: session.Grant{DeviceCode: "d", UserCode: "BCDF-GHJK", Interval: time.Hour, ExpiresAt: time.Now().Add(10 * time.Minute)}}
	m := New(context.Background(), auth, &fakeNotes{})
	m, cmd := step(t, m, m.Init()())
	if m.screen != screenLogin || cmd == nil {
		t.Fatalf("screen = %v, cmd = %v", m.screen, cmd)
	}
	m, _ = step(t, m, cmd())
	view := m.render()
	if !strings.Contains(view, "BCDF-GHJK") || !strings.Contains(view, "Settings → Approve a device") {
		t.Fatalf("login view missing code or hint:\n%s", view)
	}
}

func TestApprovedPollLoadsNotes(t *testing.T) {
	notes := &fakeNotes{notes: []*notesv1.Note{{Id: "n1", Title: "Groceries"}}}
	m := New(context.Background(), &fakeAuth{}, notes)
	m.screen = screenLogin
	m.waiting = true
	m.grant = session.Grant{DeviceCode: "d"}
	m, cmd := step(t, m, pollMsg{grant: m.grant, done: true})
	if m.screen != screenNotes || cmd == nil {
		t.Fatalf("screen = %v", m.screen)
	}
	m, _ = step(t, m, cmd())
	if !strings.Contains(m.render(), "Groceries") {
		t.Fatalf("notes view:\n%s", m.render())
	}
}

func TestStalePollIsIgnored(t *testing.T) {
	m := New(context.Background(), &fakeAuth{}, &fakeNotes{})
	m.screen = screenLogin
	m.waiting = true
	m.grant = session.Grant{DeviceCode: "new"}
	m, cmd := step(t, m, pollMsg{grant: session.Grant{DeviceCode: "old"}, done: true})
	if m.screen != screenLogin || cmd != nil {
		t.Fatalf("stale poll changed state: screen = %v", m.screen)
	}
}

func TestDeniedPollOffersNewCode(t *testing.T) {
	m := New(context.Background(), &fakeAuth{}, &fakeNotes{})
	m.screen = screenLogin
	m.waiting = true
	m.grant = session.Grant{DeviceCode: "d", UserCode: "BCDF-GHJK"}
	m, _ = step(t, m, pollMsg{grant: m.grant, err: session.ErrDenied})
	if m.waiting || !strings.Contains(m.render(), "denied") || !strings.Contains(m.render(), "r new code") {
		t.Fatalf("view after denial:\n%s", m.render())
	}
}

func TestLoggedOutListFallsBackToLogin(t *testing.T) {
	m := New(context.Background(), &fakeAuth{}, &fakeNotes{err: session.ErrLoggedOut})
	m.screen = screenNotes
	m, cmd := step(t, m, m.loadNotes())
	if m.screen != screenLogin || cmd == nil {
		t.Fatalf("screen = %v", m.screen)
	}
}

func TestOpenNoteAndBack(t *testing.T) {
	note := &notesv1.Note{Id: "n1", Title: "Trip", Blocks: []*notesv1.Block{
		{Type: notesv1.BlockType_BLOCK_TYPE_PARAGRAPH, Text: "pack the tent"},
	}}
	m := New(context.Background(), &fakeAuth{}, &fakeNotes{notes: []*notesv1.Note{note}})
	m.screen = screenNotes
	m.list = []*notesv1.Note{note}
	m, cmd := step(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = step(t, m, cmd())
	if m.screen != screenNote || !strings.Contains(m.render(), "pack the tent") {
		t.Fatalf("note view:\n%s", m.render())
	}
	m, _ = step(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.screen != screenNotes {
		t.Fatalf("screen after esc = %v", m.screen)
	}
}

func TestRenderBlocks(t *testing.T) {
	out := RenderBlocks([]*notesv1.Block{
		{Type: notesv1.BlockType_BLOCK_TYPE_TODO, Text: "milk", Checked: true},
		{Type: notesv1.BlockType_BLOCK_TYPE_TODO, Text: "eggs"},
		{Type: notesv1.BlockType_BLOCK_TYPE_NUMBERED, Text: "one"},
		{Type: notesv1.BlockType_BLOCK_TYPE_NUMBERED, Text: "two"},
		{Type: notesv1.BlockType_BLOCK_TYPE_BULLET, Text: "dot"},
		{Type: notesv1.BlockType_BLOCK_TYPE_IMAGE, ImageUrl: "https://example.invalid/x.png"},
		{Type: notesv1.BlockType_BLOCK_TYPE_NUMBERED, Text: "again"},
	}, 60)
	for _, want := range []string{"[x] milk", "[ ] eggs", "1. one", "2. two", "• dot", "1. again"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "example.invalid") {
		t.Fatalf("image url rendered:\n%s", out)
	}
}

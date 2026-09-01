package handler

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	fmevents "github.com/nnc/family-manager/libs/go/events"
	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"
	"github.com/nnc/family-manager/services/notes/db"
)

// The cast of one family: Ana owns things, Ben and Cal are the other two members. Dana lives
// in another family entirely and exists to prove family_id still bounds everything.
const (
	userAna  = "00000000-0000-4000-8000-00000000000a"
	userBen  = "00000000-0000-4000-8000-00000000000b"
	userCal  = "00000000-0000-4000-8000-00000000000c"
	userDana = "00000000-0000-4000-8000-00000000000d"

	familyOne = "00000000-0000-4000-8000-0000000000f1"
	familyTwo = "00000000-0000-4000-8000-0000000000f2"
)

// withClaims stamps a context with test claims — the auth interceptor is not in the call path
// for unit tests, so the handler reads claims from context directly.
func withClaims(userID, familyID string) context.Context {
	return fmauth.WithClaims(context.Background(),
		&fmauth.Claims{UserID: userID, FamilyID: familyID})
}

func newTestHandler() (*Handler, *fakeStore, *recorder) {
	store := newFakeStore()
	rec := &recorder{}
	return New(Options{Queries: store, Tx: store, Bus: rec}), store, rec
}

// codeIs fails the test unless err is a Connect error with the wanted code. Comparing codes
// rather than messages keeps these tests from breaking on a reworded sentence.
func codeIs(t *testing.T, err error, want connect.Code, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected %v, got nil error", what, want)
	}
	if got := connect.CodeOf(err); got != want {
		t.Fatalf("%s: code = %v, want %v (err: %v)", what, got, want, err)
	}
}

func para(id, text string) *notesv1.Block {
	return &notesv1.Block{Id: id, Type: notesv1.BlockType_BLOCK_TYPE_PARAGRAPH, Text: text}
}

func todo(id, text string, checked bool) *notesv1.Block {
	return &notesv1.Block{
		Id: id, Type: notesv1.BlockType_BLOCK_TYPE_TODO, Text: text, Checked: checked,
	}
}

// createNote is the setup shortcut: it goes through the RPC rather than writing the map, so a
// note in a test is a note the handler itself would have made.
func createNote(
	t *testing.T, h *Handler, ctx context.Context, title string, blocks ...*notesv1.Block,
) *notesv1.Note {
	t.Helper()
	resp, err := h.CreateNote(ctx, connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: title, Blocks: blocks,
	}))
	if err != nil {
		t.Fatalf("CreateNote(%q): %v", title, err)
	}
	return resp.Msg.GetNote()
}

func shareNote(
	t *testing.T, h *Handler, ctx context.Context, noteID string,
	subject notesv1.ShareSubject, member string, perm notesv1.SharePermission,
) *notesv1.Share {
	t.Helper()
	resp, err := h.ShareNote(ctx, connect.NewRequest(&notesv1.ShareNoteRequest{
		NoteId: noteID, Subject: subject, MemberUserId: member, Permission: perm,
	}))
	if err != nil {
		t.Fatalf("ShareNote: %v", err)
	}
	return resp.Msg.GetShare()
}

func getNote(h *Handler, ctx context.Context, noteID string) (*notesv1.Note, error) {
	resp, err := h.GetNote(ctx, connect.NewRequest(&notesv1.GetNoteRequest{NoteId: noteID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetNote(), nil
}

/* ------------------------------------------------------------------ privacy */

// A note is private to its creator until it is shared. This is the rule the whole service
// turns on, so it is the first thing asserted: a family member is not an audience.
func TestNoteIsPrivateToItsCreator(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Therapy notes", para("b1", "not for the family"))

	_, err := getNote(h, ben, note.GetId())
	codeIs(t, err, connect.CodeNotFound, "GetNote by a non-recipient")

	// It must not surface in the other member's list either — a leak in ListNotes is the
	// same leak, one screen earlier.
	list, err := h.ListNotes(ben, connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if n := len(list.Msg.GetNotes()); n != 0 {
		t.Errorf("ListNotes for a non-recipient returned %d notes, want 0", n)
	}

	// And the owner still sees their own.
	own, err := getNote(h, ana, note.GetId())
	if err != nil {
		t.Fatalf("GetNote by the owner: %v", err)
	}
	if !own.GetCanEdit() {
		t.Error("owner's can_edit = false, want true")
	}
}

// A MEMBER share reaches exactly one person. The third member must stay where they were.
func TestMemberViewShareReachesOnlyThatMember(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	note := createNote(t, h, ana, "Trip plan", para("b1", "flights on the 4th"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	got, err := getNote(h, ben, note.GetId())
	if err != nil {
		t.Fatalf("GetNote by the share's member: %v", err)
	}
	if got.GetCanEdit() {
		t.Error("can_edit = true for a VIEW share, want false")
	}
	if !got.GetShared() {
		t.Error("shared = false on a shared note, want true")
	}

	_, err = getNote(h, cal, note.GetId())
	codeIs(t, err, connect.CodeNotFound, "GetNote by a member the note was not shared with")
}

// A FAMILY share reaches every member of the caller's own family — and nobody outside it.
func TestFamilyShareReachesEveryMember(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "House rules", para("b1", "shoes off"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_FAMILY, "",
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	for _, member := range []string{userBen, userCal} {
		if _, err := getNote(h, withClaims(member, familyOne), note.GetId()); err != nil {
			t.Errorf("GetNote by family member %s: %v", member, err)
		}
	}

	// FAMILY is not "public": someone carrying a different family_id claim still sees nothing.
	_, err := getNote(h, withClaims(userDana, familyTwo), note.GetId())
	codeIs(t, err, connect.CodeNotFound, "GetNote from another family")
}

/* ------------------------------------------------------------------ permissions */

func TestViewShareCannotUpdate(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Recipe", para("b1", "two eggs"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := h.UpdateNote(ben, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: note.GetId(), Title: "Recipe", Blocks: []*notesv1.Block{para("b1", "three eggs")},
		ExpectedVersion: note.GetVersion(),
	}))
	// PermissionDenied, not NotFound: the caller can already see the note, so refusing the
	// write tells them nothing they did not know.
	codeIs(t, err, connect.CodePermissionDenied, "UpdateNote with a VIEW share")

	after, err := getNote(h, ana, note.GetId())
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if after.GetVersion() != note.GetVersion() {
		t.Errorf("version = %d after a refused write, want %d",
			after.GetVersion(), note.GetVersion())
	}
}

func TestEditShareCanUpdate(t *testing.T) {
	h, _, rec := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Recipe", para("b1", "two eggs"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	resp, err := h.UpdateNote(ben, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: note.GetId(), Title: "Recipe", Blocks: []*notesv1.Block{para("b1", "three eggs")},
		ExpectedVersion: note.GetVersion(),
	}))
	if err != nil {
		t.Fatalf("UpdateNote with an EDIT share: %v", err)
	}
	updated := resp.Msg.GetNote()
	if updated.GetVersion() != note.GetVersion()+1 {
		t.Errorf("version = %d, want %d", updated.GetVersion(), note.GetVersion()+1)
	}
	if updated.GetLastEditedByUserId() != userBen {
		t.Errorf("last_edited_by = %q, want %q", updated.GetLastEditedByUserId(), userBen)
	}
	if len(updated.GetBlocks()) != 1 || updated.GetBlocks()[0].GetText() != "three eggs" {
		t.Errorf("blocks = %v, want the incoming array", updated.GetBlocks())
	}
	if !rec.sawSubject(fmevents.SubjectNotesNoteUpdated) {
		t.Error("notes.note.updated not published")
	}
}

// Sharing is the owner's alone. An EDIT share grants writing the document, not handing it on.
func TestOnlyTheOwnerCanShare(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	note := createNote(t, h, ana, "Budget", para("b1", "rent"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	_, err := h.ShareNote(ben, connect.NewRequest(&notesv1.ShareNoteRequest{
		NoteId:  note.GetId(),
		Subject: notesv1.ShareSubject_SHARE_SUBJECT_MEMBER,
		// Ben tries to pass the note on to Cal.
		MemberUserId: userCal,
		Permission:   notesv1.SharePermission_SHARE_PERMISSION_VIEW,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "ShareNote by an EDIT-share holder")

	// Someone who cannot see the note at all learns nothing from the answer.
	_, err = h.ShareNote(cal, connect.NewRequest(&notesv1.ShareNoteRequest{
		NoteId:       note.GetId(),
		Subject:      notesv1.ShareSubject_SHARE_SUBJECT_MEMBER,
		MemberUserId: userBen,
		Permission:   notesv1.SharePermission_SHARE_PERMISSION_VIEW,
	}))
	codeIs(t, err, connect.CodeNotFound, "ShareNote by a stranger to the note")

	// And Cal still cannot read it.
	_, err = getNote(h, cal, note.GetId())
	codeIs(t, err, connect.CodeNotFound, "GetNote after a refused re-share")
}

/* ------------------------------------------------------------------ conditional write */

func TestUpdateNoteRejectsStaleVersionAndZeroForces(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Draft", para("b1", "first"))

	// One good write moves the note to version 2, so the client's copy of version 1 is stale.
	if _, err := h.UpdateNote(ana, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: note.GetId(), Title: "Draft", Blocks: []*notesv1.Block{para("b1", "second")},
		ExpectedVersion: note.GetVersion(),
	})); err != nil {
		t.Fatalf("first UpdateNote: %v", err)
	}

	_, err := h.UpdateNote(ana, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: note.GetId(), Title: "Draft", Blocks: []*notesv1.Block{para("b1", "stale")},
		ExpectedVersion: note.GetVersion(), // still 1
	}))
	codeIs(t, err, connect.CodeAborted, "UpdateNote with a stale expected_version")

	stored, err := getNote(h, ana, note.GetId())
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if stored.GetBlocks()[0].GetText() != "second" {
		t.Errorf("text = %q after a refused write, want %q",
			stored.GetBlocks()[0].GetText(), "second")
	}

	// 0 is the force: the app sends it after showing the conflict.
	resp, err := h.UpdateNote(ana, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: note.GetId(), Title: "Draft", Blocks: []*notesv1.Block{para("b1", "forced")},
		ExpectedVersion: 0,
	}))
	if err != nil {
		t.Fatalf("forced UpdateNote: %v", err)
	}
	if got := resp.Msg.GetNote().GetBlocks()[0].GetText(); got != "forced" {
		t.Errorf("text = %q after the forced write, want %q", got, "forced")
	}
	if got := resp.Msg.GetNote().GetVersion(); got != 3 {
		t.Errorf("version = %d after the forced write, want 3", got)
	}
}

/* ------------------------------------------------------------------ idempotent capture */

func TestCreateNoteWithSameClientIDMakesOneNote(t *testing.T) {
	h, store, rec := newTestHandler()
	ana := withClaims(userAna, familyOne)

	req := &notesv1.CreateNoteRequest{
		ClientId: "offline-capture-1",
		Title:    "Milk",
		Blocks:   []*notesv1.Block{todo("b1", "buy milk", false)},
	}

	first, err := h.CreateNote(ana, connect.NewRequest(req))
	if err != nil {
		t.Fatalf("first CreateNote: %v", err)
	}
	second, err := h.CreateNote(ana, connect.NewRequest(req))
	if err != nil {
		t.Fatalf("retried CreateNote: %v", err)
	}

	if first.Msg.GetNote().GetId() != second.Msg.GetNote().GetId() {
		t.Errorf("retry returned note %q, want the first call's %q",
			second.Msg.GetNote().GetId(), first.Msg.GetNote().GetId())
	}
	if n := len(store.notes); n != 1 {
		t.Errorf("stored notes = %d, want 1", n)
	}
	// A retry writes no second CREATED row and re-fires no event.
	if n := rec.count(fmevents.SubjectNotesNoteCreated); n != 1 {
		t.Errorf("notes.note.created published %d times, want 1", n)
	}
	activity, err := h.ListActivity(ana, connect.NewRequest(&notesv1.ListActivityRequest{
		NoteId: first.Msg.GetNote().GetId(),
	}))
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	if n := len(activity.Msg.GetActivity()); n != 1 {
		t.Errorf("activity rows = %d, want 1", n)
	}
}

// ACTIVITY_KIND_SHARED is a kind the app renders, so it has to be a kind the server writes.
// Nothing else in the service emits it — before this, sharing left the rail silent.
func TestSharingWritesSharedActivity(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Trip plan", para("b1", "flights on the 4th"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	entry := onlySharedActivity(t, h, ana, note.GetId())
	// The detail is the sharee's id: this service has no name lookup, and the app maps ids
	// to names already.
	if entry.GetDetail() != userBen {
		t.Errorf("detail = %q, want the sharee's user id %q", entry.GetDetail(), userBen)
	}
}

// A FAMILY share names nobody, so the row carries no detail and the app says "shared this note".
func TestFamilyShareActivityNamesNobody(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "House rules", para("b1", "shoes off"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_FAMILY, "",
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	if detail := onlySharedActivity(t, h, ana, note.GetId()).GetDetail(); detail != "" {
		t.Errorf("detail = %q on a family share, want empty", detail)
	}
}

// A notebook has no rail of its own, so its share is recorded on every note inside it.
func TestNotebookShareWritesActivityOnEachNote(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	nb, err := h.CreateNotebook(ana, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: "Product",
	}))
	if err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}
	notebookID := nb.Msg.GetNotebook().GetId()

	var ids []string
	for _, title := range []string{"Roadmap", "Retro"} {
		created, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
			NotebookId: notebookID, Title: title,
			Blocks: []*notesv1.Block{para("b1", "ship the thing")},
		}))
		if err != nil {
			t.Fatalf("CreateNote(%q): %v", title, err)
		}
		ids = append(ids, created.Msg.GetNote().GetId())
	}

	shareNotebook(t, h, ana, notebookID, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	for _, id := range ids {
		if detail := onlySharedActivity(t, h, ana, id).GetDetail(); detail != userBen {
			t.Errorf("note %s: detail = %q, want %q", id, detail, userBen)
		}
	}
}

// onlySharedActivity returns the note's single SHARED row, failing if there is not exactly one.
func onlySharedActivity(
	t *testing.T, h *Handler, ctx context.Context, noteID string,
) *notesv1.Activity {
	t.Helper()
	resp, err := h.ListActivity(ctx, connect.NewRequest(&notesv1.ListActivityRequest{
		NoteId: noteID,
	}))
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	var shared []*notesv1.Activity
	for _, a := range resp.Msg.GetActivity() {
		if a.GetKind() == notesv1.ActivityKind_ACTIVITY_KIND_SHARED {
			shared = append(shared, a)
		}
	}
	if len(shared) != 1 {
		t.Fatalf("SHARED activity rows = %d, want 1", len(shared))
	}
	return shared[0]
}

/* ------------------------------------------------------------------ search */

// Search runs the visibility predicate in SQL. What this asserts is that the palette can
// never be the way a private note leaks.
func TestSearchNeverReturnsAnInvisibleNote(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Surprise party",
		para("b1", "the cake is a surprise"), todo("b2", "order the surprise cake", false))

	search := func(ctx context.Context) *notesv1.SearchResponse {
		t.Helper()
		resp, err := h.Search(ctx, connect.NewRequest(&notesv1.SearchRequest{
			Query: "surprise", Facet: notesv1.SearchFacet_SEARCH_FACET_ALL,
		}))
		if err != nil {
			t.Fatalf("Search: %v", err)
		}
		return resp.Msg
	}

	if n := len(search(ana).GetHits()); n == 0 {
		t.Fatal("the owner's own search found nothing; the fixture is wrong, not the rule")
	}

	got := search(ben)
	if n := len(got.GetHits()); n != 0 {
		t.Errorf("search by a non-recipient returned %d hits, want 0: %v", n, got.GetHits())
	}
	if got.GetSearchedNotes() != 0 {
		t.Errorf("searched_notes = %d for a caller who can see nothing, want 0",
			got.GetSearchedNotes())
	}

	// Once it is shared, the same query finds it — both the note and its task line.
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_FAMILY, "",
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	after := search(ben)
	kinds := map[notesv1.SearchFacet]int{}
	for _, hit := range after.GetHits() {
		kinds[hit.GetKind()]++
		if hit.GetNoteId() != note.GetId() {
			t.Errorf("hit points at note %q, want %q", hit.GetNoteId(), note.GetId())
		}
	}
	if kinds[notesv1.SearchFacet_SEARCH_FACET_NOTES] != 1 {
		t.Errorf("NOTES hits = %d, want 1", kinds[notesv1.SearchFacet_SEARCH_FACET_NOTES])
	}
	if kinds[notesv1.SearchFacet_SEARCH_FACET_TASKS] != 1 {
		t.Errorf("TASKS hits = %d, want 1", kinds[notesv1.SearchFacet_SEARCH_FACET_TASKS])
	}
	if after.GetSearchedNotes() != 1 {
		t.Errorf("searched_notes = %d, want 1", after.GetSearchedNotes())
	}
}

/* ------------------------------------------------------------------ family scope */

// family_id bounds everything, and it bounds it BEFORE sharing is consulted: a share row
// pointing at a member of another family is not a door into this one.
func TestAnotherFamilysNoteStaysInvisibleEvenWhenShared(t *testing.T) {
	h, store, _ := newTestHandler()

	// A note in family two, owned by Dana, shared as widely as the model allows: to the whole
	// family AND directly to Ana, who lives in family one.
	note := store.insertNote(
		pgconv.MustUUID(familyTwo), pgconv.MustUUID(userDana), pgtype.UUID{},
		"", "Other family's note", []byte(`[]`), "")
	store.noteShares["family-share"] = db.NoteShare{
		ID: pgconv.MustUUID(newUUID()), NoteID: note.ID, FamilyID: note.FamilyID,
		Subject: 2, Permission: 2, GrantedByUserID: note.OwnerUserID, CreatedAt: store.tick(),
	}
	store.noteShares["member-share"] = db.NoteShare{
		ID: pgconv.MustUUID(newUUID()), NoteID: note.ID, FamilyID: note.FamilyID,
		Subject: 1, MemberUserID: pgconv.MustUUID(userAna), Permission: 2,
		GrantedByUserID: note.OwnerUserID, CreatedAt: store.tick(),
	}

	ana := withClaims(userAna, familyOne)
	noteID := pgconv.UUIDString(note.ID)

	_, err := getNote(h, ana, noteID)
	codeIs(t, err, connect.CodeNotFound, "GetNote across families")

	_, err = h.UpdateNote(ana, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: noteID, Title: "mine now", ExpectedVersion: 0,
	}))
	codeIs(t, err, connect.CodeNotFound, "UpdateNote across families")

	_, err = h.AddComment(ana, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: noteID, Body: "hello",
	}))
	codeIs(t, err, connect.CodeNotFound, "AddComment across families")

	_, err = h.DeleteNote(ana, connect.NewRequest(&notesv1.DeleteNoteRequest{NoteId: noteID}))
	codeIs(t, err, connect.CodeNotFound, "DeleteNote across families")

	list, err := h.ListNotes(ana, connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if n := len(list.Msg.GetNotes()); n != 0 {
		t.Errorf("ListNotes returned %d notes from another family, want 0", n)
	}

	shared, err := h.ListSharedWithMe(ana, connect.NewRequest(&notesv1.ListSharedWithMeRequest{}))
	if err != nil {
		t.Fatalf("ListSharedWithMe: %v", err)
	}
	if n := len(shared.Msg.GetNotes()); n != 0 {
		t.Errorf("ListSharedWithMe returned %d notes from another family, want 0", n)
	}
}

/* ------------------------------------------------------------------ notebook sharing */

// Sharing a notebook shares the notes inside it — that is the second half of the visibility
// predicate, and the half a handler-only check would miss.
func TestNotebookShareCarriesToTheNotesInside(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	nb, err := h.CreateNotebook(ana, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: "Product",
	}))
	if err != nil {
		t.Fatalf("CreateNotebook: %v", err)
	}
	created, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb.Msg.GetNotebook().GetId(),
		Title:      "Roadmap",
		Blocks:     []*notesv1.Block{para("b1", "ship the thing")},
	}))
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	note := created.Msg.GetNote()

	_, err = getNote(h, ben, note.GetId())
	codeIs(t, err, connect.CodeNotFound, "GetNote before the notebook is shared")

	if _, err := h.ShareNotebook(ana, connect.NewRequest(&notesv1.ShareNotebookRequest{
		NotebookId:   nb.Msg.GetNotebook().GetId(),
		Subject:      notesv1.ShareSubject_SHARE_SUBJECT_MEMBER,
		MemberUserId: userBen,
		Permission:   notesv1.SharePermission_SHARE_PERMISSION_EDIT,
	})); err != nil {
		t.Fatalf("ShareNotebook: %v", err)
	}

	got, err := getNote(h, ben, note.GetId())
	if err != nil {
		t.Fatalf("GetNote after the notebook is shared: %v", err)
	}
	if !got.GetCanEdit() {
		t.Error("can_edit = false under an EDIT notebook share, want true")
	}
	// A notebook EDIT share is authority over the notes, not over the folder.
	_, err = h.UpdateNotebook(ben, connect.NewRequest(&notesv1.UpdateNotebookRequest{
		NotebookId: nb.Msg.GetNotebook().GetId(), Name: "Ben's now",
	}))
	codeIs(t, err, connect.CodePermissionDenied, "UpdateNotebook by an EDIT-share holder")
}

/* ------------------------------------------------------------------ owner-only actions */

func TestOnlyTheOwnerCanDelete(t *testing.T) {
	h, store, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Keepsake", para("b1", "do not delete"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	_, err := h.DeleteNote(ben, connect.NewRequest(&notesv1.DeleteNoteRequest{
		NoteId: note.GetId(),
	}))
	codeIs(t, err, connect.CodePermissionDenied, "DeleteNote by an EDIT-share holder")
	if _, ok := store.notes[note.GetId()]; !ok {
		t.Fatal("note was deleted by someone who does not own it")
	}

	if _, err := h.DeleteNote(ana, connect.NewRequest(&notesv1.DeleteNoteRequest{
		NoteId: note.GetId(),
	})); err != nil {
		t.Fatalf("DeleteNote by the owner: %v", err)
	}
	if _, ok := store.notes[note.GetId()]; ok {
		t.Error("note survived its owner's delete")
	}
}

// Starring writes a column everyone who can see the note reads, so it needs EDIT — and it
// must not count as an edit, or every other viewer's next save would conflict.
func TestToggleStarNeedsEditAndDoesNotBumpVersion(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Starred", para("b1", "text"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := h.ToggleStar(ben, connect.NewRequest(&notesv1.ToggleStarRequest{
		NoteId: note.GetId(), Starred: true,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "ToggleStar with a VIEW share")

	resp, err := h.ToggleStar(ana, connect.NewRequest(&notesv1.ToggleStarRequest{
		NoteId: note.GetId(), Starred: true,
	}))
	if err != nil {
		t.Fatalf("ToggleStar by the owner: %v", err)
	}
	if !resp.Msg.GetNote().GetStarred() {
		t.Error("starred = false after ToggleStar(true)")
	}
	if got := resp.Msg.GetNote().GetVersion(); got != note.GetVersion() {
		t.Errorf("version = %d after starring, want %d (starring is not an edit)",
			got, note.GetVersion())
	}
}

/* ------------------------------------------------------------------ derived columns */

// task_total/task_done and preview come from the read query, not from Go. This checks the
// handler ships what the query computed rather than recomputing it a second, drifting time.
func TestNoteCarriesTaskCountsAndPreview(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Chores",
		para("b1", "before the weekend"),
		todo("b2", "wash up", true),
		todo("b3", "hoover", false),
		todo("b4", "bins", true),
	)
	if note.GetTaskTotal() != 3 {
		t.Errorf("task_total = %d, want 3", note.GetTaskTotal())
	}
	if note.GetTaskDone() != 2 {
		t.Errorf("task_done = %d, want 2", note.GetTaskDone())
	}
	if note.GetPreview() == "" {
		t.Error("preview is empty, want the flattened body")
	}
	if note.GetShared() {
		t.Error("shared = true on a note nobody has shared")
	}
}

/* ------------------------------------------------------------------ writing into a notebook */

// makeNotebook is the setup shortcut for the tests below: a notebook created through the RPC,
// optionally shared with one member at one permission.
func makeNotebook(t *testing.T, h *Handler, ctx context.Context, name string) string {
	t.Helper()
	resp, err := h.CreateNotebook(ctx, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: name,
	}))
	if err != nil {
		t.Fatalf("CreateNotebook(%q): %v", name, err)
	}
	return resp.Msg.GetNotebook().GetId()
}

func shareNotebook(
	t *testing.T, h *Handler, ctx context.Context, notebookID string,
	subject notesv1.ShareSubject, member string, perm notesv1.SharePermission,
) {
	t.Helper()
	if _, err := h.ShareNotebook(ctx, connect.NewRequest(&notesv1.ShareNotebookRequest{
		NotebookId: notebookID, Subject: subject, MemberUserId: member, Permission: perm,
	})); err != nil {
		t.Fatalf("ShareNotebook: %v", err)
	}
}

// A VIEW share on a notebook is a reading permit, not a write endpoint. The note a VIEW holder
// could otherwise file there is invisible to the notebook's owner and still blocks their
// delete, so the folder would be pinned by someone who was only ever shown it.
func TestViewNotebookShareCannotWriteIntoTheNotebook(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	nb := makeNotebook(t, h, ana, "Reading list")
	shareNotebook(t, h, ana, nb, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := h.CreateNote(ben, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb, Title: "mine now", Blocks: []*notesv1.Block{para("b1", "hello")},
	}))
	codeIs(t, err, connect.CodePermissionDenied, "CreateNote into a VIEW-shared notebook")

	// The same gate on the move path: Ben's own note may not be filed there either.
	own := createNote(t, h, ben, "Ben's note", para("b1", "hello"))
	_, err = h.MoveNote(ben, connect.NewRequest(&notesv1.MoveNoteRequest{
		NoteId: own.GetId(), NotebookId: nb,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "MoveNote into a VIEW-shared notebook")

	// An EDIT share is the permission that makes it a write endpoint, and it works.
	shareNotebook(t, h, ana, nb, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	if _, err := h.MoveNote(ben, connect.NewRequest(&notesv1.MoveNoteRequest{
		NoteId: own.GetId(), NotebookId: nb,
	})); err != nil {
		t.Fatalf("MoveNote into an EDIT-shared notebook: %v", err)
	}
}

// Moving is the owner's alone. Otherwise an EDIT share is a second, unlogged route to sharing:
// the holder files the note into a notebook of their own that the whole family can read, and
// the note's owner sees nothing in the share sheet and has no way to revoke it.
func TestMoveNoteCannotBeUsedToReshareSomeoneElsesNote(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	note := createNote(t, h, ana, "Divorce lawyers", para("b1", "call at four"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	// Ben's own notebook, shared with the whole family — all of it allowed, he owns it.
	bens := makeNotebook(t, h, ben, "Open house")
	shareNotebook(t, h, ben, bens, notesv1.ShareSubject_SHARE_SUBJECT_FAMILY, "",
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	_, err := h.MoveNote(ben, connect.NewRequest(&notesv1.MoveNoteRequest{
		NoteId: note.GetId(), NotebookId: bens,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "MoveNote on a note Ben does not own")

	// And the escalation it would have bought: Cal was never shared anything.
	if _, err := getNote(h, cal, note.GetId()); err == nil {
		t.Fatal("Cal can read Ana's note after Ben's move attempt")
	}
}

// The delete refusal must not name notes the caller cannot see: that count is the existence
// leak Notebook.note_count was written to avoid. The owner still gets a sentence that says
// what is wrong.
func TestDeleteNotebookRefusalNeverCountsInvisibleNotes(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	nb := makeNotebook(t, h, ana, "Shared shelf")
	shareNotebook(t, h, ana, nb, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	if _, err := h.CreateNote(ben, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb, Title: "Ben's", Blocks: []*notesv1.Block{para("b1", "private")},
	})); err != nil {
		t.Fatalf("CreateNote by the EDIT-share holder: %v", err)
	}

	// Ana cannot see Ben's note, and her sidebar count says so.
	got, err := h.ListNotes(ana, connect.NewRequest(&notesv1.ListNotesRequest{NotebookId: nb}))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if n := len(got.Msg.GetNotes()); n != 0 {
		t.Fatalf("ListNotes returned %d notes, want 0", n)
	}

	_, err = h.DeleteNotebook(ana, connect.NewRequest(&notesv1.DeleteNotebookRequest{
		NotebookId: nb,
	}))
	codeIs(t, err, connect.CodeFailedPrecondition, "DeleteNotebook with a foreign note inside")
	if msg := err.Error(); strings.ContainsAny(msg, "0123456789") {
		t.Errorf("refusal names a count of notes Ana cannot see: %q", msg)
	}

	// Her own note in her own notebook is hers to be told about, number and all.
	nb2 := makeNotebook(t, h, ana, "Private shelf")
	if _, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb2, Title: "Ana's", Blocks: []*notesv1.Block{para("b1", "mine")},
	})); err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	_, err = h.DeleteNotebook(ana, connect.NewRequest(&notesv1.DeleteNotebookRequest{
		NotebookId: nb2,
	}))
	codeIs(t, err, connect.CodeFailedPrecondition, "DeleteNotebook with the owner's own note")
	if !strings.Contains(err.Error(), "1 note(s)") {
		t.Errorf("refusal = %q, want the visible count in it", err.Error())
	}
}

// A note shared on its own out of a private notebook must not drag that notebook's NAME into
// the recipient's search palette: the name is user-authored content, and no share row covers
// it.
func TestSearchBreadcrumbNeverNamesAnInvisibleNotebook(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	nb := makeNotebook(t, h, ana, "Divorce lawyers")
	created, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb, Title: "Groceries", Blocks: []*notesv1.Block{para("b1", "groceries")},
	}))
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	shareNote(t, h, ana, created.Msg.GetNote().GetId(),
		notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	resp, err := h.Search(ben, connect.NewRequest(&notesv1.SearchRequest{Query: "groceries"}))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Msg.GetHits()) == 0 {
		t.Fatal("Search returned no hits; the shared note should be findable")
	}
	for _, hit := range resp.Msg.GetHits() {
		if strings.Contains(hit.GetContext(), "Divorce lawyers") {
			t.Errorf("hit context = %q, leaks the name of a notebook Ben cannot see",
				hit.GetContext())
		}
	}

	// Ana, who owns the notebook, still gets her breadcrumb.
	mine, err := h.Search(ana, connect.NewRequest(&notesv1.SearchRequest{Query: "groceries"}))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(mine.Msg.GetHits()) == 0 || mine.Msg.GetHits()[0].GetContext() != "Divorce lawyers" {
		t.Errorf("owner's breadcrumb = %q, want the notebook name",
			mine.Msg.GetHits()[0].GetContext())
	}
}

/* ------------------------------------------------------------------ text bounds */

// Free text is bounded in runes, not bytes: a Ukrainian note must not be worth half an English
// one, and the message says "characters".
func TestTextBoundsCountRunesNotBytes(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	// maxBlockText runes of Cyrillic is twice that many bytes, and must be accepted.
	long := strings.Repeat("ї", maxBlockText)
	note := createNote(t, h, ana, "Рецепт", para("b1", long))

	if _, err := h.AddComment(ana, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: note.GetId(), Body: long,
	})); err != nil {
		t.Fatalf("AddComment with %d runes: %v", maxBlockText, err)
	}

	_, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: "too long", Blocks: []*notesv1.Block{para("b1", long+"ї")},
	}))
	codeIs(t, err, connect.CodeInvalidArgument, "CreateNote one rune over the block bound")

	// And the two single-line fields are bounded at all, which they were not before.
	_, err = h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		Title: strings.Repeat("я", maxTitle+1),
	}))
	codeIs(t, err, connect.CodeInvalidArgument, "CreateNote with an over-long title")

	_, err = h.CreateNotebook(ana, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: strings.Repeat("я", maxNotebookName+1),
	}))
	codeIs(t, err, connect.CodeInvalidArgument, "CreateNotebook with an over-long name")
}

/* ------------------------------------------------------------------ archiving */

// listNotes is the shortcut the archive tests read through; it returns the titles, because
// what these assert is which notes a filter admits, not what a row looks like.
func listNotes(
	t *testing.T, h *Handler, ctx context.Context, req *notesv1.ListNotesRequest,
) []string {
	t.Helper()
	resp, err := h.ListNotes(ctx, connect.NewRequest(req))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	out := make([]string, 0, len(resp.Msg.GetNotes()))
	for _, n := range resp.Msg.GetNotes() {
		out = append(out, n.GetTitle())
	}
	return out
}

func archiveNote(
	h *Handler, ctx context.Context, noteID string, archived bool,
) (*notesv1.Note, error) {
	resp, err := h.ArchiveNote(ctx, connect.NewRequest(&notesv1.ArchiveNoteRequest{
		NoteId: noteID, Archived: archived,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetNote(), nil
}

// Archiving is a write on the shared row, so it takes EDIT — and a caller who cannot see the
// note at all is told NotFound, not that they lack permission on something that exists.
func TestArchiveNoteNeedsEdit(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := archiveNote(h, cal, note.GetId(), true)
	codeIs(t, err, connect.CodeNotFound, "ArchiveNote by someone who cannot see the note")

	_, err = archiveNote(h, ben, note.GetId(), true)
	codeIs(t, err, connect.CodePermissionDenied, "ArchiveNote by a VIEW-share holder")

	// The refusal has to be a refusal, not just an error: the note is still live.
	got, err := getNote(h, ana, note.GetId())
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.GetArchived() {
		t.Error("note is archived after a refused ArchiveNote")
	}

	// An EDIT share is the permission that carries it. Archiving loses nothing and re-shares
	// nothing, so it is not owner-only the way deleting and moving are.
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	updated, err := archiveNote(h, ben, note.GetId(), true)
	if err != nil {
		t.Fatalf("ArchiveNote by an EDIT-share holder: %v", err)
	}
	if !updated.GetArchived() {
		t.Error("archived = false on the note ArchiveNote returned")
	}
}

// The statement carries the write predicate itself, not only the handler. Called straight on
// the querier — past the handler's gate, the way a future caller that forgot it would — a
// VIEW-share holder must still write no row.
func TestSetNoteArchivedStatementRefusesAViewer(t *testing.T) {
	h, store, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := store.SetNoteArchived(context.Background(), db.SetNoteArchivedParams{
		Archived: true,
		NoteID:   pgconv.MustUUID(note.GetId()),
		FamilyID: pgconv.MustUUID(familyOne),
		UserID:   pgconv.MustUUID(userBen),
	})
	if err == nil {
		t.Fatal("SetNoteArchived wrote a row for a VIEW-share holder")
	}
	// The owner is who it is for, and that path still works.
	if _, err := store.SetNoteArchived(context.Background(), db.SetNoteArchivedParams{
		Archived: true,
		NoteID:   pgconv.MustUUID(note.GetId()),
		FamilyID: pgconv.MustUUID(familyOne),
		UserID:   pgconv.MustUUID(userAna),
	}); err != nil {
		t.Fatalf("SetNoteArchived by the owner: %v", err)
	}
}

// Starring is the same kind of write as archiving — one column on the shared row, changed for
// everyone who can see the note — so its statement carries the same predicate. Called straight
// on the querier, past the handler's mustEditNote gate the way a future caller that forgot it
// would, a VIEW-share holder must still write no row.
func TestSetNoteStarredStatementRefusesAViewer(t *testing.T) {
	h, store, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err := store.SetNoteStarred(context.Background(), db.SetNoteStarredParams{
		Starred:  true,
		NoteID:   pgconv.MustUUID(note.GetId()),
		FamilyID: pgconv.MustUUID(familyOne),
		UserID:   pgconv.MustUUID(userBen),
	})
	if err == nil {
		t.Fatal("SetNoteStarred wrote a row for a VIEW-share holder")
	}
	// The owner is who it is for, and that path still works.
	if _, err := store.SetNoteStarred(context.Background(), db.SetNoteStarredParams{
		Starred:  true,
		NoteID:   pgconv.MustUUID(note.GetId()),
		FamilyID: pgconv.MustUUID(familyOne),
		UserID:   pgconv.MustUUID(userAna),
	}); err != nil {
		t.Fatalf("SetNoteStarred by the owner: %v", err)
	}
	// And so does an EDIT share: the predicate is "may write", not "owns" — starring is not
	// owner-only the way moving and deleting are.
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userCal,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	if _, err := store.SetNoteStarred(context.Background(), db.SetNoteStarredParams{
		Starred:  false,
		NoteID:   pgconv.MustUUID(note.GetId()),
		FamilyID: pgconv.MustUUID(familyOne),
		UserID:   pgconv.MustUUID(userCal),
	}); err != nil {
		t.Fatalf("SetNoteStarred by an EDIT-share holder: %v", err)
	}
}

// Archiving moves a note between lists and back, and does it without touching the version:
// filing is not editing, and a bumped version would make everyone else's next save conflict.
func TestArchivedNoteLeavesTheDefaultListAndComesBack(t *testing.T) {
	h, _, rec := newTestHandler()
	ana := withClaims(userAna, familyOne)

	createNote(t, h, ana, "Live", para("b1", "current"))
	old := createNote(t, h, ana, "Old", para("b1", "done with"))

	archived, err := archiveNote(h, ana, old.GetId(), true)
	if err != nil {
		t.Fatalf("ArchiveNote: %v", err)
	}
	if archived.GetVersion() != old.GetVersion() {
		t.Errorf("version = %d after archiving, want %d (archiving is filing, not editing)",
			archived.GetVersion(), old.GetVersion())
	}
	// It moved between lists for everyone who can see it, which is what the event says.
	if !rec.sawSubject(fmevents.SubjectNotesNoteUpdated) {
		t.Error("ArchiveNote published no notes.note.updated")
	}

	if got := listNotes(t, h, ana, &notesv1.ListNotesRequest{}); len(got) != 1 || got[0] != "Live" {
		t.Errorf("default list = %v, want [Live]", got)
	}
	if got := listNotes(t, h, ana, &notesv1.ListNotesRequest{ArchivedOnly: true}); len(got) != 1 ||
		got[0] != "Old" {
		t.Errorf("archived_only list = %v, want [Old]", got)
	}
	if got := listNotes(t, h, ana, &notesv1.ListNotesRequest{IncludeArchived: true}); len(got) != 2 {
		t.Errorf("include_archived list = %v, want both notes", got)
	}

	if _, err := archiveNote(h, ana, old.GetId(), false); err != nil {
		t.Fatalf("ArchiveNote(false): %v", err)
	}
	if got := listNotes(t, h, ana, &notesv1.ListNotesRequest{ArchivedOnly: true}); len(got) != 0 {
		t.Errorf("archived_only list after un-archiving = %v, want empty", got)
	}
	if got := listNotes(t, h, ana, &notesv1.ListNotesRequest{}); len(got) != 2 {
		t.Errorf("default list after un-archiving = %v, want both notes", got)
	}
}

// Why archived_only is a server filter and not include_archived with the live notes dropped in
// the client: the page is cut server-side. Ask for one page of everything and the page is the
// live note; drop the live notes from it and the archive renders empty while holding notes.
func TestArchivedOnlyIsNotAClientSideFilterOfIncludeArchived(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	old := createNote(t, h, ana, "Old", para("b1", "done with"))
	if _, err := archiveNote(h, ana, old.GetId(), true); err != nil {
		t.Fatalf("ArchiveNote: %v", err)
	}
	// Created after, so it sorts first on updated_at and takes the only slot in the page.
	createNote(t, h, ana, "Live", para("b1", "current"))

	page := listNotes(t, h, ana, &notesv1.ListNotesRequest{IncludeArchived: true, PageSize: 1})
	if len(page) != 1 || page[0] != "Live" {
		t.Fatalf("include_archived page = %v, want [Live] (the setup this test is about)", page)
	}

	got := listNotes(t, h, ana, &notesv1.ListNotesRequest{ArchivedOnly: true, PageSize: 1})
	if len(got) != 1 || got[0] != "Old" {
		t.Errorf("archived_only page = %v, want [Old]", got)
	}
}

/* ------------------------------------------- comments and activity, by SQL alone */

// ListComments and ListActivity have no gate in the handler at all: the visibility predicate
// in the statement is the whole authority rule, and nothing asserted it. A caller who cannot
// see the note must get nothing from either — an empty list, not an error, because the note's
// existence is what is being withheld.
func TestCommentsAndActivityNeverReachACallerWhoCannotSeeTheNote(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	cal := withClaims(userCal, familyOne)
	dana := withClaims(userDana, familyTwo)

	note := createNote(t, h, ana, "Therapy notes", para("b1", "not for the family"))
	if _, err := h.AddComment(ana, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: note.GetId(), Body: "a thought",
	})); err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	// The owner sees both, so the test is about who is refused and not about empty tables.
	comments, err := h.ListComments(ana, connect.NewRequest(&notesv1.ListCommentsRequest{
		NoteId: note.GetId(),
	}))
	if err != nil {
		t.Fatalf("ListComments by the owner: %v", err)
	}
	if len(comments.Msg.GetComments()) != 1 {
		t.Fatalf("owner sees %d comments, want 1", len(comments.Msg.GetComments()))
	}
	activity, err := h.ListActivity(ana, connect.NewRequest(&notesv1.ListActivityRequest{
		NoteId: note.GetId(),
	}))
	if err != nil {
		t.Fatalf("ListActivity by the owner: %v", err)
	}
	if len(activity.Msg.GetActivity()) == 0 {
		t.Fatal("owner sees no activity, want the created and commented rows")
	}

	// A family member who was never shared the note, and an outsider in another family.
	for _, tc := range []struct {
		name string
		ctx  context.Context
	}{
		{"a family member with no share", cal},
		{"a member of another family", dana},
	} {
		got, err := h.ListComments(tc.ctx, connect.NewRequest(&notesv1.ListCommentsRequest{
			NoteId: note.GetId(), IncludeResolved: true,
		}))
		if err != nil {
			t.Fatalf("ListComments by %s: %v", tc.name, err)
		}
		if n := len(got.Msg.GetComments()); n != 0 {
			t.Errorf("ListComments by %s returned %d comments, want 0", tc.name, n)
		}

		rail, err := h.ListActivity(tc.ctx, connect.NewRequest(&notesv1.ListActivityRequest{
			NoteId: note.GetId(),
		}))
		if err != nil {
			t.Fatalf("ListActivity by %s: %v", tc.name, err)
		}
		if n := len(rail.Msg.GetActivity()); n != 0 {
			t.Errorf("ListActivity by %s returned %d rows, want 0", tc.name, n)
		}
	}
}

/* ------------------------------------------------------------------ notebook tree */

func updateNotebook(
	h *Handler, ctx context.Context, notebookID, name, parentID string,
) error {
	_, err := h.UpdateNotebook(ctx, connect.NewRequest(&notesv1.UpdateNotebookRequest{
		NotebookId: notebookID, Name: name, ParentId: parentID,
	}))
	return err
}

// Nesting a notebook under a parent is a write to that parent — the same act CreateNote and
// MoveNote gate with EDIT one level down. Visibility was enough here, which made a VIEW share
// a way to hang a folder inside somebody else's folder.
func TestNestingANotebookNeedsEditOnTheParent(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	shelf := makeNotebook(t, h, ana, "Shelf")
	bens := makeNotebook(t, h, ben, "Ben's")

	// Not shared at all: NotFound, because a notebook Ben cannot see must not confirm it exists.
	_, err := h.CreateNotebook(ben, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: "Sneaked in", ParentId: shelf,
	}))
	codeIs(t, err, connect.CodeNotFound, "CreateNotebook under an invisible parent")

	shareNotebook(t, h, ana, shelf, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	_, err = h.CreateNotebook(ben, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: "Sneaked in", ParentId: shelf,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "CreateNotebook under a VIEW-shared parent")

	err = updateNotebook(h, ben, bens, "Ben's", shelf)
	codeIs(t, err, connect.CodePermissionDenied, "UpdateNotebook re-parenting under a VIEW share")

	// EDIT is what makes the parent writable, and then both paths work.
	shareNotebook(t, h, ana, shelf, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	if _, err := h.CreateNotebook(ben, connect.NewRequest(&notesv1.CreateNotebookRequest{
		Name: "Invited in", ParentId: shelf,
	})); err != nil {
		t.Fatalf("CreateNotebook under an EDIT-shared parent: %v", err)
	}
	if err := updateNotebook(h, ben, bens, "Ben's", shelf); err != nil {
		t.Fatalf("UpdateNotebook re-parenting under an EDIT share: %v", err)
	}
}

// Refusing only a direct self-parent left two moves, each legal alone, that close a loop.
// Nothing server-side recurses over the tree, so the server never noticed — but the app builds
// its sidebar from parent_id, and a cycle there is a render that does not terminate.
func TestUpdateNotebookRefusesACycle(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	a := makeNotebook(t, h, ana, "A")
	b := makeNotebook(t, h, ana, "B")
	c := makeNotebook(t, h, ana, "C")

	codeIs(t, updateNotebook(h, ana, a, "A", a), connect.CodeInvalidArgument,
		"UpdateNotebook making a notebook its own parent")

	// A legal chain first: C under B under A.
	if err := updateNotebook(h, ana, b, "B", a); err != nil {
		t.Fatalf("UpdateNotebook B under A: %v", err)
	}
	if err := updateNotebook(h, ana, c, "C", b); err != nil {
		t.Fatalf("UpdateNotebook C under B: %v", err)
	}

	// Closing it at either distance is the same refusal.
	codeIs(t, updateNotebook(h, ana, a, "A", b), connect.CodeInvalidArgument,
		"UpdateNotebook closing a two-node cycle")
	codeIs(t, updateNotebook(h, ana, a, "A", c), connect.CodeInvalidArgument,
		"UpdateNotebook closing a three-node cycle")

	// The refusal did not half-apply, and it is a fact about the chain rather than about the
	// pair: break A out of C's ancestry and the same move is allowed.
	if err := updateNotebook(h, ana, b, "B", ""); err != nil {
		t.Fatalf("UpdateNotebook B back to the top level: %v", err)
	}
	if err := updateNotebook(h, ana, a, "A", c); err != nil {
		t.Fatalf("UpdateNotebook A under C once A is no longer C's ancestor: %v", err)
	}
}

/* ------------------------------- a notebook share reaches its OWNER's notes only */

// notebookLeak is the shape of the hole this section exists to close: Ana's notebook, an EDIT
// share on it so Ben can file notes there, one note from each of them inside, and then Ana
// sharing the whole notebook with the family. Before the rule was fixed, that last call
// published Ben's note to everyone — and handed Ana, who merely owns the folder, read and
// write over it.
type notebookLeak struct {
	h        *Handler
	notebook string
	anas     *notesv1.Note // Ana's own note, in Ana's notebook
	bens     *notesv1.Note // Ben's note, in Ana's notebook, under his EDIT share
}

func newNotebookLeak(t *testing.T, family notesv1.SharePermission) notebookLeak {
	t.Helper()
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	nb := makeNotebook(t, h, ana, "Kitchen table")
	shareNotebook(t, h, ana, nb, notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_EDIT)

	anas, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb, Title: "Groceries",
		Blocks: []*notesv1.Block{para("b1", "milk and eggs")},
	}))
	if err != nil {
		t.Fatalf("CreateNote by the notebook's owner: %v", err)
	}
	bens, err := h.CreateNote(ben, connect.NewRequest(&notesv1.CreateNoteRequest{
		NotebookId: nb, Title: "Ben's own",
		Blocks: []*notesv1.Block{para("b1", "the surprise is on friday")},
	}))
	if err != nil {
		t.Fatalf("CreateNote by the EDIT-share holder: %v", err)
	}

	// A comment of Ben's on his own note, so that the margins have something to leak.
	if _, err := h.AddComment(ben, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: bens.Msg.GetNote().GetId(), Body: "cake ordered",
	})); err != nil {
		t.Fatalf("AddComment by the note's owner: %v", err)
	}

	// The call that used to publish somebody else's note.
	shareNotebook(t, h, ana, nb, notesv1.ShareSubject_SHARE_SUBJECT_FAMILY, "", family)

	return notebookLeak{h: h, notebook: nb, anas: anas.Msg.GetNote(), bens: bens.Msg.GetNote()}
}

func hasNote(notes []*notesv1.Note, id string) bool {
	for _, n := range notes {
		if n.GetId() == id {
			return true
		}
	}
	return false
}

// Sharing a notebook shares the notes ITS OWNER put in it. A note somebody else filed there is
// not the sharer's to give away, so no share of the folder can reach it — not for the rest of
// the family, and not for the folder's own owner, who would otherwise grant herself read and
// write over a family member's private note by sharing her own notebook.
func TestNotebookShareNeverReachesANoteItsOwnerDidNotFile(t *testing.T) {
	f := newNotebookLeak(t, notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	// Carol, who was given nothing but the family share on Ana's notebook.
	if _, err := getNote(f.h, cal, f.bens.GetId()); err == nil {
		t.Fatal("Cal can read Ben's note through a family share of Ana's notebook")
	} else {
		codeIs(t, err, connect.CodeNotFound, "GetNote on a note filed by someone else")
	}

	// And the notebook's owner, who is the one holding the sharing pen.
	if _, err := getNote(f.h, ana, f.bens.GetId()); err == nil {
		t.Fatal("Ana granted herself Ben's note by sharing the notebook it sits in")
	} else {
		codeIs(t, err, connect.CodeNotFound, "GetNote by the notebook's owner")
	}
	_, err := f.h.UpdateNote(ana, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: f.bens.GetId(), Title: "Ana's now",
		Blocks: []*notesv1.Block{para("b1", "rewritten")},
	}))
	codeIs(t, err, connect.CodeNotFound, "UpdateNote by the notebook's owner")

	// What the share is FOR still works: Ana's own notes in the folder reach the family.
	got, err := getNote(f.h, cal, f.anas.GetId())
	if err != nil {
		t.Fatalf("GetNote on the sharer's own note in the shared notebook: %v", err)
	}
	if !got.GetCanEdit() {
		t.Error("can_edit = false under an EDIT notebook share of the owner's own note")
	}
	// And Ben still has his own note, which is the other half of "nothing else changed".
	if _, err := getNote(f.h, ben, f.bens.GetId()); err != nil {
		t.Fatalf("GetNote by the note's own owner: %v", err)
	}

	list, err := f.h.ListNotes(cal, connect.NewRequest(&notesv1.ListNotesRequest{}))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if hasNote(list.Msg.GetNotes(), f.bens.GetId()) {
		t.Error("ListNotes shows Cal a note filed into Ana's notebook by Ben")
	}
	if !hasNote(list.Msg.GetNotes(), f.anas.GetId()) {
		t.Error("ListNotes hides the sharer's own note; the share bought nothing")
	}
}

// The same rule, on every other surface that reaches a note through its notebook. One of them
// forgetting is the whole leak back, so they are asserted together rather than trusted to the
// one predicate being copied correctly.
func TestNotebookShareLeakIsClosedOnEverySurface(t *testing.T) {
	f := newNotebookLeak(t, notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)
	cal := withClaims(userCal, familyOne)

	// Search, both facets — the palette is the quietest way to read a note you cannot open.
	hits, err := f.h.Search(cal, connect.NewRequest(&notesv1.SearchRequest{
		Query: "surprise", Facet: notesv1.SearchFacet_SEARCH_FACET_ALL,
	}))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, hit := range hits.Msg.GetHits() {
		if hit.GetNoteId() == f.bens.GetId() {
			t.Errorf("search hit reaches a note the notebook share must not carry: %v", hit)
		}
	}
	// The corpus count is the same predicate, so it must not count it either: Cal can see
	// Ana's note and nothing else.
	if n := hits.Msg.GetSearchedNotes(); n != 1 {
		t.Errorf("searched_notes = %d, want 1 (the sharer's own note only)", n)
	}

	// "Shared with me" — the rail where a leaked note would be most visible of all.
	shared, err := f.h.ListSharedWithMe(cal, connect.NewRequest(&notesv1.ListSharedWithMeRequest{}))
	if err != nil {
		t.Fatalf("ListSharedWithMe: %v", err)
	}
	if hasNote(shared.Msg.GetNotes(), f.bens.GetId()) {
		t.Error("ListSharedWithMe offers Cal a note filed into Ana's notebook by Ben")
	}
	if !hasNote(shared.Msg.GetNotes(), f.anas.GetId()) {
		t.Error("ListSharedWithMe hides the sharer's own note")
	}

	// note_count is the caller's count, and Ben's note is not the caller's.
	books, err := f.h.ListNotebooks(cal, connect.NewRequest(&notesv1.ListNotebooksRequest{}))
	if err != nil {
		t.Fatalf("ListNotebooks: %v", err)
	}
	for _, nb := range books.Msg.GetNotebooks() {
		if nb.GetId() == f.notebook && nb.GetNoteCount() != 1 {
			t.Errorf("note_count = %d for the shared notebook, want 1", nb.GetNoteCount())
		}
	}

	// The margins: a comment thread and the activity rail are read access to the note's
	// content by another name.
	comments, err := f.h.ListComments(cal, connect.NewRequest(&notesv1.ListCommentsRequest{
		NoteId: f.bens.GetId(), IncludeResolved: true,
	}))
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if n := len(comments.Msg.GetComments()); n != 0 {
		t.Errorf("ListComments returned %d rows on a note Cal cannot see, want 0", n)
	}
	rail, err := f.h.ListActivity(cal, connect.NewRequest(&notesv1.ListActivityRequest{
		NoteId: f.bens.GetId(),
	}))
	if err != nil {
		t.Fatalf("ListActivity: %v", err)
	}
	if n := len(rail.Msg.GetActivity()); n != 0 {
		t.Errorf("ListActivity returned %d rows on a note Cal cannot see, want 0", n)
	}
	// The writes an EDIT notebook share would otherwise have granted over it.
	_, err = f.h.ToggleStar(cal, connect.NewRequest(&notesv1.ToggleStarRequest{
		NoteId: f.bens.GetId(), Starred: true,
	}))
	codeIs(t, err, connect.CodeNotFound, "ToggleStar on a note filed by someone else")
	_, err = f.h.ArchiveNote(ana, connect.NewRequest(&notesv1.ArchiveNoteRequest{
		NoteId: f.bens.GetId(), Archived: true,
	}))
	codeIs(t, err, connect.CodeNotFound, "ArchiveNote by the notebook's owner")
	_, err = f.h.AddComment(cal, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: f.bens.GetId(), Body: "so it is friday",
	}))
	codeIs(t, err, connect.CodeNotFound, "AddComment on a note the notebook share must not carry")

	// Ben's own note is untouched by all of it.
	if _, err := getNote(f.h, ben, f.bens.GetId()); err != nil {
		t.Fatalf("GetNote by the note's owner: %v", err)
	}
}

// `shared` is the glyph the list row draws, and the owner reads it as "somebody else can see
// this". It must therefore count the notebook share too — and only where that share reaches,
// which is the notebook owner's own notes.
func TestSharedFlagCountsTheNotebookShareAndNothingElse(t *testing.T) {
	f := newNotebookLeak(t, notesv1.SharePermission_SHARE_PERMISSION_VIEW)
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	anas, err := getNote(f.h, ana, f.anas.GetId())
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if !anas.GetShared() {
		t.Error("shared = false on a note the whole family can read through its notebook")
	}

	// Ben's note carries no share of its own and its notebook's share does not reach it, so
	// nobody but Ben can read it — and telling him otherwise would be the same lie inverted.
	bens, err := getNote(f.h, ben, f.bens.GetId())
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if bens.GetShared() {
		t.Error("shared = true on a note nobody but its owner can reach")
	}

	// The list row is where the glyph is actually drawn, so it is asserted there too.
	list, err := f.h.ListNotes(ana, connect.NewRequest(&notesv1.ListNotesRequest{
		NotebookId: f.notebook,
	}))
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(list.Msg.GetNotes()) != 1 {
		t.Fatalf("Ana sees %d notes in her notebook, want 1", len(list.Msg.GetNotes()))
	}
	if !list.Msg.GetNotes()[0].GetShared() {
		t.Error("ListNotes row says private for a note the family can read")
	}
}

/* ------------------------------------------------------ UpdateNote's two guards */

// UpdateNote used to be the one write that carried no permission predicate in SQL. Called
// straight on the querier, past the handler's gate the way a future caller that forgot it
// would, a VIEW-share holder must now write no row — the same assertion SetNoteStarred and
// SetNoteArchived already carry.
func TestUpdateNoteStatementRefusesAViewer(t *testing.T) {
	h, store, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)

	note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))
	shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	params := func(user string) db.UpdateNoteParams {
		return db.UpdateNoteParams{
			Title: "rewritten", Blocks: []byte("[]"), BodyText: "",
			UserID:   pgconv.MustUUID(user),
			NoteID:   pgconv.MustUUID(note.GetId()),
			FamilyID: pgconv.MustUUID(familyOne),
		}
	}
	if _, err := store.UpdateNote(context.Background(), params(userBen)); err == nil {
		t.Fatal("UpdateNote wrote a row for a VIEW-share holder")
	}
	if _, err := store.UpdateNote(context.Background(), params(userAna)); err != nil {
		t.Fatalf("UpdateNote by the owner: %v", err)
	}
}

// racingStore fires one callback the moment the handler has read a note, standing in for the
// commit that lands between UpdateNote's pre-flight read and its write. It is how both of
// UpdateNote's guards get exercised at the statement rather than at the gate, which is the
// only place the two refusals are indistinguishable.
type racingStore struct {
	*fakeStore
	once func()
}

func (r *racingStore) GetNote(ctx context.Context, arg db.GetNoteParams) (db.GetNoteRow, error) {
	row, err := r.fakeStore.GetNote(ctx, arg)
	if err == nil && r.once != nil {
		fire := r.once
		r.once = nil
		fire()
	}
	return row, err
}

// A statement that wrote no row does not say why, and the two reasons are different rpc
// errors: ABORTED means "retry with the current version", PermissionDenied means "stop". The
// handler must not read one as the other.
func TestUpdateNoteTellsAVersionConflictFromARefusal(t *testing.T) {
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	setup := func(t *testing.T, race func(*fakeStore, string)) (*Handler, string) {
		t.Helper()
		store := newFakeStore()
		racer := &racingStore{fakeStore: store}
		h := New(Options{Queries: racer, Tx: store, Bus: &recorder{}})
		note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))
		shareNote(t, h, ana, note.GetId(), notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
			notesv1.SharePermission_SHARE_PERMISSION_EDIT)
		racer.once = func() { race(store, note.GetId()) }
		return h, note.GetId()
	}

	// Ana downgrades Ben's share to VIEW after his editor has loaded the note. The version
	// guard is not in play (expected_version 0 forces the write), so the only thing that can
	// refuse is the predicate, and the answer must say so.
	h, noteID := setup(t, func(store *fakeStore, id string) {
		for k, sh := range store.noteShares {
			if sameUUID(sh.NoteID, pgconv.MustUUID(id)) {
				sh.Permission = 1
				store.noteShares[k] = sh
			}
		}
	})
	_, err := h.UpdateNote(ben, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: noteID, Title: "Ben's edit",
		Blocks: []*notesv1.Block{para("b1", "ship it on friday")}, ExpectedVersion: 0,
	}))
	codeIs(t, err, connect.CodePermissionDenied, "UpdateNote after the share was downgraded")

	// Ana commits her own edit in the same window. Ben may still write, so the refusal is the
	// lost race, and the message names the version he now has to merge onto.
	h, noteID = setup(t, func(store *fakeStore, id string) {
		n := store.notes[id]
		n.Version += 2
		store.notes[id] = n
	})
	_, err = h.UpdateNote(ben, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: noteID, Title: "Ben's edit",
		Blocks: []*notesv1.Block{para("b1", "ship it on friday")}, ExpectedVersion: 1,
	}))
	codeIs(t, err, connect.CodeAborted, "UpdateNote after somebody else committed")
	if !strings.Contains(err.Error(), "version 3") {
		t.Errorf("conflict message = %q, want the version the writer must merge onto", err)
	}
}

/* ------------------------------------------------------------------ image blocks */

// v1 exposes no image blocks and runs with no object store, so UploadNoteImage is a procedure
// this deployment does not implement — not an incident, and not a precondition anybody can go
// and satisfy. It must answer Unimplemented, and it must answer before it looks a note up:
// a procedure that does not exist has no business confirming that someone else's note does.
func TestUploadNoteImageIsUnimplementedWithoutStorage(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note := createNote(t, h, ana, "Roadmap", para("b1", "ship the editor"))

	_, err := h.UploadNoteImage(ana, connect.NewRequest(&notesv1.UploadNoteImageRequest{
		NoteId: note.GetId(), ContentType: "image/png", Image: []byte("not really a png"),
	}))
	codeIs(t, err, connect.CodeUnimplemented, "UploadNoteImage with no storage configured")

	// Same answer for a caller who cannot see the note, and for a note that does not exist:
	// the refusal is about the deployment, so it leaks nothing about either.
	_, err = h.UploadNoteImage(ben, connect.NewRequest(&notesv1.UploadNoteImageRequest{
		NoteId: note.GetId(), ContentType: "image/png", Image: []byte("not really a png"),
	}))
	codeIs(t, err, connect.CodeUnimplemented, "UploadNoteImage by a caller who cannot see the note")
	_, err = h.UploadNoteImage(ana, connect.NewRequest(&notesv1.UploadNoteImageRequest{
		NoteId: "", ContentType: "", Image: nil,
	}))
	codeIs(t, err, connect.CodeUnimplemented, "UploadNoteImage with no arguments at all")
}

// Sharing a notebook writes an ACTIVITY_KIND_SHARED row onto the notes inside it. The rule
// from notes.sql applies to that insert too: the share reaches only the notes the notebook's
// owner filed there, so only those get the row. Without the rule the leak comes back in a
// quieter shape — Ben opens his own private note and its rail tells him Ana shared it with the
// family, naming a person who cannot in fact read it.
func TestNotebookShareActivityStaysOnTheOwnersOwnNotes(t *testing.T) {
	f := newNotebookLeak(t, notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	sharedRows := func(ctx context.Context, noteID, who string) int {
		t.Helper()
		resp, err := f.h.ListActivity(ctx, connect.NewRequest(&notesv1.ListActivityRequest{
			NoteId: noteID,
		}))
		if err != nil {
			t.Fatalf("ListActivity for %s: %v", who, err)
		}
		n := 0
		for _, a := range resp.Msg.GetActivity() {
			if a.GetKind() == notesv1.ActivityKind_ACTIVITY_KIND_SHARED {
				n++
			}
		}
		return n
	}

	if got := sharedRows(ben, f.bens.GetId(), "the note's own owner"); got != 0 {
		t.Errorf("Ben's private note carries %d share row(s) from a share of Ana's notebook", got)
	}
	if got := sharedRows(ana, f.anas.GetId(), "the notebook's owner"); got == 0 {
		t.Error("the sharer's own note in the shared notebook records no share at all")
	}
}

// Authorship is not access. A comment outlives the share that produced it, so ResolveComment
// ANDs the author/owner authority with the visibility predicate: once Ana revokes the share,
// Ben can no longer reach into her note to flip a row in it, not even one he wrote.
func TestResolveCommentNeedsVisibilityNotJustAuthorship(t *testing.T) {
	h, _, _ := newTestHandler()
	ana := withClaims(userAna, familyOne)
	ben := withClaims(userBen, familyOne)

	note, err := h.CreateNote(ana, connect.NewRequest(&notesv1.CreateNoteRequest{
		Title:  "Party",
		Blocks: []*notesv1.Block{para("b1", "who is bringing what")},
	}))
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	noteID := note.Msg.GetNote().GetId()
	share := shareNote(t, h, ana, noteID,
		notesv1.ShareSubject_SHARE_SUBJECT_MEMBER, userBen,
		notesv1.SharePermission_SHARE_PERMISSION_VIEW)

	comment, err := h.AddComment(ben, connect.NewRequest(&notesv1.AddCommentRequest{
		NoteId: noteID, Body: "I have the cake",
	}))
	if err != nil {
		t.Fatalf("AddComment by the sharee: %v", err)
	}
	commentID := comment.Msg.GetComment().GetId()

	// While the share stands, the author closes his own thread.
	if _, err := h.ResolveComment(ben, connect.NewRequest(&notesv1.ResolveCommentRequest{
		CommentId: commentID, Resolved: true,
	})); err != nil {
		t.Fatalf("ResolveComment by the author while shared: %v", err)
	}

	if _, err := h.Unshare(ana, connect.NewRequest(&notesv1.UnshareRequest{
		NoteId: noteID, ShareId: share.GetId(),
	})); err != nil {
		t.Fatalf("Unshare: %v", err)
	}

	_, err = h.ResolveComment(ben, connect.NewRequest(&notesv1.ResolveCommentRequest{
		CommentId: commentID, Resolved: false,
	}))
	codeIs(t, err, connect.CodeNotFound, "ResolveComment by an author whose share was revoked")

	// The note's owner is unaffected: she satisfies both halves of the predicate.
	if _, err := h.ResolveComment(ana, connect.NewRequest(&notesv1.ResolveCommentRequest{
		CommentId: commentID, Resolved: false,
	})); err != nil {
		t.Fatalf("ResolveComment by the note's owner: %v", err)
	}
}

// Package handler implements notes.v1.NotesService over Connect (and gRPC, from the same
// type). Authorization lives here: the caller's identity and family_id come from the verified
// access token, never from the request body.
//
// The unit of privacy is the note. Reads run the visibility predicate that lives in the
// queries, so a note the caller cannot see comes back as no row and is answered NotFound —
// never PermissionDenied, which would confirm the note exists. Writes are the same walk with
// permission = EDIT: a note the caller can see but not write is PermissionDenied, because at
// that point they already know it is there.
package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	fmevents "github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/libs/go/rpc"
	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"
	"github.com/nnc/family-manager/services/notes/db"
	notesevents "github.com/nnc/family-manager/services/notes/internal/events"
)

// ImageStore is the slice of libs/go/storage this service uses. Narrow on purpose, same as
// EventBus: tests pass a fake instead of standing up MinIO.
type ImageStore interface {
	Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) (string, error)
}

// Tx is the transaction boundary. Implemented by internal/store over a pgx pool, and by the
// fake store in tests.
//
// It takes a callback rather than returning a transaction handle so that there is no way to
// begin one and forget to finish it, and so the Querier bound to the transaction is the only
// one in scope while it is open.
type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Handler serves notes.v1.NotesService.
type Handler struct {
	q           db.Querier
	tx          Tx
	bus         EventBus
	images      ImageStore
	imageBucket string
	// imagesEnabled records whether a real object store was wired in. It cannot be read off
	// h.images, which New fills with noopImages precisely so the rest of the type has one
	// code path, and UploadNoteImage needs the distinction: no store is a permanent refusal
	// in v1, a store that fails is an incident.
	imagesEnabled bool
	log           *slog.Logger
	now           func() time.Time
}

// Options configures a Handler. Only Queries is required.
type Options struct {
	Queries db.Querier
	// Tx groups the writes that must not half-apply — a note and the activity row that
	// records it. Nil means every write runs on its own, which is what a zero-valued
	// Options in a test gets.
	Tx  Tx
	Bus EventBus
	// Images and ImageBucket back UploadNoteImage. Leaving Images nil makes that one RPC
	// answer Unimplemented instead of every other procedure refusing to start — a MinIO
	// outage shouldn't take down note reads and writes any more than a NATS outage does.
	// Nil is also the v1 deployment: no image blocks, no bucket, see the rpc's proto comment.
	Images      ImageStore
	ImageBucket string
	Log         *slog.Logger
	// Now is injected by tests so timestamps are deterministic.
	Now func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:             opts.Queries,
		tx:            opts.Tx,
		bus:           opts.Bus,
		images:        opts.Images,
		imageBucket:   opts.ImageBucket,
		imagesEnabled: opts.Images != nil,
		log:           opts.Log,
		now:           opts.Now,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.now == nil {
		h.now = time.Now
	}
	if h.tx == nil {
		h.tx = withoutTx{h.q}
	}
	if h.bus == nil {
		h.bus = noopBus{}
	}
	if h.images == nil {
		h.images = noopImages{}
	}
	return h
}

// withoutTx runs a "transaction" as plain sequential queries. It exists so the handler has
// one code path whether or not a pool was wired in.
type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(ctx context.Context, fn func(q db.Querier) error) error { return fn(w.q) }

// noopImages lets the service run (and every non-image test pass) with no MinIO attached.
// UploadNoteImage refuses before it reaches here (imagesEnabled is false in exactly that
// case), so this error is the backstop for a future caller, not the refusal a client sees.
type noopImages struct{}

func (noopImages) Put(context.Context, string, string, io.Reader, int64, string) (string, error) {
	return "", errors.New("image storage not configured")
}

/* ------------------------------------------------------------------ caller identity */

// caller is the authenticated identity, in both the form the responses use (strings) and the
// form the queries use (pgtype.UUID). Resolving it once per RPC keeps the two from drifting
// and keeps family_id out of every request message.
type caller struct {
	familyID string
	userID   string
	fam      pgtype.UUID
	user     pgtype.UUID
}

func (h *Handler) caller(ctx context.Context) (caller, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return caller{}, err
	}
	if claims.UserID == "" {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no subject"))
	}
	if claims.FamilyID == "" {
		return caller{}, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("caller belongs to no family"))
	}
	fam, err := pgconv.UUID(claims.FamilyID)
	if err != nil || !fam.Valid {
		return caller{}, connect.NewError(connect.CodeInvalidArgument,
			errors.New("token carries a malformed family id"))
	}
	user, err := pgconv.UUID(claims.UserID)
	if err != nil || !user.Valid {
		return caller{}, connect.NewError(connect.CodeInvalidArgument,
			errors.New("token carries a malformed subject"))
	}
	return caller{familyID: claims.FamilyID, userID: claims.UserID, fam: fam, user: user}, nil
}

// requireUUID parses an id the request cannot do without.
func requireUUID(s, field string) (pgtype.UUID, error) {
	id, err := pgconv.UUID(s)
	if err != nil || !id.Valid {
		return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s is required", field))
	}
	return id, nil
}

// optionalUUID parses an id whose empty value is meaningful — an unfiled note, a top-level
// notebook — into the invalid (SQL NULL) pgtype.UUID the queries treat as "none".
func optionalUUID(s, field string) (pgtype.UUID, error) {
	if trimmed(s) == "" {
		return pgtype.UUID{}, nil
	}
	id, err := pgconv.UUID(s)
	if err != nil || !id.Valid {
		return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s is malformed", field))
	}
	return id, nil
}

// checkTextLength bounds one free-text field in RUNES. AGENTS.md: text arriving from a client
// is bounded in runes, not bytes — a Ukrainian recipe must not be worth half an English one,
// and a message that says "characters" has to be counting them.
func checkTextLength(value, field string, limit int) error {
	if utf8.RuneCountInString(value) > limit {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s holds at most %d characters", field, limit))
	}
	return nil
}

var (
	errNoteNotFound     = errors.New("note not found")
	errNotebookNotFound = errors.New("notebook not found")
)

func notFoundNote() error {
	return connect.NewError(connect.CodeNotFound, errNoteNotFound)
}

// viewOnlyNote is the one refusal a caller who can already see a note gets for a write. It is
// PermissionDenied and not NotFound because they know the note is there; the message is the
// same wherever the refusal comes from — the pre-flight check, the SQL predicate, or the
// re-read after a statement wrote nothing.
func viewOnlyNote() error {
	return connect.NewError(connect.CodePermissionDenied,
		errors.New("you have view-only access to this note"))
}

func notFoundNotebook() error {
	return connect.NewError(connect.CodeNotFound, errNotebookNotFound)
}

/* ------------------------------------------------------------------ notebooks */

func (h *Handler) CreateNotebook(
	ctx context.Context, req *connect.Request[notesv1.CreateNotebookRequest],
) (*connect.Response[notesv1.CreateNotebookResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if err := checkTextLength(name, "a notebook name", maxNotebookName); err != nil {
		return nil, err
	}
	parentID, err := optionalUUID(req.Msg.GetParentId(), "parent_id")
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		// Nesting a notebook under a parent is a write to that parent, the same act
		// CreateNote and MoveNote gate with mustEditNotebook one level down: a VIEW share
		// would otherwise be a way to hang a folder inside somebody else's folder.
		if _, err := h.mustEditNotebook(ctx, c, parentID); err != nil {
			return nil, err
		}
	}

	nb, err := h.q.CreateNotebook(ctx, db.CreateNotebookParams{
		FamilyID:    c.fam,
		OwnerUserID: c.user,
		ParentID:    parentID,
		Name:        name,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create notebook")
	}
	// A notebook nobody has put a note in yet holds zero notes; no count query needed.
	return connect.NewResponse(&notesv1.CreateNotebookResponse{
		Notebook: toProtoNotebook(viewFromNotebook(nb, 0)),
	}), nil
}

func (h *Handler) ListNotebooks(
	ctx context.Context, req *connect.Request[notesv1.ListNotebooksRequest],
) (*connect.Response[notesv1.ListNotebooksResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListNotebooks(ctx, db.ListNotebooksParams{
		UserID:          c.user,
		FamilyID:        c.fam,
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list notebooks")
	}
	out := make([]*notesv1.Notebook, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoNotebook(viewFromNotebookRow(r)))
	}
	return connect.NewResponse(&notesv1.ListNotebooksResponse{Notebooks: out}), nil
}

func (h *Handler) UpdateNotebook(
	ctx context.Context, req *connect.Request[notesv1.UpdateNotebookRequest],
) (*connect.Response[notesv1.UpdateNotebookResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notebookID, err := requireUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if err := checkTextLength(name, "a notebook name", maxNotebookName); err != nil {
		return nil, err
	}
	parentID, err := optionalUUID(req.Msg.GetParentId(), "parent_id")
	if err != nil {
		return nil, err
	}
	if parentID.Valid && parentID == notebookID {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("a notebook cannot be its own parent"))
	}
	if err := h.mustOwnNotebook(ctx, c, notebookID); err != nil {
		return nil, err
	}
	if parentID.Valid {
		// Same gate as CreateNotebook, and as CreateNote/MoveNote one level down: putting a
		// thing into a notebook is writing that notebook.
		if _, err := h.mustEditNotebook(ctx, c, parentID); err != nil {
			return nil, err
		}
		if err := h.refuseNotebookCycle(ctx, c, notebookID, parentID); err != nil {
			return nil, err
		}
	}

	if _, err := h.q.UpdateNotebook(ctx, db.UpdateNotebookParams{
		Name:       name,
		ParentID:   parentID,
		Archived:   req.Msg.GetArchived(),
		NotebookID: notebookID,
		FamilyID:   c.fam,
		UserID:     c.user,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNotebook()
		}
		return nil, h.internal(ctx, err, "update notebook")
	}

	// Re-read rather than render the UPDATE's row: note_count is the caller's count and only
	// the read query computes it, and an archived notebook's count is what the sidebar draws.
	row, err := h.notebookForCaller(ctx, c, notebookID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&notesv1.UpdateNotebookResponse{
		Notebook: toProtoNotebook(viewFromGetNotebook(row)),
	}), nil
}

func (h *Handler) DeleteNotebook(
	ctx context.Context, req *connect.Request[notesv1.DeleteNotebookRequest],
) (*connect.Response[notesv1.DeleteNotebookResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notebookID, err := requireUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	if err := h.mustOwnNotebook(ctx, c, notebookID); err != nil {
		return nil, err
	}

	// The refusal is checked here so the caller gets a sentence it can show. The DELETE
	// carries the same condition, so a note created between the two does not slip through.
	//
	// Two numbers come back because there are two sentences to say. The total is what blocks
	// the delete — a notebook shared into must not be deletable out from under someone else's
	// note — but it may count notes this caller cannot see, and printing that number is the
	// existence leak Notebook.note_count exists to avoid. So only the visible count is ever
	// spoken; when it is zero and the total is not, the caller is told the shape of the
	// problem and who can fix it, without a number.
	counts, err := h.q.CountNotebookNotes(ctx, db.CountNotebookNotesParams{
		UserID: c.user, NotebookID: notebookID, FamilyID: c.fam,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "count notebook notes")
	}
	switch {
	case counts.Visible > 0:
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("notebook still holds %d note(s); move or delete them first", counts.Visible))
	case counts.Total > 0:
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(
			"notebook holds notes belonging to the people it is shared with; "+
				"they have to move or delete them first"))
	}

	rows, err := h.q.DeleteNotebook(ctx, db.DeleteNotebookParams{
		NotebookID: notebookID, FamilyID: c.fam, UserID: c.user,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete notebook")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("notebook is not empty"))
	}
	return connect.NewResponse(&notesv1.DeleteNotebookResponse{}), nil
}

// notebookForCaller reads a notebook through the visibility predicate. Not visible is
// NotFound, never PermissionDenied.
func (h *Handler) notebookForCaller(
	ctx context.Context, c caller, notebookID pgtype.UUID,
) (db.GetNotebookRow, error) {
	row, err := h.q.GetNotebook(ctx, db.GetNotebookParams{
		UserID: c.user, NotebookID: notebookID, FamilyID: c.fam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetNotebookRow{}, notFoundNotebook()
		}
		return db.GetNotebookRow{}, h.internal(ctx, err, "get notebook")
	}
	return row, nil
}

// mustEditNotebook is the gate for putting something INTO a notebook — creating a note there,
// moving one there, or nesting another notebook under it. Visibility is not enough: a VIEW
// share would otherwise be a write endpoint
// into somebody else's folder, and the note that lands there is invisible to the folder's
// owner (the note predicate has no "the notebook's owner sees what is in it" clause) while
// still blocking their DeleteNotebook. It is the same rule UploadNoteImage states: putting
// something into a thing is writing it.
func (h *Handler) mustEditNotebook(
	ctx context.Context, c caller, notebookID pgtype.UUID,
) (db.GetNotebookRow, error) {
	row, err := h.notebookForCaller(ctx, c, notebookID)
	if err != nil {
		return db.GetNotebookRow{}, err
	}
	if !row.CanEdit {
		return db.GetNotebookRow{}, connect.NewError(connect.CodePermissionDenied,
			errors.New("you have view-only access to this notebook"))
	}
	return row, nil
}

// refuseNotebookCycle rejects a re-parent that would close a loop in the notebook tree.
//
// Rejecting only a direct self-parent is not enough: two moves, each legal on its own, build
// A.parent = B and B.parent = A. Nothing in this service walks the tree, so the server would
// never notice — but the app builds its sidebar from parent_id, and a cycle there is a render
// that does not terminate. The check belongs here because this is where the edge is created.
//
// The walk starts at the proposed parent and goes up; reaching the notebook being edited means
// the new edge closes the loop. It runs over ids only and depth-capped in SQL, so it says
// nothing about notebooks the caller cannot see and cannot hang on a cycle already stored.
func (h *Handler) refuseNotebookCycle(
	ctx context.Context, c caller, notebookID, parentID pgtype.UUID,
) error {
	ancestors, err := h.q.NotebookAncestors(ctx, db.NotebookAncestorsParams{
		NotebookID: parentID, FamilyID: c.fam,
	})
	if err != nil {
		return h.internal(ctx, err, "walk notebook ancestors")
	}
	for _, id := range ancestors {
		if id == notebookID {
			return connect.NewError(connect.CodeInvalidArgument,
				errors.New("that would put the notebook inside itself"))
		}
	}
	return nil
}

// mustOwnNotebook is the authority check for renaming, archiving, deleting and sharing a
// notebook: an EDIT share grants writes to the notes inside, not authority over the folder.
func (h *Handler) mustOwnNotebook(ctx context.Context, c caller, notebookID pgtype.UUID) error {
	row, err := h.notebookForCaller(ctx, c, notebookID)
	if err != nil {
		return err
	}
	if pgconv.UUIDString(row.OwnerUserID) != c.userID {
		return connect.NewError(connect.CodePermissionDenied,
			errors.New("only the notebook's owner can do that"))
	}
	return nil
}

/* ------------------------------------------------------------------ notes */

func (h *Handler) CreateNote(
	ctx context.Context, req *connect.Request[notesv1.CreateNoteRequest],
) (*connect.Response[notesv1.CreateNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notebookID, err := optionalUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	if notebookID.Valid {
		// Filing a note into a notebook is a write to that notebook, so it needs EDIT there
		// and not merely the right to look at it.
		if _, err := h.mustEditNotebook(ctx, c, notebookID); err != nil {
			return nil, err
		}
	}
	blocks := req.Msg.GetBlocks()
	if err := checkBlocks(blocks); err != nil {
		return nil, err
	}
	encoded, err := encodeBlocks(blocks)
	if err != nil {
		return nil, h.internal(ctx, err, "encode blocks")
	}
	title := trimmed(req.Msg.GetTitle())
	if err := checkTextLength(title, "a note title", maxTitle); err != nil {
		return nil, err
	}
	body := bodyText(blocks)
	clientID := trimmed(req.Msg.GetClientId())

	var (
		noteID  pgtype.UUID
		created bool
	)
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if clientID != "" {
			row, err := q.CreateNoteByClientID(ctx, db.CreateNoteByClientIDParams{
				FamilyID: c.fam, OwnerUserID: c.user, NotebookID: notebookID,
				ClientID: clientID, Title: title, Blocks: encoded, BodyText: body,
			})
			if err != nil {
				return h.internal(ctx, err, "create note")
			}
			noteID = row.ID
			// The upsert answers with the FIRST call's row, so the row alone cannot say
			// whether this call made it. The activity log can: a note this service created
			// always has its CREATED row, so an empty log means this INSERT is the one that
			// inserted, and a retry writes no second activity row and re-fires no event.
			prior, err := q.ListActivity(ctx, db.ListActivityParams{
				NoteID: row.ID, FamilyID: c.fam, UserID: c.user, LimitCount: 1,
			})
			if err != nil {
				return h.internal(ctx, err, "read note activity")
			}
			created = len(prior) == 0
		} else {
			row, err := q.CreateNote(ctx, db.CreateNoteParams{
				FamilyID: c.fam, OwnerUserID: c.user, NotebookID: notebookID,
				ClientID: "", Title: title, Blocks: encoded, BodyText: body,
			})
			if err != nil {
				return h.internal(ctx, err, "create note")
			}
			noteID, created = row.ID, true
		}
		if !created {
			return nil
		}
		if _, err := q.AddActivity(ctx, db.AddActivityParams{
			NoteID:      noteID,
			ActorUserID: c.user,
			Kind:        int16(notesv1.ActivityKind_ACTIVITY_KIND_CREATED),
			Detail:      title,
		}); err != nil {
			return h.internal(ctx, err, "record activity")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	if created {
		h.publish(ctx, fmevents.SubjectNotesNoteCreated,
			notesevents.NoteEvent(c.familyID, note.GetId(), c.userID, h.now()))
	}
	return connect.NewResponse(&notesv1.CreateNoteResponse{Note: note}), nil
}

func (h *Handler) GetNote(
	ctx context.Context, req *connect.Request[notesv1.GetNoteRequest],
) (*connect.Response[notesv1.GetNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&notesv1.GetNoteResponse{Note: note}), nil
}

func (h *Handler) ListNotes(
	ctx context.Context, req *connect.Request[notesv1.ListNotesRequest],
) (*connect.Response[notesv1.ListNotesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notebookID, err := optionalUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListNotes(ctx, db.ListNotesParams{
		UserID:          c.user,
		FamilyID:        c.fam,
		NotebookID:      notebookID,
		StarredOnly:     req.Msg.GetStarredOnly(),
		IncludeArchived: req.Msg.GetIncludeArchived(),
		ArchivedOnly:    req.Msg.GetArchivedOnly(),
		SharedOnly:      req.Msg.GetSharedOnly(),
		Sort:            sortKey(req.Msg.GetSort()),
		PageSize:        pageSize(req.Msg.GetPageSize()),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list notes")
	}
	out := make([]*notesv1.Note, 0, len(rows))
	for _, r := range rows {
		// Shares stay empty on a list row: one share query per note to draw a glyph the
		// `shared` flag already answers is a hundred round trips for a hundred notes.
		out = append(out, toProtoNote(viewFromList(r), nil))
	}
	return connect.NewResponse(&notesv1.ListNotesResponse{Notes: out}), nil
}

func (h *Handler) UpdateNote(
	ctx context.Context, req *connect.Request[notesv1.UpdateNoteRequest],
) (*connect.Response[notesv1.UpdateNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	blocks := req.Msg.GetBlocks()
	if err := checkBlocks(blocks); err != nil {
		return nil, err
	}

	// The whole note is read, not just the permission: the activity kind is a diff of the
	// stored blocks against the incoming ones, and the version check wants a message naming
	// both numbers. One read gives all three.
	current, err := h.q.GetNote(ctx, db.GetNoteParams{
		UserID: c.user, NoteID: noteID, FamilyID: c.fam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNote()
		}
		return nil, h.internal(ctx, err, "get note")
	}
	if !current.CanEdit {
		return nil, viewOnlyNote()
	}
	expected := req.Msg.GetExpectedVersion()
	if expected != 0 && expected != current.Version {
		return nil, versionConflict(expected, current.Version)
	}

	title := trimmed(req.Msg.GetTitle())
	if err := checkTextLength(title, "a note title", maxTitle); err != nil {
		return nil, err
	}
	encoded, err := encodeBlocks(blocks)
	if err != nil {
		return nil, h.internal(ctx, err, "encode blocks")
	}
	kind, detail := editKind(current.Title, decodeBlocks(current.Blocks), title, blocks)

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		updated, err := q.UpdateNote(ctx, db.UpdateNoteParams{
			Title:           title,
			Blocks:          encoded,
			BodyText:        bodyText(blocks),
			UserID:          c.user,
			NoteID:          noteID,
			FamilyID:        c.fam,
			ExpectedVersion: expected,
		})
		if err != nil {
			// No row means one of the statement's two guards rejected the write, and which
			// one decides the rpc error: the note moved out of reach, the caller's EDIT went
			// away, or somebody else committed between the read above and this statement.
			if errors.Is(err, pgx.ErrNoRows) {
				return h.updateRefusal(ctx, q, c, noteID, expected)
			}
			return h.internal(ctx, err, "update note")
		}
		if _, err := q.AddActivity(ctx, db.AddActivityParams{
			NoteID:      updated.ID,
			ActorUserID: c.user,
			Kind:        kind,
			Detail:      detail,
		}); err != nil {
			return h.internal(ctx, err, "record activity")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	h.publish(ctx, fmevents.SubjectNotesNoteUpdated,
		notesevents.NoteEvent(c.familyID, note.GetId(), c.userID, h.now()))
	return connect.NewResponse(&notesv1.UpdateNoteResponse{Note: note}), nil
}

// updateRefusal names the guard that rejected an UpdateNote. The statement carries both the
// write predicate and expected_version, and from here either refusal looks the same — no row —
// while the two are different rpc errors the app reacts to differently: ABORTED is a retry
// with fresh content, PermissionDenied is not. So the permission is re-read rather than
// guessed, on this path only: a successful write never pays for it, and the answer is the same
// one the pre-flight check gives — NotFound for a note that is gone or was never visible,
// PermissionDenied for one the caller may see but not write, ABORTED for the lost race.
//
// The re-read runs on the transaction's own Querier, so what it reports is the state the
// refused statement saw.
func (h *Handler) updateRefusal(
	ctx context.Context, q db.Querier, c caller, noteID pgtype.UUID, expected int64,
) error {
	row, err := q.GetNoteEditPermission(ctx, db.GetNoteEditPermissionParams{
		UserID: c.user, NoteID: noteID, FamilyID: c.fam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundNote()
		}
		return h.internal(ctx, err, "get note permission")
	}
	if !row.CanEdit {
		return viewOnlyNote()
	}
	return versionConflict(expected, row.Version)
}

// versionConflict is the one message the app shows on a lost race, so it names both versions
// rather than saying "conflict".
func versionConflict(expected, stored int64) error {
	return connect.NewError(connect.CodeAborted, fmt.Errorf(
		"this note changed since you opened it (you have version %d, the saved note is version %d)",
		expected, stored))
}

// editKind classifies an update for the activity rail. Ticking one checkbox is the single
// most common write in the app and reads as its own line ("Maya checked Buy milk"); anything
// else — a changed title, changed text, a block added, moved or removed, two boxes at once —
// is one EDITED line, because the rail is a history, not a diff viewer.
func editKind(
	oldTitle string, oldBlocks []*notesv1.Block, newTitle string, newBlocks []*notesv1.Block,
) (int16, string) {
	edited := int16(notesv1.ActivityKind_ACTIVITY_KIND_EDITED)
	if oldTitle != newTitle || len(oldBlocks) != len(newBlocks) {
		return edited, ""
	}
	toggled := -1
	for i := range newBlocks {
		o, n := oldBlocks[i], newBlocks[i]
		if o.GetId() != n.GetId() || o.GetType() != n.GetType() || o.GetText() != n.GetText() ||
			o.GetLevel() != n.GetLevel() || o.GetImageUrl() != n.GetImageUrl() ||
			o.GetLanguage() != n.GetLanguage() {
			return edited, ""
		}
		if o.GetChecked() != n.GetChecked() {
			if toggled >= 0 {
				return edited, ""
			}
			toggled = i
		}
	}
	if toggled < 0 || newBlocks[toggled].GetType() != notesv1.BlockType_BLOCK_TYPE_TODO {
		return edited, ""
	}
	block := newBlocks[toggled]
	if block.GetChecked() {
		return int16(notesv1.ActivityKind_ACTIVITY_KIND_TASK_CHECKED), preview(block.GetText())
	}
	return int16(notesv1.ActivityKind_ACTIVITY_KIND_TASK_UNCHECKED), preview(block.GetText())
}

func (h *Handler) MoveNote(
	ctx context.Context, req *connect.Request[notesv1.MoveNoteRequest],
) (*connect.Response[notesv1.MoveNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	notebookID, err := optionalUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	// Moving is the owner's alone, like deleting. Which notebook a note sits in decides who
	// else can read it — the visibility predicate reaches a note through its notebook — so an
	// EDIT share would otherwise be a way to re-share someone else's note to a whole
	// notebook's audience, invisibly: GetNote's share sheet lists note_shares only, and the
	// note's owner cannot revoke a share on a notebook they do not own.
	perm, err := h.notePermission(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	if pgconv.UUIDString(perm.OwnerUserID) != c.userID {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("only the note's owner can move it"))
	}
	if notebookID.Valid {
		// Moving a note into a notebook the caller cannot see would hide it from them, and
		// filing into someone else's folder is a write to that folder: the destination needs
		// EDIT, the same gate CreateNote uses.
		if _, err := h.mustEditNotebook(ctx, c, notebookID); err != nil {
			return nil, err
		}
	}

	if _, err := h.q.MoveNote(ctx, db.MoveNoteParams{
		NotebookID: notebookID, UserID: c.user, NoteID: noteID, FamilyID: c.fam,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNote()
		}
		return nil, h.internal(ctx, err, "move note")
	}

	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	h.publish(ctx, fmevents.SubjectNotesNoteUpdated,
		notesevents.NoteEvent(c.familyID, note.GetId(), c.userID, h.now()))
	return connect.NewResponse(&notesv1.MoveNoteResponse{Note: note}), nil
}

func (h *Handler) ToggleStar(
	ctx context.Context, req *connect.Request[notesv1.ToggleStarRequest],
) (*connect.Response[notesv1.ToggleStarResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	// starred is one column on the shared row, not a per-viewer flag, so starring a note
	// changes it for everyone who can see it — which makes it a write, and writes need EDIT.
	if _, err := h.mustEditNote(ctx, c, noteID); err != nil {
		return nil, err
	}

	// UserID is passed because the statement carries the same write predicate this gate does
	// (see the query's comment); a viewer reaching it writes no row.
	if _, err := h.q.SetNoteStarred(ctx, db.SetNoteStarredParams{
		Starred: req.Msg.GetStarred(), NoteID: noteID, FamilyID: c.fam, UserID: c.user,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNote()
		}
		return nil, h.internal(ctx, err, "star note")
	}

	// No event and no activity row: starring is not an edit (the query does not bump version
	// or last_edited_by_user_id either), and a rail full of stars buries the real history.
	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&notesv1.ToggleStarResponse{Note: note}), nil
}

func (h *Handler) ArchiveNote(
	ctx context.Context, req *connect.Request[notesv1.ArchiveNoteRequest],
) (*connect.Response[notesv1.ArchiveNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	// archived is one column on the shared row, like starred: archiving takes the note out of
	// everyone's list, not just the caller's, which makes it a write, and writes need EDIT.
	// Not owner-only — unlike deleting and moving, archiving loses nothing and re-shares
	// nothing: the blocks, the comments, the shares and the notebook all stay where they are,
	// and the same EDIT share that may rewrite the whole document may also file it away.
	if _, err := h.mustEditNote(ctx, c, noteID); err != nil {
		return nil, err
	}

	if _, err := h.q.SetNoteArchived(ctx, db.SetNoteArchivedParams{
		Archived: req.Msg.GetArchived(), NoteID: noteID, FamilyID: c.fam, UserID: c.user,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNote()
		}
		return nil, h.internal(ctx, err, "archive note")
	}

	// No activity row: ActivityKind has no ARCHIVED member, and recording it as EDITED would
	// put a line in the rail that says something untrue about the document. An event does go
	// out, unlike starring — archiving moves the note between lists for everybody who can see
	// it, which is the same kind of change MoveNote publishes.
	note, err := h.noteResponse(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	h.publish(ctx, fmevents.SubjectNotesNoteUpdated,
		notesevents.NoteEvent(c.familyID, note.GetId(), c.userID, h.now()))
	return connect.NewResponse(&notesv1.ArchiveNoteResponse{Note: note}), nil
}

func (h *Handler) DeleteNote(
	ctx context.Context, req *connect.Request[notesv1.DeleteNoteRequest],
) (*connect.Response[notesv1.DeleteNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	perm, err := h.notePermission(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	// Deletion is the owner's alone: an EDIT share grants writing the document, not
	// destroying it.
	if pgconv.UUIDString(perm.OwnerUserID) != c.userID {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("only the note's owner can delete it"))
	}

	rows, err := h.q.DeleteNote(ctx, db.DeleteNoteParams{
		NoteID: noteID, FamilyID: c.fam, UserID: c.user,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete note")
	}
	if rows == 0 {
		return nil, notFoundNote()
	}
	return connect.NewResponse(&notesv1.DeleteNoteResponse{}), nil
}

// maxImageBytes caps an upload well below MinIO's limits — an image block is a photo or a
// screenshot the client should already be compressing, not a raw sensor dump.
const maxImageBytes = 8 << 20 // 8 MiB

var imageExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (h *Handler) UploadNoteImage(
	ctx context.Context, req *connect.Request[notesv1.UploadNoteImageRequest],
) (*connect.Response[notesv1.UploadNoteImageResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	// v1 ships no image blocks and is deployed with no object store, so this refusal is
	// permanent and expected for the whole deployment rather than a failure worth a reference
	// id and an internal-error log line (see the rpc's comment in notes.proto).
	//
	// Unimplemented rather than FailedPrecondition: nothing the caller controls — no argument,
	// no retry, no state they can put the system into — makes this procedure work here, and
	// FailedPrecondition would promise a precondition they could go and satisfy. It answers
	// before the arguments are parsed and before the note is looked up, because a procedure
	// this deployment does not implement must not spend a round trip confirming that somebody
	// else's note exists.
	if !h.imagesEnabled {
		return nil, connect.NewError(connect.CodeUnimplemented,
			errors.New("image blocks are not available in this version"))
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	contentType := trimmed(req.Msg.GetContentType())
	ext, ok := imageExtensions[contentType]
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unsupported content_type %q", contentType))
	}
	data := req.Msg.GetImage()
	if len(data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image is required"))
	}
	if len(data) > maxImageBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image exceeds 8MB limit"))
	}
	// Putting bytes into a note is writing it, so it needs EDIT — otherwise a VIEW share
	// would be a free upload endpoint into the family's bucket.
	if _, err := h.mustEditNote(ctx, c, noteID); err != nil {
		return nil, err
	}

	// Content-addressed within the note: one image can appear in several blocks, an editor
	// that re-sends the same bytes overwrites one object instead of leaking a new one, and
	// the family prefix keeps one family's keys out of another's listing.
	sum := sha256.Sum256(data)
	key := fmt.Sprintf("%s/%s/%s%s", c.familyID, pgconv.UUIDString(noteID),
		hex.EncodeToString(sum[:16]), ext)

	url, err := h.images.Put(ctx, h.imageBucket, key, bytes.NewReader(data),
		int64(len(data)), contentType)
	if err != nil {
		return nil, h.internal(ctx, err, "upload note image")
	}
	// The URL is not written to the note here: it belongs in an IMAGE block, and the block
	// array is only ever written whole by UpdateNote. The client puts it there.
	return connect.NewResponse(&notesv1.UploadNoteImageResponse{ImageUrl: url}), nil
}

// checkBlocks bounds one note before it reaches the database, so a runaway client gets an
// argument error instead of a row every list query then has to carry.
func checkBlocks(blocks []*notesv1.Block) error {
	if len(blocks) > maxBlocks {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("a note holds at most %d blocks", maxBlocks))
	}
	for _, b := range blocks {
		// Runes, not bytes: the message says characters, and a Ukrainian note must not be
		// worth half an English one.
		if utf8.RuneCountInString(b.GetText()) > maxBlockText {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("a block holds at most %d characters", maxBlockText))
		}
	}
	return nil
}

// notePermission answers "can this caller see, and may they write, this note?" — invisible is
// NotFound, which is the rule the whole service turns on.
func (h *Handler) notePermission(
	ctx context.Context, c caller, noteID pgtype.UUID,
) (db.GetNoteEditPermissionRow, error) {
	row, err := h.q.GetNoteEditPermission(ctx, db.GetNoteEditPermissionParams{
		UserID: c.user, NoteID: noteID, FamilyID: c.fam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetNoteEditPermissionRow{}, notFoundNote()
		}
		return db.GetNoteEditPermissionRow{}, h.internal(ctx, err, "get note permission")
	}
	return row, nil
}

// mustEditNote is notePermission plus the EDIT gate. PermissionDenied here is safe: the
// caller can already see the note, so refusing the write tells them nothing new.
func (h *Handler) mustEditNote(
	ctx context.Context, c caller, noteID pgtype.UUID,
) (db.GetNoteEditPermissionRow, error) {
	row, err := h.notePermission(ctx, c, noteID)
	if err != nil {
		return db.GetNoteEditPermissionRow{}, err
	}
	if !row.CanEdit {
		return db.GetNoteEditPermissionRow{}, viewOnlyNote()
	}
	return row, nil
}

// noteResponse renders the full note — the shape GetNote returns, shares included. Every
// write path ends here rather than rendering the row its own statement returned: the derived
// columns (preview, task counts, shared, can_edit) are computed by the read query, and
// computing them a second time in Go is where the two answers start to disagree.
func (h *Handler) noteResponse(
	ctx context.Context, c caller, noteID pgtype.UUID,
) (*notesv1.Note, error) {
	row, err := h.q.GetNote(ctx, db.GetNoteParams{
		UserID: c.user, NoteID: noteID, FamilyID: c.fam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFoundNote()
		}
		return nil, h.internal(ctx, err, "get note")
	}
	shares, err := h.q.ListNoteShares(ctx, db.ListNoteSharesParams{
		NoteID: noteID, FamilyID: c.fam, UserID: c.user,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list note shares")
	}
	out := make([]*notesv1.Share, 0, len(shares))
	for _, s := range shares {
		out = append(out, toProtoNoteShare(s))
	}
	return toProtoNote(viewFromGet(row), out), nil
}

/* ------------------------------------------------------------------ search */

func (h *Handler) Search(
	ctx context.Context, req *connect.Request[notesv1.SearchRequest],
) (*connect.Response[notesv1.SearchResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	query := trimmed(req.Msg.GetQuery())
	limit := searchLimit(req.Msg.GetLimit())

	// The corpus size is counted outside the timed section: it is the footer's "128 notes",
	// not part of what the palette is waiting for.
	searched, err := h.q.CountVisibleNotes(ctx, db.CountVisibleNotesParams{
		FamilyID: c.fam, UserID: c.user,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "count visible notes")
	}
	if query == "" {
		// An empty box is not an error and not a request for everything: the palette opens
		// with it, and the footer still wants its corpus size.
		return connect.NewResponse(&notesv1.SearchResponse{SearchedNotes: searched}), nil
	}

	facet := req.Msg.GetFacet()
	wantNotes := facet == notesv1.SearchFacet_SEARCH_FACET_NOTES || isAllFacet(facet)
	wantTasks := facet == notesv1.SearchFacet_SEARCH_FACET_TASKS || isAllFacet(facet)
	wantNotebooks := facet == notesv1.SearchFacet_SEARCH_FACET_NOTEBOOKS || isAllFacet(facet)

	hits := make([]*notesv1.SearchHit, 0, limit)
	start := h.now()
	if wantNotes {
		rows, err := h.q.SearchNotes(ctx, db.SearchNotesParams{
			Query: query, FamilyID: c.fam, UserID: c.user, LimitCount: limit,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "search notes")
		}
		for _, r := range rows {
			hits = append(hits, &notesv1.SearchHit{
				Kind:       notesv1.SearchFacet_SEARCH_FACET_NOTES,
				NoteId:     pgconv.UUIDString(r.ID),
				NotebookId: pgconv.UUIDString(r.NotebookID),
				Title:      r.Title,
				Context:    breadcrumb(r.NotebookName),
				// ts_headline comes back as text; sqlc types the column []byte because the
				// function's return type is not one it maps.
				Snippet:   string(r.Snippet),
				UpdatedAt: pgconv.Timestamp(r.UpdatedAt),
			})
		}
	}
	if wantTasks {
		rows, err := h.q.SearchTasks(ctx, db.SearchTasksParams{
			Query: query, FamilyID: c.fam, UserID: c.user, LimitCount: limit,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "search tasks")
		}
		for _, r := range rows {
			hits = append(hits, &notesv1.SearchHit{
				Kind:       notesv1.SearchFacet_SEARCH_FACET_TASKS,
				NoteId:     pgconv.UUIDString(r.NoteID),
				NotebookId: pgconv.UUIDString(r.NotebookID),
				Title:      r.Title,
				Context:    breadcrumb(r.NotebookName),
				Snippet:    r.Snippet,
				UpdatedAt:  pgconv.Timestamp(r.UpdatedAt),
			})
		}
	}
	if wantNotebooks {
		rows, err := h.q.SearchNotebooks(ctx, db.SearchNotebooksParams{
			FamilyID: c.fam, Query: query, UserID: c.user, LimitCount: limit,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "search notebooks")
		}
		for _, r := range rows {
			hits = append(hits, &notesv1.SearchHit{
				Kind:       notesv1.SearchFacet_SEARCH_FACET_NOTEBOOKS,
				NotebookId: pgconv.UUIDString(r.ID),
				Title:      r.Name,
				Context:    breadcrumb(r.ParentName),
				UpdatedAt:  pgconv.Timestamp(r.UpdatedAt),
			})
		}
	}
	// Measured around the queries themselves, so the number the footer prints is the
	// server's work and not the round trip the user's network dominates.
	elapsed := h.now().Sub(start)

	if int32(len(hits)) > limit {
		// Each facet ran with its own limit; ALL takes the first `limit` of the three lists
		// concatenated, which keeps the palette's page one page.
		hits = hits[:limit]
	}
	return connect.NewResponse(&notesv1.SearchResponse{
		Hits:          hits,
		ElapsedMs:     elapsedMillis(elapsed),
		SearchedNotes: searched,
	}), nil
}

// isAllFacet treats UNSPECIFIED as ALL: a client that sends no facet is the palette's default
// tab, which searches everything.
func isAllFacet(f notesv1.SearchFacet) bool {
	return f == notesv1.SearchFacet_SEARCH_FACET_ALL ||
		f == notesv1.SearchFacet_SEARCH_FACET_UNSPECIFIED
}

// elapsedMillis never reports a negative or absurd number, whatever the clock did.
func elapsedMillis(d time.Duration) int32 {
	ms := d.Milliseconds()
	if ms < 0 {
		return 0
	}
	if ms > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(ms)
}

/* ------------------------------------------------------------------ sharing */

func (h *Handler) ShareNote(
	ctx context.Context, req *connect.Request[notesv1.ShareNoteRequest],
) (*connect.Response[notesv1.ShareNoteResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	memberID, err := shareSubjectMember(req.Msg.GetSubject(), req.Msg.GetMemberUserId(), c)
	if err != nil {
		return nil, err
	}
	perm, err := h.notePermission(ctx, c, noteID)
	if err != nil {
		return nil, err
	}
	if pgconv.UUIDString(perm.OwnerUserID) != c.userID {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("only the note's owner can share it"))
	}
	permission := permissionFromProto(req.Msg.GetPermission())

	var share db.NoteShare
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		if memberID.Valid {
			share, err = q.ShareNoteWithMember(ctx, db.ShareNoteWithMemberParams{
				MemberUserID: memberID, Permission: permission, GrantedByUserID: c.user,
				NoteID: noteID, FamilyID: c.fam,
			})
		} else {
			share, err = q.ShareNoteWithFamily(ctx, db.ShareNoteWithFamilyParams{
				Permission: permission, GrantedByUserID: c.user, NoteID: noteID, FamilyID: c.fam,
			})
		}
		if err != nil {
			// The statement is an INSERT ... SELECT over the owned row, so no row means the
			// ownership check in SQL disagreed with the one above — a note deleted in between.
			if errors.Is(err, pgx.ErrNoRows) {
				return notFoundNote()
			}
			return h.internal(ctx, err, "share note")
		}
		// The rail is a log of actions, not a picture of the current grants — ListShares is
		// that — so a re-share that only changes the permission is a second row, on purpose.
		if _, err := q.AddActivity(ctx, db.AddActivityParams{
			NoteID:      noteID,
			ActorUserID: c.user,
			Kind:        int16(notesv1.ActivityKind_ACTIVITY_KIND_SHARED),
			Detail:      shareDetail(memberID),
		}); err != nil {
			return h.internal(ctx, err, "record activity")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := toProtoNoteShare(share)
	h.publishShare(ctx, c, "note", pgconv.UUIDString(noteID), out)
	return connect.NewResponse(&notesv1.ShareNoteResponse{Share: out}), nil
}

func (h *Handler) ShareNotebook(
	ctx context.Context, req *connect.Request[notesv1.ShareNotebookRequest],
) (*connect.Response[notesv1.ShareNotebookResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notebookID, err := requireUUID(req.Msg.GetNotebookId(), "notebook_id")
	if err != nil {
		return nil, err
	}
	memberID, err := shareSubjectMember(req.Msg.GetSubject(), req.Msg.GetMemberUserId(), c)
	if err != nil {
		return nil, err
	}
	if err := h.mustOwnNotebook(ctx, c, notebookID); err != nil {
		return nil, err
	}
	permission := permissionFromProto(req.Msg.GetPermission())

	var share db.NotebookShare
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		if memberID.Valid {
			share, err = q.ShareNotebookWithMember(ctx, db.ShareNotebookWithMemberParams{
				MemberUserID: memberID, Permission: permission, GrantedByUserID: c.user,
				NotebookID: notebookID, FamilyID: c.fam,
			})
		} else {
			share, err = q.ShareNotebookWithFamily(ctx, db.ShareNotebookWithFamilyParams{
				Permission: permission, GrantedByUserID: c.user,
				NotebookID: notebookID, FamilyID: c.fam,
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return notFoundNotebook()
			}
			return h.internal(ctx, err, "share notebook")
		}
		// A notebook has no rail of its own, so the share is recorded on the notes it just
		// made visible — one row each, written by one statement.
		if err := q.AddNotebookShareActivity(ctx, db.AddNotebookShareActivityParams{
			ActorUserID: c.user,
			Kind:        int16(notesv1.ActivityKind_ACTIVITY_KIND_SHARED),
			Detail:      shareDetail(memberID),
			NotebookID:  notebookID,
			FamilyID:    c.fam,
		}); err != nil {
			return h.internal(ctx, err, "record activity")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := toProtoNotebookShare(share)
	h.publishShare(ctx, c, "notebook", pgconv.UUIDString(notebookID), out)
	return connect.NewResponse(&notesv1.ShareNotebookResponse{Share: out}), nil
}

// shareSubjectMember validates the subject/member pairing shared by both share RPCs and
// returns the member id to write: valid for MEMBER, NULL for FAMILY.
//
// What it does NOT do is check that member_user_id is a member of the caller's family: this
// service has no user table and, by the contract's own rule, never calls services/family to
// resolve membership. A share written to a user id outside the family is inert — every read
// predicate is ANDed with the note's family_id, so an outsider's token never reaches the row —
// but it is still a row ListShares will render. Closing it needs either a membership lookup
// or the family's member ids on the claim set; both are contract changes, so the gap is
// recorded here rather than papered over.
// ShareNoteRequest.member_user_id's comment in libs/proto/notes/v1/notes.proto claims this
// check happens and, until one of those lands, overstates what the server does.
func shareSubjectMember(
	subject notesv1.ShareSubject, memberUserID string, c caller,
) (pgtype.UUID, error) {
	switch subject {
	case notesv1.ShareSubject_SHARE_SUBJECT_MEMBER:
		member, err := requireUUID(memberUserID, "member_user_id")
		if err != nil {
			return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
				errors.New("member_user_id is required when subject is MEMBER"))
		}
		if pgconv.UUIDString(member) == c.userID {
			return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
				errors.New("you already own this"))
		}
		return member, nil
	case notesv1.ShareSubject_SHARE_SUBJECT_FAMILY:
		if trimmed(memberUserID) != "" {
			return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
				errors.New("member_user_id must be empty when subject is FAMILY"))
		}
		return pgtype.UUID{}, nil
	default:
		return pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
			errors.New("subject must be MEMBER or FAMILY"))
	}
}

// shareDetail is what an ACTIVITY_KIND_SHARED row carries: the sharee's user id, or empty
// for a whole-family share ("shared this note", with nobody named). It is an id and not a
// name because this service has no user table and never calls services/family; the app
// already maps ids to names for actor_user_id and does the same here.
func shareDetail(memberID pgtype.UUID) string {
	if !memberID.Valid {
		return ""
	}
	return pgconv.UUIDString(memberID)
}

func (h *Handler) publishShare(
	ctx context.Context, c caller, kind, resourceID string, share *notesv1.Share,
) {
	h.publish(ctx, fmevents.SubjectNotesShareGranted, notesevents.ShareEvent(
		c.familyID, kind, resourceID, share.GetMemberUserId(),
		share.GetSubject().String(), share.GetPermission().String(), c.userID, h.now(),
	))
}

func (h *Handler) Unshare(
	ctx context.Context, req *connect.Request[notesv1.UnshareRequest],
) (*connect.Response[notesv1.UnshareResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	shareID, err := requireUUID(req.Msg.GetShareId(), "share_id")
	if err != nil {
		return nil, err
	}
	noteID, notebookID, err := exactlyOneTarget(req.Msg.GetNoteId(), req.Msg.GetNotebookId())
	if err != nil {
		return nil, err
	}

	var rows int64
	if noteID.Valid {
		rows, err = h.q.DeleteNoteShare(ctx, db.DeleteNoteShareParams{
			ShareID: shareID, NoteID: noteID, FamilyID: c.fam, UserID: c.user,
		})
	} else {
		rows, err = h.q.DeleteNotebookShare(ctx, db.DeleteNotebookShareParams{
			ShareID: shareID, NotebookID: notebookID, FamilyID: c.fam, UserID: c.user,
		})
	}
	if err != nil {
		return nil, h.internal(ctx, err, "unshare")
	}
	if rows == 0 {
		// The DELETE carries the ownership check, so this covers "no such share", "not your
		// resource" and "wrong parent" alike — and answering all three the same way is the
		// point: a stranger probing share ids learns nothing from the answer.
		return nil, connect.NewError(connect.CodeNotFound, errors.New("share not found"))
	}
	return connect.NewResponse(&notesv1.UnshareResponse{}), nil
}

func (h *Handler) ListShares(
	ctx context.Context, req *connect.Request[notesv1.ListSharesRequest],
) (*connect.Response[notesv1.ListSharesResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, notebookID, err := exactlyOneTarget(req.Msg.GetNoteId(), req.Msg.GetNotebookId())
	if err != nil {
		return nil, err
	}

	out := []*notesv1.Share{}
	if noteID.Valid {
		rows, err := h.q.ListNoteShares(ctx, db.ListNoteSharesParams{
			NoteID: noteID, FamilyID: c.fam, UserID: c.user,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list note shares")
		}
		for _, s := range rows {
			out = append(out, toProtoNoteShare(s))
		}
	} else {
		rows, err := h.q.ListNotebookShares(ctx, db.ListNotebookSharesParams{
			NotebookID: notebookID, FamilyID: c.fam, UserID: c.user,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list notebook shares")
		}
		for _, s := range rows {
			out = append(out, toProtoNotebookShare(s))
		}
	}
	return connect.NewResponse(&notesv1.ListSharesResponse{Shares: out}), nil
}

// exactlyOneTarget enforces the "exactly one of note_id or notebook_id" rule the Unshare and
// ListShares messages state. Both set is a client bug, not a query to guess at.
func exactlyOneTarget(noteID, notebookID string) (pgtype.UUID, pgtype.UUID, error) {
	hasNote, hasNotebook := trimmed(noteID) != "", trimmed(notebookID) != ""
	if hasNote == hasNotebook {
		return pgtype.UUID{}, pgtype.UUID{}, connect.NewError(connect.CodeInvalidArgument,
			errors.New("set exactly one of note_id or notebook_id"))
	}
	if hasNote {
		id, err := requireUUID(noteID, "note_id")
		return id, pgtype.UUID{}, err
	}
	id, err := requireUUID(notebookID, "notebook_id")
	return pgtype.UUID{}, id, err
}

func (h *Handler) ListSharedWithMe(
	ctx context.Context, req *connect.Request[notesv1.ListSharedWithMeRequest],
) (*connect.Response[notesv1.ListSharedWithMeResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	notes, err := h.q.ListSharedWithMe(ctx, db.ListSharedWithMeParams{
		UserID: c.user, FamilyID: c.fam, PageSize: pageSize(req.Msg.GetPageSize()),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list shared notes")
	}
	notebooks, err := h.q.ListSharedNotebooks(ctx, db.ListSharedNotebooksParams{
		UserID: c.user, FamilyID: c.fam,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list shared notebooks")
	}

	outNotes := make([]*notesv1.Note, 0, len(notes))
	for _, n := range notes {
		outNotes = append(outNotes, toProtoNote(viewFromShared(n), nil))
	}
	outNotebooks := make([]*notesv1.Notebook, 0, len(notebooks))
	for _, nb := range notebooks {
		outNotebooks = append(outNotebooks, toProtoNotebook(viewFromSharedNotebook(nb)))
	}
	return connect.NewResponse(&notesv1.ListSharedWithMeResponse{
		Notes: outNotes, Notebooks: outNotebooks,
	}), nil
}

/* ------------------------------------------------------------------ comments */

func (h *Handler) AddComment(
	ctx context.Context, req *connect.Request[notesv1.AddCommentRequest],
) (*connect.Response[notesv1.AddCommentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	body := trimmed(req.Msg.GetBody())
	if body == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("body is required"))
	}
	if utf8.RuneCountInString(body) > maxBlockText {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("a comment holds at most %d characters", maxBlockText))
	}

	// Commenting needs only read access — a VIEW share is an invitation to say something
	// about the note — and the INSERT ... SELECT carries that predicate itself.
	var comment db.NoteComment
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var err error
		comment, err = q.AddComment(ctx, db.AddCommentParams{
			AuthorUserID: c.user, Body: body, NoteID: noteID, FamilyID: c.fam,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return notFoundNote()
			}
			return h.internal(ctx, err, "add comment")
		}
		if _, err := q.AddActivity(ctx, db.AddActivityParams{
			NoteID:      noteID,
			ActorUserID: c.user,
			Kind:        int16(notesv1.ActivityKind_ACTIVITY_KIND_COMMENTED),
			Detail:      preview(body),
		}); err != nil {
			return h.internal(ctx, err, "record activity")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&notesv1.AddCommentResponse{
		Comment: toProtoComment(comment),
	}), nil
}

func (h *Handler) ListComments(
	ctx context.Context, req *connect.Request[notesv1.ListCommentsRequest],
) (*connect.Response[notesv1.ListCommentsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListComments(ctx, db.ListCommentsParams{
		NoteID:          noteID,
		FamilyID:        c.fam,
		IncludeResolved: req.Msg.GetIncludeResolved(),
		UserID:          c.user,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list comments")
	}
	out := make([]*notesv1.Comment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoComment(r))
	}
	return connect.NewResponse(&notesv1.ListCommentsResponse{Comments: out}), nil
}

func (h *Handler) ResolveComment(
	ctx context.Context, req *connect.Request[notesv1.ResolveCommentRequest],
) (*connect.Response[notesv1.ResolveCommentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	commentID, err := requireUUID(req.Msg.GetCommentId(), "comment_id")
	if err != nil {
		return nil, err
	}
	comment, err := h.q.ResolveComment(ctx, db.ResolveCommentParams{
		Resolved: req.Msg.GetResolved(), CommentID: commentID, FamilyID: c.fam, UserID: c.user,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The statement allows the comment's author or the note's owner. A caller who is
			// neither gets NotFound rather than PermissionDenied: they may not be able to see
			// the note at all, and one answer for both cases leaks nothing.
			return nil, connect.NewError(connect.CodeNotFound, errors.New("comment not found"))
		}
		return nil, h.internal(ctx, err, "resolve comment")
	}
	return connect.NewResponse(&notesv1.ResolveCommentResponse{
		Comment: toProtoComment(comment),
	}), nil
}

/* ------------------------------------------------------------------ activity */

func (h *Handler) ListActivity(
	ctx context.Context, req *connect.Request[notesv1.ListActivityRequest],
) (*connect.Response[notesv1.ListActivityResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	noteID, err := requireUUID(req.Msg.GetNoteId(), "note_id")
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListActivity(ctx, db.ListActivityParams{
		NoteID:     noteID,
		FamilyID:   c.fam,
		UserID:     c.user,
		LimitCount: activityLimit(req.Msg.GetLimit()),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list activity")
	}
	out := make([]*notesv1.Activity, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoActivity(r))
	}
	return connect.NewResponse(&notesv1.ListActivityResponse{Activity: out}), nil
}

/* ------------------------------------------------------------------ errors */

// internal hands the cause to the log and an opaque reference to the caller, so a pgx error
// never becomes part of a response body.
func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

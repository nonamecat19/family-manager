package handler

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	fmevents "github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/notes/db"
)

// fakeStore is an in-memory db.Querier. What is under test is the handler's half of the
// rules — who may read, who may write, what an idempotent create does — so the visibility
// predicate that lives in internal/db/queries/ is reimplemented here rather than mocked away:
// a fake that answered "visible" to everyone would make every privacy test pass by default,
// which is exactly where a fake is worth nothing.
//
// The predicate, mirrored from notes.sql: a note is visible to a user when they own it, OR a
// note_shares row grants it to them (directly, or to the whole family), OR a notebook_shares
// row grants its notebook the same way AND that notebook belongs to the note's owner — always
// ANDed with family_id. can_edit is the same walk with permission = EDIT.
//
// That last clause is the share model and not a detail: a notebook share carries the notes its
// OWNER filed there and nothing else, so filing a note into someone else's folder can never
// publish it and sharing a folder can never hand its owner someone else's note.
type fakeStore struct {
	notes          map[string]db.Note
	notebooks      map[string]db.Notebook
	noteShares     map[string]db.NoteShare
	notebookShares map[string]db.NotebookShare
	comments       map[string]db.NoteComment
	activity       map[string]db.NoteActivity

	// clock advances a millisecond per write so created_at/updated_at ordering is
	// deterministic; map iteration below is not, and an ORDER BY that sorts on equal
	// timestamps would pass or fail by luck.
	clock time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		notes:          map[string]db.Note{},
		notebooks:      map[string]db.Notebook{},
		noteShares:     map[string]db.NoteShare{},
		notebookShares: map[string]db.NotebookShare{},
		comments:       map[string]db.NoteComment{},
		activity:       map[string]db.NoteActivity{},
		clock:          time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

func (s *fakeStore) tick() pgtype.Timestamptz {
	s.clock = s.clock.Add(time.Millisecond)
	return pgtype.Timestamptz{Time: s.clock, Valid: true}
}

/* ------------------------------------------------------------------ predicates */

// sameUUID compares two ids the way SQL does: a NULL never equals anything, itself included.
func sameUUID(a, b pgtype.UUID) bool {
	return a.Valid && b.Valid && a.Bytes == b.Bytes
}

// grantsTo reports whether a share row reaches this user: a FAMILY row reaches everyone in
// the family, a MEMBER row only its member.
func grantsTo(subject int16, member, user pgtype.UUID) bool {
	return subject == 2 || sameUUID(member, user)
}

func (s *fakeStore) noteVisible(n db.Note, user pgtype.UUID) bool {
	return s.noteReach(n, user, false)
}

func (s *fakeStore) noteCanEdit(n db.Note, user pgtype.UUID) bool {
	return s.noteReach(n, user, true)
}

// noteReach is the visibility predicate; editOnly adds the permission = EDIT clause.
func (s *fakeStore) noteReach(n db.Note, user pgtype.UUID, editOnly bool) bool {
	if sameUUID(n.OwnerUserID, user) {
		return true
	}
	for _, sh := range s.noteShares {
		if !sameUUID(sh.NoteID, n.ID) || !grantsTo(sh.Subject, sh.MemberUserID, user) {
			continue
		}
		if !editOnly || sh.Permission == 2 {
			return true
		}
	}
	for _, sh := range s.notebookShares {
		if !sameUUID(sh.NotebookID, n.NotebookID) || !grantsTo(sh.Subject, sh.MemberUserID, user) {
			continue
		}
		// The notebook must be the note owner's own, exactly as the SQL's join to notebooks
		// requires: "share my notebook" shares the notes I keep in it, never yours.
		if !s.notebookOwnedBy(n.NotebookID, n.OwnerUserID) {
			continue
		}
		if !editOnly || sh.Permission == 2 {
			return true
		}
	}
	return false
}

// notebookOwnedBy mirrors `JOIN notebooks nbo ... AND nbo.owner_user_id = n.owner_user_id`.
// A note with no notebook fails it, which is what the SQL's NULL comparison does too.
func (s *fakeStore) notebookOwnedBy(notebookID, owner pgtype.UUID) bool {
	nb, ok := s.notebooks[pgconv.UUIDString(notebookID)]
	return ok && sameUUID(nb.OwnerUserID, owner)
}

// notebookCanEdit mirrors GetNotebook.can_edit: ownership, or a notebook_shares row with
// permission = EDIT. It answers "may this user put a note in here", not "may they rename it".
func (s *fakeStore) notebookCanEdit(nb db.Notebook, user pgtype.UUID) bool {
	if sameUUID(nb.OwnerUserID, user) {
		return true
	}
	for _, sh := range s.notebookShares {
		if sameUUID(sh.NotebookID, nb.ID) && sh.Permission == 2 &&
			grantsTo(sh.Subject, sh.MemberUserID, user) {
			return true
		}
	}
	return false
}

func (s *fakeStore) notebookVisible(nb db.Notebook, user pgtype.UUID) bool {
	if sameUUID(nb.OwnerUserID, user) {
		return true
	}
	for _, sh := range s.notebookShares {
		if sameUUID(sh.NotebookID, nb.ID) && grantsTo(sh.Subject, sh.MemberUserID, user) {
			return true
		}
	}
	return false
}

// noteShared mirrors the `shared` column — the list row's people glyph. It is "can anybody but
// the owner reach this note", so it counts the notebook share as well as the direct ones, and
// counts it only when the notebook is the note owner's (a note parked in someone else's shared
// folder is reachable by nobody).
func (s *fakeStore) noteShared(n db.Note) bool {
	for _, sh := range s.noteShares {
		if sameUUID(sh.NoteID, n.ID) {
			return true
		}
	}
	if !s.notebookOwnedBy(n.NotebookID, n.OwnerUserID) {
		return false
	}
	for _, sh := range s.notebookShares {
		if sameUUID(sh.NotebookID, n.NotebookID) {
			return true
		}
	}
	return false
}

// noteCount is the CALLER's count of a notebook's notes, the way the SQL computes it.
func (s *fakeStore) noteCount(notebookID, user pgtype.UUID) int32 {
	var n int32
	for _, note := range s.notes {
		if sameUUID(note.NotebookID, notebookID) && !note.Archived && s.noteVisible(note, user) {
			n++
		}
	}
	return n
}

/* ------------------------------------------------------------------ blocks */

// rawBlock reads the stored jsonb the way the queries do: `b->>'type'` and `b->>'checked'`,
// not the proto. The task counters and SearchTasks both go through here.
type rawBlock struct {
	ID      string `json:"id"`
	Type    int32  `json:"type"`
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}

func rawBlocks(encoded []byte) []rawBlock {
	if len(encoded) == 0 {
		return nil
	}
	var out []rawBlock
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}
	return out
}

func taskCounts(encoded []byte) (total, done int32) {
	for _, b := range rawBlocks(encoded) {
		if b.Type != int32(3) { // BLOCK_TYPE_TODO
			continue
		}
		total++
		if b.Checked {
			done++
		}
	}
	return total, done
}

// sqlPreview is left(body_text, 140).
func sqlPreview(body string) string {
	runes := []rune(body)
	if len(runes) <= previewLimit {
		return body
	}
	return string(runes[:previewLimit])
}

// matches stands in for `search @@ websearch_to_tsquery('simple', query)`. A substring match
// is not a tsquery, but it agrees with one on the only thing these tests assert: whether a
// note the caller cannot see can ever appear in a result set.
func matches(haystack, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), q)
}

/* ------------------------------------------------------------------ notes */

func (s *fakeStore) CreateNote(_ context.Context, arg db.CreateNoteParams) (db.CreateNoteRow, error) {
	n := s.insertNote(arg.FamilyID, arg.OwnerUserID, arg.NotebookID, arg.ClientID,
		arg.Title, arg.Blocks, arg.BodyText)
	return db.CreateNoteRow(noteRow(n)), nil
}

func (s *fakeStore) CreateNoteByClientID(
	_ context.Context, arg db.CreateNoteByClientIDParams,
) (db.CreateNoteByClientIDRow, error) {
	// The partial unique index is (family_id, owner_user_id, client_id) WHERE client_id <> '',
	// and DO UPDATE with a no-op assignment returns the row the FIRST call inserted.
	if arg.ClientID != "" {
		for _, n := range s.notes {
			if sameUUID(n.FamilyID, arg.FamilyID) && sameUUID(n.OwnerUserID, arg.OwnerUserID) &&
				n.ClientID == arg.ClientID {
				return db.CreateNoteByClientIDRow(noteRow(n)), nil
			}
		}
	}
	n := s.insertNote(arg.FamilyID, arg.OwnerUserID, arg.NotebookID, arg.ClientID,
		arg.Title, arg.Blocks, arg.BodyText)
	return db.CreateNoteByClientIDRow(noteRow(n)), nil
}

func (s *fakeStore) insertNote(
	familyID, ownerID, notebookID pgtype.UUID, clientID, title string, blocks []byte, body string,
) db.Note {
	at := s.tick()
	n := db.Note{
		ID:                 pgconv.MustUUID(newUUID()),
		FamilyID:           familyID,
		OwnerUserID:        ownerID,
		NotebookID:         notebookID,
		ClientID:           clientID,
		Title:              title,
		Blocks:             blocks,
		BodyText:           body,
		Version:            1,
		LastEditedByUserID: ownerID,
		CreatedAt:          at,
		UpdatedAt:          at,
	}
	s.notes[pgconv.UUIDString(n.ID)] = n
	return n
}

// noteRow is the column list every notes statement returns. sqlc generates a distinct struct
// per statement with identical fields, so one conversion feeds them all.
func noteRow(n db.Note) db.CreateNoteRow {
	return db.CreateNoteRow{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, NotebookID: n.NotebookID,
		ClientID: n.ClientID, Title: n.Title, Blocks: n.Blocks, BodyText: n.BodyText,
		Starred: n.Starred, Archived: n.Archived, Version: n.Version,
		LastEditedByUserID: n.LastEditedByUserID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
	}
}

func (s *fakeStore) GetNote(_ context.Context, arg db.GetNoteParams) (db.GetNoteRow, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
		return db.GetNoteRow{}, pgx.ErrNoRows
	}
	total, done := taskCounts(n.Blocks)
	return db.GetNoteRow{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, NotebookID: n.NotebookID,
		ClientID: n.ClientID, Title: n.Title, Blocks: n.Blocks, BodyText: n.BodyText,
		Starred: n.Starred, Archived: n.Archived, Version: n.Version,
		LastEditedByUserID: n.LastEditedByUserID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
		Shared: s.noteShared(n), TaskTotal: total, TaskDone: done,
		Preview: sqlPreview(n.BodyText), CanEdit: s.noteCanEdit(n, arg.UserID),
	}, nil
}

func (s *fakeStore) GetNoteEditPermission(
	_ context.Context, arg db.GetNoteEditPermissionParams,
) (db.GetNoteEditPermissionRow, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
		return db.GetNoteEditPermissionRow{}, pgx.ErrNoRows
	}
	return db.GetNoteEditPermissionRow{
		ID: n.ID, OwnerUserID: n.OwnerUserID, NotebookID: n.NotebookID, Version: n.Version,
		CanEdit: s.noteCanEdit(n, arg.UserID),
	}, nil
}

func (s *fakeStore) ListNotes(_ context.Context, arg db.ListNotesParams) ([]db.ListNotesRow, error) {
	var kept []db.Note
	for _, n := range s.notes {
		if !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
			continue
		}
		if arg.NotebookID.Valid && !sameUUID(n.NotebookID, arg.NotebookID) {
			continue
		}
		if arg.StarredOnly && !n.Starred {
			continue
		}
		// archived_only implies include_archived, and then keeps only the archived rows.
		if !arg.IncludeArchived && !arg.ArchivedOnly && n.Archived {
			continue
		}
		if arg.ArchivedOnly && !n.Archived {
			continue
		}
		if arg.SharedOnly && sameUUID(n.OwnerUserID, arg.UserID) {
			continue
		}
		kept = append(kept, n)
	}
	sortNotes(kept, arg.Sort)
	kept = limitNotes(kept, arg.PageSize)

	out := make([]db.ListNotesRow, 0, len(kept))
	for _, n := range kept {
		total, done := taskCounts(n.Blocks)
		out = append(out, db.ListNotesRow{
			ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, NotebookID: n.NotebookID,
			ClientID: n.ClientID, Title: n.Title, Blocks: n.Blocks, BodyText: n.BodyText,
			Starred: n.Starred, Archived: n.Archived, Version: n.Version,
			LastEditedByUserID: n.LastEditedByUserID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
			Shared: s.noteShared(n), TaskTotal: total, TaskDone: done,
			Preview: sqlPreview(n.BodyText), CanEdit: s.noteCanEdit(n, arg.UserID),
		})
	}
	return out, nil
}

// sortNotes mirrors the ORDER BY in ListNotes.
func sortNotes(ns []db.Note, key string) {
	sort.SliceStable(ns, func(i, j int) bool {
		a, b := ns[i], ns[j]
		switch key {
		case "title":
			return strings.ToLower(strings.TrimSpace(a.Title)) <
				strings.ToLower(strings.TrimSpace(b.Title))
		case "created":
			return a.CreatedAt.Time.After(b.CreatedAt.Time)
		default:
			return a.UpdatedAt.Time.After(b.UpdatedAt.Time)
		}
	})
}

// limitNotes mirrors LIMIT NULLIF(page_size, 0): zero means unlimited.
func limitNotes(ns []db.Note, pageSize int32) []db.Note {
	if pageSize > 0 && int(pageSize) < len(ns) {
		return ns[:pageSize]
	}
	return ns
}

func (s *fakeStore) ListSharedWithMe(
	_ context.Context, arg db.ListSharedWithMeParams,
) ([]db.ListSharedWithMeRow, error) {
	var kept []db.Note
	for _, n := range s.notes {
		if !sameUUID(n.FamilyID, arg.FamilyID) || sameUUID(n.OwnerUserID, arg.UserID) {
			continue
		}
		if n.Archived || !s.noteVisible(n, arg.UserID) {
			continue
		}
		kept = append(kept, n)
	}
	sortNotes(kept, "updated")
	kept = limitNotes(kept, arg.PageSize)

	out := make([]db.ListSharedWithMeRow, 0, len(kept))
	for _, n := range kept {
		total, done := taskCounts(n.Blocks)
		out = append(out, db.ListSharedWithMeRow{
			ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, NotebookID: n.NotebookID,
			ClientID: n.ClientID, Title: n.Title, Blocks: n.Blocks, BodyText: n.BodyText,
			Starred: n.Starred, Archived: n.Archived, Version: n.Version,
			LastEditedByUserID: n.LastEditedByUserID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
			Shared: s.noteShared(n), TaskTotal: total, TaskDone: done,
			Preview: sqlPreview(n.BodyText), CanEdit: s.noteCanEdit(n, arg.UserID),
		})
	}
	return out, nil
}

func (s *fakeStore) UpdateNote(_ context.Context, arg db.UpdateNoteParams) (db.UpdateNoteRow, error) {
	key := pgconv.UUIDString(arg.NoteID)
	n, ok := s.notes[key]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) {
		return db.UpdateNoteRow{}, pgx.ErrNoRows
	}
	// Both guards are in the WHERE clause and both answer with no row, which is why the
	// handler re-reads the permission rather than assuming a conflict: the write predicate
	// (owner, or an EDIT share on the note or on its owner's notebook)...
	if !s.noteCanEdit(n, arg.UserID) {
		return db.UpdateNoteRow{}, pgx.ErrNoRows
	}
	// ...and expected_version: 0 forces the write, anything else must match.
	if arg.ExpectedVersion != 0 && n.Version != arg.ExpectedVersion {
		return db.UpdateNoteRow{}, pgx.ErrNoRows
	}
	n.Title = arg.Title
	n.Blocks = arg.Blocks
	n.BodyText = arg.BodyText
	n.Version++
	n.LastEditedByUserID = arg.UserID
	n.UpdatedAt = s.tick()
	s.notes[key] = n
	return db.UpdateNoteRow(noteRow(n)), nil
}

func (s *fakeStore) MoveNote(_ context.Context, arg db.MoveNoteParams) (db.MoveNoteRow, error) {
	key := pgconv.UUIDString(arg.NoteID)
	n, ok := s.notes[key]
	// The statement is scoped by id + family_id + owner_user_id: filing a note is the
	// owner's, because where it sits decides who else can read it.
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !sameUUID(n.OwnerUserID, arg.UserID) {
		return db.MoveNoteRow{}, pgx.ErrNoRows
	}
	n.NotebookID = arg.NotebookID
	n.LastEditedByUserID = arg.UserID
	n.UpdatedAt = s.tick()
	s.notes[key] = n
	return db.MoveNoteRow(noteRow(n)), nil
}

func (s *fakeStore) SetNoteStarred(
	_ context.Context, arg db.SetNoteStarredParams,
) (db.SetNoteStarredRow, error) {
	key := pgconv.UUIDString(arg.NoteID)
	n, ok := s.notes[key]
	// The statement carries the write predicate itself — owner, or an EDIT share on the note or
	// on its notebook — so a caller who only has sight of the note writes nothing here even if
	// the handler's gate were ever removed. Mirrored, not mocked away, like SetNoteArchived.
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteCanEdit(n, arg.UserID) {
		return db.SetNoteStarredRow{}, pgx.ErrNoRows
	}
	// Starring bumps neither version, last_edited_by_user_id nor updated_at — see the query.
	n.Starred = arg.Starred
	s.notes[key] = n
	return db.SetNoteStarredRow(noteRow(n)), nil
}

func (s *fakeStore) SetNoteArchived(
	_ context.Context, arg db.SetNoteArchivedParams,
) (db.SetNoteArchivedRow, error) {
	key := pgconv.UUIDString(arg.NoteID)
	n, ok := s.notes[key]
	// The statement carries the write predicate itself — owner, or an EDIT share on the note
	// or on its notebook — so a caller who only has sight of the note writes nothing here even
	// if the handler's gate were ever removed. Mirrored, not mocked away.
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteCanEdit(n, arg.UserID) {
		return db.SetNoteArchivedRow{}, pgx.ErrNoRows
	}
	n.Archived = arg.Archived
	n.UpdatedAt = s.tick()
	s.notes[key] = n
	return db.SetNoteArchivedRow(noteRow(n)), nil
}

func (s *fakeStore) DeleteNote(_ context.Context, arg db.DeleteNoteParams) (int64, error) {
	key := pgconv.UUIDString(arg.NoteID)
	n, ok := s.notes[key]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !sameUUID(n.OwnerUserID, arg.UserID) {
		return 0, nil
	}
	delete(s.notes, key)
	// ON DELETE CASCADE.
	for id, sh := range s.noteShares {
		if sameUUID(sh.NoteID, n.ID) {
			delete(s.noteShares, id)
		}
	}
	for id, c := range s.comments {
		if sameUUID(c.NoteID, n.ID) {
			delete(s.comments, id)
		}
	}
	for id, a := range s.activity {
		if sameUUID(a.NoteID, n.ID) {
			delete(s.activity, id)
		}
	}
	return 1, nil
}

/* ------------------------------------------------------------------ notebooks */

func (s *fakeStore) CreateNotebook(
	_ context.Context, arg db.CreateNotebookParams,
) (db.Notebook, error) {
	at := s.tick()
	nb := db.Notebook{
		ID:          pgconv.MustUUID(newUUID()),
		FamilyID:    arg.FamilyID,
		OwnerUserID: arg.OwnerUserID,
		ParentID:    arg.ParentID,
		Name:        arg.Name,
		CreatedAt:   at,
		UpdatedAt:   at,
	}
	s.notebooks[pgconv.UUIDString(nb.ID)] = nb
	return nb, nil
}

func (s *fakeStore) GetNotebook(
	_ context.Context, arg db.GetNotebookParams,
) (db.GetNotebookRow, error) {
	nb, ok := s.notebooks[pgconv.UUIDString(arg.NotebookID)]
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) || !s.notebookVisible(nb, arg.UserID) {
		return db.GetNotebookRow{}, pgx.ErrNoRows
	}
	return db.GetNotebookRow{
		ID: nb.ID, FamilyID: nb.FamilyID, OwnerUserID: nb.OwnerUserID, ParentID: nb.ParentID,
		Name: nb.Name, Archived: nb.Archived, CreatedAt: nb.CreatedAt, UpdatedAt: nb.UpdatedAt,
		NoteCount: s.noteCount(nb.ID, arg.UserID),
		CanEdit:   s.notebookCanEdit(nb, arg.UserID),
	}, nil
}

func (s *fakeStore) ListNotebooks(
	_ context.Context, arg db.ListNotebooksParams,
) ([]db.ListNotebooksRow, error) {
	var kept []db.Notebook
	for _, nb := range s.notebooks {
		if !sameUUID(nb.FamilyID, arg.FamilyID) || !s.notebookVisible(nb, arg.UserID) {
			continue
		}
		if !arg.IncludeArchived && nb.Archived {
			continue
		}
		kept = append(kept, nb)
	}
	sortNotebooks(kept)
	out := make([]db.ListNotebooksRow, 0, len(kept))
	for _, nb := range kept {
		out = append(out, db.ListNotebooksRow{
			ID: nb.ID, FamilyID: nb.FamilyID, OwnerUserID: nb.OwnerUserID, ParentID: nb.ParentID,
			Name: nb.Name, Archived: nb.Archived, CreatedAt: nb.CreatedAt, UpdatedAt: nb.UpdatedAt,
			NoteCount: s.noteCount(nb.ID, arg.UserID),
		})
	}
	return out, nil
}

func (s *fakeStore) ListSharedNotebooks(
	_ context.Context, arg db.ListSharedNotebooksParams,
) ([]db.ListSharedNotebooksRow, error) {
	var kept []db.Notebook
	for _, nb := range s.notebooks {
		if !sameUUID(nb.FamilyID, arg.FamilyID) || sameUUID(nb.OwnerUserID, arg.UserID) {
			continue
		}
		if nb.Archived || !s.notebookVisible(nb, arg.UserID) {
			continue
		}
		kept = append(kept, nb)
	}
	sortNotebooks(kept)
	out := make([]db.ListSharedNotebooksRow, 0, len(kept))
	for _, nb := range kept {
		out = append(out, db.ListSharedNotebooksRow{
			ID: nb.ID, FamilyID: nb.FamilyID, OwnerUserID: nb.OwnerUserID, ParentID: nb.ParentID,
			Name: nb.Name, Archived: nb.Archived, CreatedAt: nb.CreatedAt, UpdatedAt: nb.UpdatedAt,
			NoteCount: s.noteCount(nb.ID, arg.UserID),
		})
	}
	return out, nil
}

func sortNotebooks(nbs []db.Notebook) {
	sort.SliceStable(nbs, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(nbs[i].Name)) <
			strings.ToLower(strings.TrimSpace(nbs[j].Name))
	})
}

func (s *fakeStore) UpdateNotebook(
	_ context.Context, arg db.UpdateNotebookParams,
) (db.Notebook, error) {
	key := pgconv.UUIDString(arg.NotebookID)
	nb, ok := s.notebooks[key]
	// The statement carries the ownership check itself.
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) || !sameUUID(nb.OwnerUserID, arg.UserID) {
		return db.Notebook{}, pgx.ErrNoRows
	}
	nb.Name = arg.Name
	nb.ParentID = arg.ParentID
	nb.Archived = arg.Archived
	nb.UpdatedAt = s.tick()
	s.notebooks[key] = nb
	return nb, nil
}

func (s *fakeStore) DeleteNotebook(_ context.Context, arg db.DeleteNotebookParams) (int64, error) {
	key := pgconv.UUIDString(arg.NotebookID)
	nb, ok := s.notebooks[key]
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) || !sameUUID(nb.OwnerUserID, arg.UserID) {
		return 0, nil
	}
	// NOT EXISTS (SELECT 1 FROM notes WHERE notebook_id = notebooks.id
	//             AND notes.family_id = notebooks.family_id).
	for _, n := range s.notes {
		if sameUUID(n.NotebookID, nb.ID) && sameUUID(n.FamilyID, nb.FamilyID) {
			return 0, nil
		}
	}
	delete(s.notebooks, key)
	for id, sh := range s.notebookShares {
		if sameUUID(sh.NotebookID, nb.ID) {
			delete(s.notebookShares, id)
		}
	}
	return 1, nil
}

// CountNotebookNotes returns both counts the statement does: the total ignores visibility on
// purpose (it backs DeleteNotebook's refusal), the visible one is what the refusal may name.
// Both are scoped by family_id like the statement.
// NotebookAncestors mirrors the recursive CTE: parent_id upward from the starting notebook,
// family-scoped, no visibility predicate, and depth-capped so a cycle already in the map
// terminates the walk instead of hanging the test.
func (s *fakeStore) NotebookAncestors(
	_ context.Context, arg db.NotebookAncestorsParams,
) ([]pgtype.UUID, error) {
	var out []pgtype.UUID
	id := arg.NotebookID
	for depth := 0; depth < 64 && id.Valid; depth++ {
		nb, ok := s.notebooks[pgconv.UUIDString(id)]
		if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) {
			break
		}
		out = append(out, nb.ID)
		id = nb.ParentID
	}
	return out, nil
}

func (s *fakeStore) CountNotebookNotes(
	_ context.Context, arg db.CountNotebookNotesParams,
) (db.CountNotebookNotesRow, error) {
	var out db.CountNotebookNotesRow
	for _, note := range s.notes {
		if !sameUUID(note.NotebookID, arg.NotebookID) || !sameUUID(note.FamilyID, arg.FamilyID) {
			continue
		}
		out.Total++
		if s.noteVisible(note, arg.UserID) {
			out.Visible++
		}
	}
	return out, nil
}

/* ------------------------------------------------------------------ shares */

func (s *fakeStore) ShareNoteWithMember(
	_ context.Context, arg db.ShareNoteWithMemberParams,
) (db.NoteShare, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	// INSERT ... SELECT over the OWNED row: a non-owner writes nothing and gets no row.
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) ||
		!sameUUID(n.OwnerUserID, arg.GrantedByUserID) {
		return db.NoteShare{}, pgx.ErrNoRows
	}
	// ON CONFLICT (note_id, member_user_id) WHERE subject = 1.
	for id, sh := range s.noteShares {
		if sh.Subject == 1 && sameUUID(sh.NoteID, arg.NoteID) &&
			sameUUID(sh.MemberUserID, arg.MemberUserID) {
			sh.Permission = arg.Permission
			sh.GrantedByUserID = arg.GrantedByUserID
			s.noteShares[id] = sh
			return sh, nil
		}
	}
	sh := db.NoteShare{
		ID: pgconv.MustUUID(newUUID()), NoteID: n.ID, FamilyID: n.FamilyID, Subject: 1,
		MemberUserID: arg.MemberUserID, Permission: arg.Permission,
		GrantedByUserID: arg.GrantedByUserID, CreatedAt: s.tick(),
	}
	s.noteShares[pgconv.UUIDString(sh.ID)] = sh
	return sh, nil
}

func (s *fakeStore) ShareNoteWithFamily(
	_ context.Context, arg db.ShareNoteWithFamilyParams,
) (db.NoteShare, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) ||
		!sameUUID(n.OwnerUserID, arg.GrantedByUserID) {
		return db.NoteShare{}, pgx.ErrNoRows
	}
	for id, sh := range s.noteShares {
		if sh.Subject == 2 && sameUUID(sh.NoteID, arg.NoteID) {
			sh.Permission = arg.Permission
			sh.GrantedByUserID = arg.GrantedByUserID
			s.noteShares[id] = sh
			return sh, nil
		}
	}
	sh := db.NoteShare{
		ID: pgconv.MustUUID(newUUID()), NoteID: n.ID, FamilyID: n.FamilyID, Subject: 2,
		Permission: arg.Permission, GrantedByUserID: arg.GrantedByUserID, CreatedAt: s.tick(),
	}
	s.noteShares[pgconv.UUIDString(sh.ID)] = sh
	return sh, nil
}

func (s *fakeStore) ShareNotebookWithMember(
	_ context.Context, arg db.ShareNotebookWithMemberParams,
) (db.NotebookShare, error) {
	nb, ok := s.notebooks[pgconv.UUIDString(arg.NotebookID)]
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) ||
		!sameUUID(nb.OwnerUserID, arg.GrantedByUserID) {
		return db.NotebookShare{}, pgx.ErrNoRows
	}
	for id, sh := range s.notebookShares {
		if sh.Subject == 1 && sameUUID(sh.NotebookID, arg.NotebookID) &&
			sameUUID(sh.MemberUserID, arg.MemberUserID) {
			sh.Permission = arg.Permission
			sh.GrantedByUserID = arg.GrantedByUserID
			s.notebookShares[id] = sh
			return sh, nil
		}
	}
	sh := db.NotebookShare{
		ID: pgconv.MustUUID(newUUID()), NotebookID: nb.ID, FamilyID: nb.FamilyID, Subject: 1,
		MemberUserID: arg.MemberUserID, Permission: arg.Permission,
		GrantedByUserID: arg.GrantedByUserID, CreatedAt: s.tick(),
	}
	s.notebookShares[pgconv.UUIDString(sh.ID)] = sh
	return sh, nil
}

func (s *fakeStore) ShareNotebookWithFamily(
	_ context.Context, arg db.ShareNotebookWithFamilyParams,
) (db.NotebookShare, error) {
	nb, ok := s.notebooks[pgconv.UUIDString(arg.NotebookID)]
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) ||
		!sameUUID(nb.OwnerUserID, arg.GrantedByUserID) {
		return db.NotebookShare{}, pgx.ErrNoRows
	}
	for id, sh := range s.notebookShares {
		if sh.Subject == 2 && sameUUID(sh.NotebookID, arg.NotebookID) {
			sh.Permission = arg.Permission
			sh.GrantedByUserID = arg.GrantedByUserID
			s.notebookShares[id] = sh
			return sh, nil
		}
	}
	sh := db.NotebookShare{
		ID: pgconv.MustUUID(newUUID()), NotebookID: nb.ID, FamilyID: nb.FamilyID, Subject: 2,
		Permission: arg.Permission, GrantedByUserID: arg.GrantedByUserID, CreatedAt: s.tick(),
	}
	s.notebookShares[pgconv.UUIDString(sh.ID)] = sh
	return sh, nil
}

func (s *fakeStore) DeleteNoteShare(_ context.Context, arg db.DeleteNoteShareParams) (int64, error) {
	key := pgconv.UUIDString(arg.ShareID)
	sh, ok := s.noteShares[key]
	if !ok || !sameUUID(sh.NoteID, arg.NoteID) || !sameUUID(sh.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	n, ok := s.notes[pgconv.UUIDString(sh.NoteID)]
	if !ok || !sameUUID(n.OwnerUserID, arg.UserID) {
		return 0, nil
	}
	delete(s.noteShares, key)
	return 1, nil
}

func (s *fakeStore) DeleteNotebookShare(
	_ context.Context, arg db.DeleteNotebookShareParams,
) (int64, error) {
	key := pgconv.UUIDString(arg.ShareID)
	sh, ok := s.notebookShares[key]
	if !ok || !sameUUID(sh.NotebookID, arg.NotebookID) || !sameUUID(sh.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	nb, ok := s.notebooks[pgconv.UUIDString(sh.NotebookID)]
	if !ok || !sameUUID(nb.OwnerUserID, arg.UserID) {
		return 0, nil
	}
	delete(s.notebookShares, key)
	return 1, nil
}

func (s *fakeStore) ListNoteShares(
	_ context.Context, arg db.ListNoteSharesParams,
) ([]db.NoteShare, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
		return nil, nil
	}
	var out []db.NoteShare
	for _, sh := range s.noteShares {
		if sameUUID(sh.NoteID, n.ID) {
			out = append(out, sh)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time)
	})
	return out, nil
}

func (s *fakeStore) ListNotebookShares(
	_ context.Context, arg db.ListNotebookSharesParams,
) ([]db.NotebookShare, error) {
	nb, ok := s.notebooks[pgconv.UUIDString(arg.NotebookID)]
	if !ok || !sameUUID(nb.FamilyID, arg.FamilyID) || !s.notebookVisible(nb, arg.UserID) {
		return nil, nil
	}
	var out []db.NotebookShare
	for _, sh := range s.notebookShares {
		if sameUUID(sh.NotebookID, nb.ID) {
			out = append(out, sh)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time)
	})
	return out, nil
}

/* ------------------------------------------------------------------ search */

func (s *fakeStore) CountVisibleNotes(
	_ context.Context, arg db.CountVisibleNotesParams,
) (int32, error) {
	var n int32
	for _, note := range s.notes {
		if sameUUID(note.FamilyID, arg.FamilyID) && !note.Archived &&
			s.noteVisible(note, arg.UserID) {
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) SearchNotes(
	_ context.Context, arg db.SearchNotesParams,
) ([]db.SearchNotesRow, error) {
	var kept []db.Note
	for _, n := range s.notes {
		if !sameUUID(n.FamilyID, arg.FamilyID) || n.Archived {
			continue
		}
		if !matches(n.Title+" "+n.BodyText, arg.Query) {
			continue
		}
		if !s.noteVisible(n, arg.UserID) {
			continue
		}
		kept = append(kept, n)
	}
	sortNotes(kept, "updated")
	kept = limitNotes(kept, arg.LimitCount)

	out := make([]db.SearchNotesRow, 0, len(kept))
	for _, n := range kept {
		out = append(out, db.SearchNotesRow{
			ID: n.ID, NotebookID: n.NotebookID, Title: n.Title, OwnerUserID: n.OwnerUserID,
			UpdatedAt: n.UpdatedAt, Rank: 1, Snippet: []byte(n.BodyText),
			NotebookName: s.notebookName(n.NotebookID, arg.FamilyID, arg.UserID),
		})
	}
	return out, nil
}

func (s *fakeStore) SearchTasks(
	_ context.Context, arg db.SearchTasksParams,
) ([]db.SearchTasksRow, error) {
	var kept []db.Note
	for _, n := range s.notes {
		if !sameUUID(n.FamilyID, arg.FamilyID) || n.Archived {
			continue
		}
		if !matches(n.Title+" "+n.BodyText, arg.Query) || !s.noteVisible(n, arg.UserID) {
			continue
		}
		kept = append(kept, n)
	}
	sortNotes(kept, "updated")

	out := make([]db.SearchTasksRow, 0, len(kept))
	for _, n := range kept {
		for _, b := range rawBlocks(n.Blocks) {
			if b.Type != 3 || !matches(b.Text, arg.Query) {
				continue
			}
			out = append(out, db.SearchTasksRow{
				NoteID: n.ID, NotebookID: n.NotebookID, Title: n.Title, UpdatedAt: n.UpdatedAt,
				BlockID: b.ID, Snippet: b.Text, Checked: b.Checked, Rank: 1,
				NotebookName: s.notebookName(n.NotebookID, arg.FamilyID, arg.UserID),
			})
		}
	}
	if arg.LimitCount > 0 && int(arg.LimitCount) < len(out) {
		out = out[:arg.LimitCount]
	}
	return out, nil
}

func (s *fakeStore) SearchNotebooks(
	_ context.Context, arg db.SearchNotebooksParams,
) ([]db.SearchNotebooksRow, error) {
	var kept []db.Notebook
	for _, nb := range s.notebooks {
		if !sameUUID(nb.FamilyID, arg.FamilyID) || nb.Archived {
			continue
		}
		if !matches(nb.Name, arg.Query) || !s.notebookVisible(nb, arg.UserID) {
			continue
		}
		kept = append(kept, nb)
	}
	sortNotebooks(kept)
	if arg.LimitCount > 0 && int(arg.LimitCount) < len(kept) {
		kept = kept[:arg.LimitCount]
	}
	out := make([]db.SearchNotebooksRow, 0, len(kept))
	for _, nb := range kept {
		out = append(out, db.SearchNotebooksRow{
			ID: nb.ID, Name: nb.Name, ParentID: nb.ParentID, UpdatedAt: nb.UpdatedAt,
			ParentName: s.notebookName(nb.ParentID, arg.FamilyID, arg.UserID),
		})
	}
	return out, nil
}

// notebookName mirrors the breadcrumb subselects: the name comes back only when the caller
// can see that notebook, coalesced to "" otherwise, so a hit shared out of a private notebook
// does not drag the notebook's name along with it.
func (s *fakeStore) notebookName(id pgtype.UUID, family, user pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	nb, ok := s.notebooks[pgconv.UUIDString(id)]
	if !ok || !sameUUID(nb.FamilyID, family) || !s.notebookVisible(nb, user) {
		return ""
	}
	return nb.Name
}

/* ------------------------------------------------------------------ comments & activity */

func (s *fakeStore) AddComment(_ context.Context, arg db.AddCommentParams) (db.NoteComment, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	// INSERT ... SELECT over the VISIBLE note: read access is enough to comment.
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.AuthorUserID) {
		return db.NoteComment{}, pgx.ErrNoRows
	}
	c := db.NoteComment{
		ID: pgconv.MustUUID(newUUID()), NoteID: n.ID, AuthorUserID: arg.AuthorUserID,
		Body: arg.Body, CreatedAt: s.tick(),
	}
	s.comments[pgconv.UUIDString(c.ID)] = c
	return c, nil
}

func (s *fakeStore) ListComments(
	_ context.Context, arg db.ListCommentsParams,
) ([]db.NoteComment, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
		return nil, nil
	}
	var out []db.NoteComment
	for _, c := range s.comments {
		if !sameUUID(c.NoteID, n.ID) {
			continue
		}
		if !arg.IncludeResolved && c.Resolved {
			continue
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Time.Before(out[j].CreatedAt.Time)
	})
	return out, nil
}

func (s *fakeStore) ResolveComment(
	_ context.Context, arg db.ResolveCommentParams,
) (db.NoteComment, error) {
	key := pgconv.UUIDString(arg.CommentID)
	c, ok := s.comments[key]
	if !ok {
		return db.NoteComment{}, pgx.ErrNoRows
	}
	n, ok := s.notes[pgconv.UUIDString(c.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) {
		return db.NoteComment{}, pgx.ErrNoRows
	}
	// The comment's author or the note's owner — nobody else closes someone else's thread.
	if !sameUUID(c.AuthorUserID, arg.UserID) && !sameUUID(n.OwnerUserID, arg.UserID) {
		return db.NoteComment{}, pgx.ErrNoRows
	}
	// ANDed with visibility, mirroring the statement: authoring a comment does not outlive the
	// access that produced it.
	if !s.noteVisible(n, arg.UserID) {
		return db.NoteComment{}, pgx.ErrNoRows
	}
	c.Resolved = arg.Resolved
	s.comments[key] = c
	return c, nil
}

func (s *fakeStore) AddActivity(
	_ context.Context, arg db.AddActivityParams,
) (db.NoteActivity, error) {
	a := db.NoteActivity{
		ID: pgconv.MustUUID(newUUID()), NoteID: arg.NoteID, ActorUserID: arg.ActorUserID,
		Kind: arg.Kind, Detail: arg.Detail, CreatedAt: s.tick(),
	}
	s.activity[pgconv.UUIDString(a.ID)] = a
	return a, nil
}

func (s *fakeStore) AddNotebookShareActivity(
	_ context.Context, arg db.AddNotebookShareActivityParams,
) error {
	for _, n := range s.notes {
		if !sameUUID(n.NotebookID, arg.NotebookID) || !sameUUID(n.FamilyID, arg.FamilyID) {
			continue
		}
		if n.Archived {
			continue
		}
		// The notebook-share rule: the share reaches only the notes the notebook's owner filed
		// there, so only those notes get the activity row.
		if nb, ok := s.notebooks[pgconv.UUIDString(arg.NotebookID)]; !ok ||
			!sameUUID(nb.OwnerUserID, n.OwnerUserID) {
			continue
		}
		a := db.NoteActivity{
			ID: pgconv.MustUUID(newUUID()), NoteID: n.ID, ActorUserID: arg.ActorUserID,
			Kind: arg.Kind, Detail: arg.Detail, CreatedAt: s.tick(),
		}
		s.activity[pgconv.UUIDString(a.ID)] = a
	}
	return nil
}

func (s *fakeStore) ListActivity(
	_ context.Context, arg db.ListActivityParams,
) ([]db.NoteActivity, error) {
	n, ok := s.notes[pgconv.UUIDString(arg.NoteID)]
	if !ok || !sameUUID(n.FamilyID, arg.FamilyID) || !s.noteVisible(n, arg.UserID) {
		return nil, nil
	}
	var out []db.NoteActivity
	for _, a := range s.activity {
		if sameUUID(a.NoteID, n.ID) {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.Time.After(out[j].CreatedAt.Time)
	})
	if arg.LimitCount > 0 && int(arg.LimitCount) < len(out) {
		out = out[:arg.LimitCount]
	}
	return out, nil
}

/* ------------------------------------------------------------------ transactions */

// InTx makes the fake satisfy Tx, and rolls back for real.
//
// A fake that ran the callback and kept whatever it wrote would let every atomicity test pass
// whether or not the handler used a transaction at all. Instead it snapshots the maps and
// restores them when the callback returns an error — the observable half of what Postgres
// does, which is what the handler's behaviour depends on.
func (s *fakeStore) InTx(_ context.Context, fn func(db.Querier) error) error {
	undo := s.snapshot()
	if err := fn(s); err != nil {
		undo()
		return err
	}
	return nil
}

func (s *fakeStore) snapshot() func() {
	notes := maps.Clone(s.notes)
	notebooks := maps.Clone(s.notebooks)
	noteShares := maps.Clone(s.noteShares)
	notebookShares := maps.Clone(s.notebookShares)
	comments := maps.Clone(s.comments)
	activity := maps.Clone(s.activity)
	// Blocks is the one []byte in the row; the handler replaces it wholesale, never appends.
	for k, n := range notes {
		n.Blocks = slices.Clone(n.Blocks)
		notes[k] = n
	}
	return func() {
		s.notes, s.notebooks = notes, notebooks
		s.noteShares, s.notebookShares = noteShares, notebookShares
		s.comments, s.activity = comments, activity
	}
}

/* ------------------------------------------------------------------ helpers */

// recorder is the EventBus these tests pass instead of standing up NATS.
type recorder struct {
	published []fmevents.Subject
	err       error
}

func (r *recorder) Publish(_ context.Context, subject fmevents.Subject, _ proto.Message) error {
	if r.err != nil {
		return r.err
	}
	r.published = append(r.published, subject)
	return nil
}

func (r *recorder) sawSubject(s fmevents.Subject) bool {
	return slices.Contains(r.published, s)
}

func (r *recorder) count(s fmevents.Subject) int {
	n := 0
	for _, got := range r.published {
		if got == s {
			n++
		}
	}
	return n
}

// uuidCounter is atomic so `go test -race` stays quiet even if a future test parallelises.
var uuidCounter atomic.Int64

func newUUID() string {
	n := uuidCounter.Add(1)
	// A v4-shaped id whose tail is the counter, so a failure message names a readable id.
	const hex = "0123456789abcdef"
	tail := make([]byte, 12)
	for i := 11; i >= 0; i-- {
		tail[i] = hex[n&0xf]
		n >>= 4
	}
	return "00000000-0000-4000-8000-" + string(tail)
}

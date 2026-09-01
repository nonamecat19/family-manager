package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	fmevents "github.com/nnc/family-manager/libs/go/events"
	notesv1 "github.com/nnc/family-manager/sdk/go/notes/v1"
	"github.com/nnc/family-manager/services/notes/db"
)

// EventBus is the slice of libs/go/events this service uses. Narrow on purpose: tests pass a
// recorder instead of standing up NATS.
type EventBus interface {
	Publish(ctx context.Context, subject fmevents.Subject, msg proto.Message) error
}

// noopBus lets the service run (and tests pass) with no broker attached.
type noopBus struct{}

func (noopBus) Publish(context.Context, fmevents.Subject, proto.Message) error { return nil }

// publish is fire-and-forget by design: a note that was saved must not be reported as failed
// because the broker hiccuped. The failure is logged, not returned.
func (h *Handler) publish(ctx context.Context, subject fmevents.Subject, msg proto.Message) {
	if err := h.bus.Publish(ctx, subject, msg); err != nil {
		h.log.WarnContext(ctx, "publish failed", slog.String("subject", string(subject)),
			slog.String("error", err.Error()))
	}
}

func trimmed(s string) string { return strings.TrimSpace(s) }

/* ------------------------------------------------------------------ blocks */

// storedBlock is a Block as it sits in the notes.blocks jsonb column.
//
// It is hand-written rather than protojson because the queries read the column: the task
// counters and SearchTasks match `b->>'type'`, and the id has to stay `id` for a comment to
// point at a block. Encoding through protojson would make that key set the marshaller's
// choice (lowerCamel, omitted zero values, enum names) instead of this file's.
//
// The enum is stored as its NUMBER. The queries accept both encodings — `IN ('3',
// 'BLOCK_TYPE_TODO')` — but only one of them is ever written, and it is this one.
type storedBlock struct {
	ID       string `json:"id"`
	Type     int32  `json:"type"`
	Text     string `json:"text"`
	Checked  bool   `json:"checked"`
	Level    int32  `json:"level"`
	ImageURL string `json:"image_url"`
	Language string `json:"language"`
}

// maxBlocks bounds one note. A note is a short document; a client sending more than this is
// looping, and the array is read back by every list row.
const maxBlocks = 2000

// maxBlockText bounds one block's text, so a single paste cannot make a note that no list
// query can afford to scan. Counted in RUNES wherever it is applied, like every other free
// text bound here.
const maxBlockText = 100_000

// maxTitle and maxNotebookName bound the two single-line fields a client sends. Both feed the
// generated tsvector and both are drawn in a list row, so "whatever fits in a 16 MiB request"
// is not a bound: a title nobody can read still costs every search that scans it.
const (
	maxTitle        = 500
	maxNotebookName = 200
)

// encodeBlocks turns the wire blocks into the stored array. It never yields JSON `null`: the
// column is jsonb NOT NULL and jsonb_array_elements() errors on a non-array, which would turn
// an empty note into a failing list query for the whole family.
func encodeBlocks(blocks []*notesv1.Block) ([]byte, error) {
	out := make([]storedBlock, 0, len(blocks))
	for _, b := range blocks {
		if b == nil {
			continue
		}
		out = append(out, storedBlock{
			ID:       b.GetId(),
			Type:     int32(b.GetType()),
			Text:     b.GetText(),
			Checked:  b.GetChecked(),
			Level:    b.GetLevel(),
			ImageURL: b.GetImageUrl(),
			Language: b.GetLanguage(),
		})
	}
	return json.Marshal(out)
}

// decodeBlocks reads the stored array back. A column this service cannot parse is reported as
// an empty block list rather than an error: the note's title, sharing and activity are still
// worth serving, and the alternative is one bad row breaking a whole list response.
func decodeBlocks(raw []byte) []*notesv1.Block {
	if len(raw) == 0 {
		return nil
	}
	var stored []storedBlock
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil
	}
	out := make([]*notesv1.Block, 0, len(stored))
	for _, s := range stored {
		out = append(out, &notesv1.Block{
			Id:       s.ID,
			Type:     notesv1.BlockType(s.Type),
			Text:     s.Text,
			Checked:  s.Checked,
			Level:    s.Level,
			ImageUrl: s.ImageURL,
			Language: s.Language,
		})
	}
	return out
}

// bodyText is the flattened note: every block's text, newline-joined, in order. It is what
// the generated tsvector indexes and what preview is cut from, so it is recomputed on every
// write rather than stored by the client.
func bodyText(blocks []*notesv1.Block) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if t := trimmed(b.GetText()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}

// previewLimit matches left(body_text, 140) in the queries, so a preview computed here and
// one computed by Postgres are the same string.
const previewLimit = 140

// preview is the first ~140 characters of the flattened text, cut on a rune boundary — the
// SQL cuts on characters too, and half a multi-byte rune is not a character.
func preview(body string) string {
	if utf8.RuneCountInString(body) <= previewLimit {
		return body
	}
	count := 0
	for i := range body {
		if count == previewLimit {
			return body[:i]
		}
		count++
	}
	return body
}

/* ------------------------------------------------------------------ notes */

// noteView is the one note shape the converter renders. sqlc generates a separate row struct
// per statement — GetNoteRow, ListNotesRow, ListSharedWithMeRow — with identical columns, and
// Go will not let one function take all three. Funnelling them through this struct keeps a
// single toProtoNote instead of three that drift.
type noteView struct {
	ID                 pgtype.UUID
	FamilyID           pgtype.UUID
	OwnerUserID        pgtype.UUID
	NotebookID         pgtype.UUID
	Title              string
	Blocks             []byte
	Starred            bool
	Archived           bool
	Version            int64
	LastEditedByUserID pgtype.UUID
	CreatedAt          pgtype.Timestamptz
	UpdatedAt          pgtype.Timestamptz
	Shared             bool
	TaskTotal          int32
	TaskDone           int32
	Preview            string
	CanEdit            bool
}

func viewFromGet(r db.GetNoteRow) noteView {
	return noteView{
		ID: r.ID, FamilyID: r.FamilyID, OwnerUserID: r.OwnerUserID, NotebookID: r.NotebookID,
		Title: r.Title, Blocks: r.Blocks, Starred: r.Starred, Archived: r.Archived,
		Version: r.Version, LastEditedByUserID: r.LastEditedByUserID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Shared: r.Shared,
		TaskTotal: r.TaskTotal, TaskDone: r.TaskDone, Preview: r.Preview, CanEdit: r.CanEdit,
	}
}

func viewFromList(r db.ListNotesRow) noteView {
	return noteView{
		ID: r.ID, FamilyID: r.FamilyID, OwnerUserID: r.OwnerUserID, NotebookID: r.NotebookID,
		Title: r.Title, Blocks: r.Blocks, Starred: r.Starred, Archived: r.Archived,
		Version: r.Version, LastEditedByUserID: r.LastEditedByUserID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Shared: r.Shared,
		TaskTotal: r.TaskTotal, TaskDone: r.TaskDone, Preview: r.Preview, CanEdit: r.CanEdit,
	}
}

func viewFromShared(r db.ListSharedWithMeRow) noteView {
	return noteView{
		ID: r.ID, FamilyID: r.FamilyID, OwnerUserID: r.OwnerUserID, NotebookID: r.NotebookID,
		Title: r.Title, Blocks: r.Blocks, Starred: r.Starred, Archived: r.Archived,
		Version: r.Version, LastEditedByUserID: r.LastEditedByUserID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Shared: r.Shared,
		TaskTotal: r.TaskTotal, TaskDone: r.TaskDone, Preview: r.Preview, CanEdit: r.CanEdit,
	}
}

// toProtoNote renders one note. shares is nil for list rows: GetNote fills them, ListNotes
// leaves them empty (Note.shares says so), and `shared` is the cheap flag the list row draws.
func toProtoNote(v noteView, shares []*notesv1.Share) *notesv1.Note {
	return &notesv1.Note{
		Id:                 pgconv.UUIDString(v.ID),
		FamilyId:           pgconv.UUIDString(v.FamilyID),
		OwnerUserId:        pgconv.UUIDString(v.OwnerUserID),
		NotebookId:         pgconv.UUIDString(v.NotebookID),
		Title:              v.Title,
		Blocks:             decodeBlocks(v.Blocks),
		Starred:            v.Starred,
		Archived:           v.Archived,
		Version:            v.Version,
		LastEditedByUserId: pgconv.UUIDString(v.LastEditedByUserID),
		CreatedAt:          pgconv.Timestamp(v.CreatedAt),
		UpdatedAt:          pgconv.Timestamp(v.UpdatedAt),
		Shares:             shares,
		Shared:             v.Shared,
		TaskTotal:          v.TaskTotal,
		TaskDone:           v.TaskDone,
		Preview:            v.Preview,
		CanEdit:            v.CanEdit,
	}
}

/* ------------------------------------------------------------------ notebooks */

// notebookView is noteView's counterpart: CreateNotebook and UpdateNotebook return the plain
// db.Notebook, the reads return a row carrying note_count.
type notebookView struct {
	ID          pgtype.UUID
	FamilyID    pgtype.UUID
	OwnerUserID pgtype.UUID
	ParentID    pgtype.UUID
	Name        string
	Archived    bool
	NoteCount   int32
	CreatedAt   pgtype.Timestamptz
	UpdatedAt   pgtype.Timestamptz
}

func viewFromNotebook(n db.Notebook, noteCount int32) notebookView {
	return notebookView{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, ParentID: n.ParentID,
		Name: n.Name, Archived: n.Archived, NoteCount: noteCount,
		CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
	}
}

func viewFromNotebookRow(n db.ListNotebooksRow) notebookView {
	return notebookView{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, ParentID: n.ParentID,
		Name: n.Name, Archived: n.Archived, NoteCount: n.NoteCount,
		CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
	}
}

func viewFromGetNotebook(n db.GetNotebookRow) notebookView {
	return notebookView{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, ParentID: n.ParentID,
		Name: n.Name, Archived: n.Archived, NoteCount: n.NoteCount,
		CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
	}
}

func viewFromSharedNotebook(n db.ListSharedNotebooksRow) notebookView {
	return notebookView{
		ID: n.ID, FamilyID: n.FamilyID, OwnerUserID: n.OwnerUserID, ParentID: n.ParentID,
		Name: n.Name, Archived: n.Archived, NoteCount: n.NoteCount,
		CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
	}
}

func toProtoNotebook(v notebookView) *notesv1.Notebook {
	return &notesv1.Notebook{
		Id:          pgconv.UUIDString(v.ID),
		FamilyId:    pgconv.UUIDString(v.FamilyID),
		OwnerUserId: pgconv.UUIDString(v.OwnerUserID),
		ParentId:    pgconv.UUIDString(v.ParentID),
		Name:        v.Name,
		Archived:    v.Archived,
		NoteCount:   v.NoteCount,
		CreatedAt:   pgconv.Timestamp(v.CreatedAt),
		UpdatedAt:   pgconv.Timestamp(v.UpdatedAt),
	}
}

/* ------------------------------------------------------------------ shares */

func toProtoNoteShare(s db.NoteShare) *notesv1.Share {
	return &notesv1.Share{
		Id:              pgconv.UUIDString(s.ID),
		Subject:         subjectToProto(s.Subject),
		MemberUserId:    pgconv.UUIDString(s.MemberUserID),
		Permission:      permissionToProto(s.Permission),
		GrantedByUserId: pgconv.UUIDString(s.GrantedByUserID),
		CreatedAt:       pgconv.Timestamp(s.CreatedAt),
	}
}

func toProtoNotebookShare(s db.NotebookShare) *notesv1.Share {
	return &notesv1.Share{
		Id:              pgconv.UUIDString(s.ID),
		Subject:         subjectToProto(s.Subject),
		MemberUserId:    pgconv.UUIDString(s.MemberUserID),
		Permission:      permissionToProto(s.Permission),
		GrantedByUserId: pgconv.UUIDString(s.GrantedByUserID),
		CreatedAt:       pgconv.Timestamp(s.CreatedAt),
	}
}

// The share enums are stored as their proto numbers, and the CHECK constraints in the
// migration pin the accepted set — so these conversions are casts with a floor rather than
// switch tables, and an out-of-range row reads back as UNSPECIFIED instead of a lie.
func subjectToProto(v int16) notesv1.ShareSubject {
	switch v {
	case int16(notesv1.ShareSubject_SHARE_SUBJECT_MEMBER):
		return notesv1.ShareSubject_SHARE_SUBJECT_MEMBER
	case int16(notesv1.ShareSubject_SHARE_SUBJECT_FAMILY):
		return notesv1.ShareSubject_SHARE_SUBJECT_FAMILY
	default:
		return notesv1.ShareSubject_SHARE_SUBJECT_UNSPECIFIED
	}
}

func permissionToProto(v int16) notesv1.SharePermission {
	switch v {
	case int16(notesv1.SharePermission_SHARE_PERMISSION_VIEW):
		return notesv1.SharePermission_SHARE_PERMISSION_VIEW
	case int16(notesv1.SharePermission_SHARE_PERMISSION_EDIT):
		return notesv1.SharePermission_SHARE_PERMISSION_EDIT
	default:
		return notesv1.SharePermission_SHARE_PERMISSION_UNSPECIFIED
	}
}

// permissionFromProto defaults to VIEW. An unspecified permission must not silently become
// EDIT: the safe default for a grant is the narrower one.
func permissionFromProto(p notesv1.SharePermission) int16 {
	if p == notesv1.SharePermission_SHARE_PERMISSION_EDIT {
		return int16(notesv1.SharePermission_SHARE_PERMISSION_EDIT)
	}
	return int16(notesv1.SharePermission_SHARE_PERMISSION_VIEW)
}

/* ------------------------------------------------------------------ comments & activity */

func toProtoComment(c db.NoteComment) *notesv1.Comment {
	return &notesv1.Comment{
		Id:           pgconv.UUIDString(c.ID),
		NoteId:       pgconv.UUIDString(c.NoteID),
		AuthorUserId: pgconv.UUIDString(c.AuthorUserID),
		Body:         c.Body,
		Resolved:     c.Resolved,
		CreatedAt:    pgconv.Timestamp(c.CreatedAt),
	}
}

func toProtoActivity(a db.NoteActivity) *notesv1.Activity {
	return &notesv1.Activity{
		Id:          pgconv.UUIDString(a.ID),
		NoteId:      pgconv.UUIDString(a.NoteID),
		ActorUserId: pgconv.UUIDString(a.ActorUserID),
		Kind:        activityKindToProto(a.Kind),
		Detail:      a.Detail,
		CreatedAt:   pgconv.Timestamp(a.CreatedAt),
	}
}

func activityKindToProto(v int16) notesv1.ActivityKind {
	if _, ok := notesv1.ActivityKind_name[int32(v)]; ok {
		return notesv1.ActivityKind(v)
	}
	return notesv1.ActivityKind_ACTIVITY_KIND_UNSPECIFIED
}

/* ------------------------------------------------------------------ list & search inputs */

// sortKey maps the closed NoteSort enum onto the discriminator ListNotes switches on. An
// unknown value falls through to recently-updated rather than erroring — a client on a newer
// contract should get a list, not a 400.
func sortKey(s notesv1.NoteSort) string {
	switch s {
	case notesv1.NoteSort_NOTE_SORT_TITLE:
		return "title"
	case notesv1.NoteSort_NOTE_SORT_CREATED:
		return "created"
	default:
		return "updated"
	}
}

// maxPageSize bounds a list. 0 means "no limit" to the query, so a client asking for
// everything gets this instead.
const maxPageSize = 500

// pageSize floors a negative page size at 0 (the query's "unlimited" sentinel) and caps the
// rest, so one request cannot ask for the whole family's history.
func pageSize(v int32) int32 {
	if v <= 0 || v > maxPageSize {
		return maxPageSize
	}
	return v
}

// searchLimit is per facet. The palette shows a short list; ALL runs three queries and takes
// the first `limit` of the concatenation.
const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

func searchLimit(v int32) int32 {
	if v <= 0 {
		return defaultSearchLimit
	}
	if v > maxSearchLimit {
		return maxSearchLimit
	}
	return v
}

// activityLimit mirrors searchLimit for the activity rail.
const (
	defaultActivityLimit = 50
	maxActivityLimit     = 200
)

func activityLimit(v int32) int32 {
	if v <= 0 {
		return defaultActivityLimit
	}
	if v > maxActivityLimit {
		return maxActivityLimit
	}
	return v
}

// breadcrumb builds SearchHit.context. The parts the server knows are the notebook and the
// parent notebook; it deliberately does not invent the person and the date the design shows,
// because this service has no name for a user id — services/family owns that, and asking it
// per hit would put a fan-out inside the palette's latency budget.
func breadcrumb(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := trimmed(p); t != "" {
			kept = append(kept, t)
		}
	}
	return strings.Join(kept, " / ")
}

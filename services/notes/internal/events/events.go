// Package events builds the payloads the notes service publishes. The subjects themselves
// live in libs/go/events with every other domain's — SubjectNotesNoteCreated,
// SubjectNotesNoteUpdated, SubjectNotesShareGranted — because that file is the repo-wide
// registry and a subject declared anywhere else is a subject nobody can find.
//
// One deviation from the other services remains, and it is deliberate: the payloads are
// structpb rather than generated event messages. notes.proto declares no *Event messages
// (recipes.proto and finance.proto do), and the contract is frozen — so the alternative to a
// struct is publishing nothing at all. A consumer reading these reads named string fields,
// which is what a generated message would have given it.
package events

import (
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

// Domain is the JetStream stream these subjects belong to; main.go ensures it at boot.
const Domain = "notes"

// NoteEvent describes something that happened to one note. actorUserID is who did it, which
// is not necessarily the note's owner once the note is shared for EDIT.
func NoteEvent(familyID, noteID, actorUserID string, occurredAt time.Time) *structpb.Struct {
	return newStruct(map[string]any{
		"family_id":     familyID,
		"note_id":       noteID,
		"actor_user_id": actorUserID,
		"occurred_at":   occurredAt.UTC().Format(time.RFC3339Nano),
	})
}

// ShareEvent describes a grant. resourceKind is "note" or "notebook"; memberUserID is empty
// for a whole-family grant, mirroring the Share message on the wire.
func ShareEvent(
	familyID, resourceKind, resourceID, memberUserID, subject, permission, actorUserID string,
	occurredAt time.Time,
) *structpb.Struct {
	return newStruct(map[string]any{
		"family_id":      familyID,
		"resource_kind":  resourceKind,
		"resource_id":    resourceID,
		"member_user_id": memberUserID,
		"subject":        subject,
		"permission":     permission,
		"actor_user_id":  actorUserID,
		"occurred_at":    occurredAt.UTC().Format(time.RFC3339Nano),
	})
}

// newStruct cannot fail for the string-only maps above; an empty struct rather than a panic
// is still the right answer if that ever stops being true, because a publish is best-effort.
func newStruct(fields map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(fields)
	if err != nil {
		return &structpb.Struct{}
	}
	return s
}

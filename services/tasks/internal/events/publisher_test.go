package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	baseevents "github.com/nnc/family-manager/libs/go/events"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
)

type fakeBus struct {
	stream    string
	subject   baseevents.Subject
	message   proto.Message
	streamErr error
}

func (b *fakeBus) EnsureStream(_ context.Context, domain string) error {
	b.stream = domain
	return b.streamErr
}

func (b *fakeBus) Publish(_ context.Context, subject baseevents.Subject, msg proto.Message) error {
	b.subject = subject
	b.message = msg
	return nil
}

func TestAssignedPublishesAfterEnsuringStream(t *testing.T) {
	bus := &fakeBus{}
	at := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	task := db.Task{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-000000000001"),
		FamilyID: pgconv.MustUUID("00000000-0000-4000-8000-000000000002"),
		Title:    "Pay rent",
	}
	if err := NewPublisher(bus).Assigned(context.Background(), task, []string{"user-a"}, "user-b", at); err != nil {
		t.Fatal(err)
	}
	got, ok := bus.message.(*tasksv1.TaskAssignedEvent)
	if bus.stream != "tasks" || bus.subject != baseevents.SubjectTasksTaskAssigned || !ok {
		t.Fatalf("publication = %q %q %T", bus.stream, bus.subject, bus.message)
	}
	if got.GetTaskId() != pgconv.UUIDString(task.ID) || got.GetFamilyId() != pgconv.UUIDString(task.FamilyID) || got.GetTitle() != task.Title || got.GetAssignedByUserId() != "user-b" || len(got.GetAssigneeUserIds()) != 1 || got.GetAssigneeUserIds()[0] != "user-a" || !got.GetOccurredAt().AsTime().Equal(at) {
		t.Errorf("event = %+v", got)
	}
}

func TestAssignedSkipsEmptyAndStopsWhenStreamUnavailable(t *testing.T) {
	bus := &fakeBus{}
	p := NewPublisher(bus)
	if err := p.Assigned(context.Background(), db.Task{}, nil, "", time.Time{}); err != nil || bus.stream != "" {
		t.Fatalf("empty assignment called stream: %q, %v", bus.stream, err)
	}
	want := errors.New("stream unavailable")
	bus.streamErr = want
	if err := p.Assigned(context.Background(), db.Task{}, []string{"user"}, "", time.Time{}); !errors.Is(err, want) || bus.message != nil {
		t.Fatalf("stream failure: %v, publication %T", err, bus.message)
	}
}

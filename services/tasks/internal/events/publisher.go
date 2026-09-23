package events

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	baseevents "github.com/nnc/family-manager/libs/go/events"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
)

type Bus interface {
	EnsureStream(context.Context, string) error
	Publish(context.Context, baseevents.Subject, proto.Message) error
}

type Publisher struct {
	bus Bus
}

func NewPublisher(bus Bus) *Publisher { return &Publisher{bus: bus} }

func (p *Publisher) Assigned(ctx context.Context, task db.Task, assignees []string, assignedBy string, at time.Time) error {
	if len(assignees) == 0 {
		return nil
	}
	if err := p.bus.EnsureStream(ctx, "tasks"); err != nil {
		return fmt.Errorf("tasks events: ensure stream: %w", err)
	}
	return p.bus.Publish(ctx, baseevents.SubjectTasksTaskAssigned, &tasksv1.TaskAssignedEvent{
		FamilyId:         pgconv.UUIDString(task.FamilyID),
		TaskId:           pgconv.UUIDString(task.ID),
		Title:            task.Title,
		AssigneeUserIds:  assignees,
		AssignedByUserId: assignedBy,
		OccurredAt:       timestamppb.New(at),
	})
}

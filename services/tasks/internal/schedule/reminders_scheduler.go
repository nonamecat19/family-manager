package schedule

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/tasks/db"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
)

type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

type ReminderPoster interface {
	PostDueReminders(ctx context.Context) (int, error)
}

type Reminders struct {
	poster ReminderPoster
	every  time.Duration
	log    *slog.Logger
}

func NewReminders(poster ReminderPoster, every time.Duration, log *slog.Logger) *Reminders {
	return &Reminders{poster: poster, every: every, log: log}
}

func (s *Reminders) Run(ctx context.Context) {
	if s.every <= 0 {
		s.log.Info("reminder scheduler disabled")
		return
	}
	ticker := time.NewTicker(s.every)
	defer ticker.Stop()
	for {
		s.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Reminders) tick(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, min(s.every, maxSchedulerTick))
	defer cancel()
	posted, err := s.poster.PostDueReminders(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.log.Error("post due reminders", slog.String("error", err.Error()))
		}
		return
	}
	if posted > 0 {
		s.log.Info("reminders posted", slog.Int("count", posted))
	}
}

const maxSchedulerTick = 30 * time.Second

type reminderPoster struct {
	q   db.Querier
	bus EventBus
	now func() time.Time
	log *slog.Logger
}

func NewReminderPoster(q db.Querier, bus EventBus, now func() time.Time, log *slog.Logger) *reminderPoster {
	return &reminderPoster{q: q, bus: bus, now: now, log: log}
}

func (p *reminderPoster) PostDueReminders(ctx context.Context) (int, error) {
	now := p.now()
	rows, err := p.q.ListDueReminders(ctx, db.ListDueRemindersParams{
		Now:     pgconv.TimestampFrom(now),
		MaxRows: 100,
	})
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	var posted int
	for _, r := range rows {
		select {
		case <-ctx.Done():
			return posted, ctx.Err()
		default:
		}
		if err := p.postReminder(ctx, r); err != nil {
			p.log.ErrorContext(ctx, "post reminder",
				slog.String("error", err.Error()),
				slog.String("reminder_id", pgconv.UUIDString(r.ID)),
				slog.String("user_id", pgconv.UUIDString(r.UserID)),
			)
			continue
		}
		posted++
	}
	return posted, nil
}

func (p *reminderPoster) postReminder(ctx context.Context, r db.TaskReminder) error {
	kind := r.Kind
	var subject events.Subject
	var msg proto.Message

	switch kind {
	case "task_due":
		task, err := p.q.GetTask(ctx, db.GetTaskParams{ID: r.ItemID, FamilyID: r.FamilyID})
		if err != nil {
			return err
		}
		assignees, err := p.q.ListAssignees(ctx, db.ListAssigneesParams{
			FamilyID: r.FamilyID,
			TaskIds:  []pgtype.UUID{r.ItemID},
		})
		if err != nil {
			return err
		}
		var assigneeIDs []string
		for _, a := range assignees {
			assigneeIDs = append(assigneeIDs, pgconv.UUIDString(a.UserID))
		}

		subject = events.SubjectTasksTaskDue
		msg = &tasksv1.TaskDueEvent{
			FamilyId:           pgconv.UUIDString(r.FamilyID),
			TaskId:             pgconv.UUIDString(r.ItemID),
			Title:              task.Title,
			AssigneeUserIds:    assigneeIDs,
			DueAt:              timestamppb.New(task.DueAt.Time),
			OccurredAt:         timestamppb.New(p.now()),
		}
	case "birthday":
		birthday, err := p.q.GetBirthday(ctx, db.GetBirthdayParams{ID: r.ItemID, FamilyID: r.FamilyID})
		if err != nil {
			return err
		}

		// Parse the occurrence key to get days_until
		// The occurrence format is "YYYY-MM-DD" or "YYYY-MM-DD-Nd"
		daysUntil := int32(0)
		nextOn := ""
		turningAge := int32(0)

		// TODO: properly parse occurrence and calculate days_until, turning_age
		// For now, use the occurrence as next_on
		nextOn = r.Occurrence

		subject = events.SubjectTasksBirthdayUpcoming
		msg = &tasksv1.BirthdayUpcomingEvent{
			FamilyId:    pgconv.UUIDString(r.FamilyID),
			BirthdayId:  pgconv.UUIDString(r.ItemID),
			Name:        birthday.Name,
			NextOn:      nextOn,
			DaysUntil:   daysUntil,
			TurningAge:  turningAge,
			OccurredAt:  timestamppb.New(p.now()),
		}
	default:
		return errors.New("unknown reminder kind: " + kind)
	}

	if p.bus != nil {
		if err := p.bus.Publish(ctx, subject, msg); err != nil {
			return err
		}
	}

	_, err := p.q.AckReminder(ctx, db.AckReminderParams{
		ID:        r.ID,
		FamilyID:  r.FamilyID,
		UserID:    r.UserID,
	})
	return err
}
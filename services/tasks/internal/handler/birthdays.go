package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

func (h *Handler) ListBirthdays(
	ctx context.Context, req *connect.Request[tasksv1.ListBirthdaysRequest],
) (*connect.Response[tasksv1.ListBirthdaysResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	rows, err := h.q.ListBirthdays(ctx, c.familyID)
	if err != nil {
		return nil, h.internal(ctx, err, "list birthdays")
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	birthdays := make([]*tasksv1.Birthday, 0, len(rows))
	for _, b := range rows {
		birthdays = append(birthdays, h.toProtoBirthday(b, now, loc))
	}
	return connect.NewResponse(&tasksv1.ListBirthdaysResponse{Birthdays: birthdays}), nil
}

func (h *Handler) CreateBirthday(
	ctx context.Context, req *connect.Request[tasksv1.CreateBirthdayRequest],
) (*connect.Response[tasksv1.CreateBirthdayResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	msg := req.Msg
	if err := checkText("name", msg.GetName(), maxTitleRunes); err != nil {
		return nil, err
	}

	day := int(msg.GetDay())
	month := int(msg.GetMonth())
	year := int(msg.GetYear())
	remindDays := int(msg.GetRemindDaysBefore())

	if err := schedule.ValidateBirthday(day, month, year, remindDays, h.now()); err != nil {
		return nil, invalid("%v", err)
	}

	var birthday db.Birthday
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var innerErr error
		birthday, innerErr = q.CreateBirthday(ctx, db.CreateBirthdayParams{
			FamilyID:          c.familyID,
			Name:              msg.GetName(),
			Day:               int32(day),
			Month:             int32(month),
			Year:              yearPtr(year),
			RemindDaysBefore:  int32(remindDays),
			CreatedByUserID:   c.userID,
		})
		if innerErr != nil {
			return innerErr
		}

		// Refresh known_members via family client
		if h.familyPub != nil {
			members, innerErr := h.familyPub.ListMembers(ctx, c.user, c.family)
			if innerErr == nil {
				keepUsers := make([]pgtype.UUID, 0, len(members))
				for _, m := range members {
					uid, _ := pgconv.UUID(m.UserID)
					keepUsers = append(keepUsers, uid)
				}
				if len(keepUsers) > 0 {
					_ = q.DeleteKnownMembersExcept(ctx, db.DeleteKnownMembersExceptParams{
						FamilyID: c.familyID,
						Keep:     keepUsers,
					})
				}
			}
		}

		// Create birthday reminders for all known members
		occ := schedule.NextOccurrence(day, month, year, h.now().In(loc))
		if !occ.On.IsZero() {
			occStr := occ.On.Format("2006-01-02")
			remindAts := schedule.BirthdayRemindAt(occ, remindDays, loc, h.now())
			for _, remindAt := range remindAts {
				rows, innerErr := q.ListKnownMembers(ctx, c.familyID)
				if innerErr != nil {
					return innerErr
				}
				for _, m := range rows {
					if innerErr = q.UpsertReminder(ctx, db.UpsertReminderParams{
						FamilyID:   c.familyID,
						UserID:     m.UserID,
						Kind:       "birthday",
						ItemID:     birthday.ID,
						Occurrence: occStr,
						RemindAt:   pgTimestamptz(remindAt),
					}); innerErr != nil {
						return innerErr
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create birthday")
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	return connect.NewResponse(&tasksv1.CreateBirthdayResponse{Birthday: h.toProtoBirthday(birthday, now, loc)}), nil
}

func (h *Handler) UpdateBirthday(
	ctx context.Context, req *connect.Request[tasksv1.UpdateBirthdayRequest],
) (*connect.Response[tasksv1.UpdateBirthdayResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	msg := req.Msg
	birthdayID, err := requireUUID("birthday_id", msg.GetBirthdayId())
	if err != nil {
		return nil, err
	}

	var birthday db.Birthday
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		existing, innerErr := q.GetBirthday(ctx, db.GetBirthdayParams{ID: birthdayID, FamilyID: c.familyID})
		if errors.Is(innerErr, pgx.ErrNoRows) {
			return notFound("birthday")
		}
		if innerErr != nil {
			return h.internal(ctx, innerErr, "get birthday")
		}

		name := existing.Name
		if msg.Name != nil {
			if err := checkText("name", msg.GetName(), maxTitleRunes); err != nil {
				return err
			}
			name = msg.GetName()
		}

		day := int(existing.Day)
		if msg.Day != nil {
			day = int(msg.GetDay())
		}
		month := int(existing.Month)
		if msg.Month != nil {
			month = int(msg.GetMonth())
		}
		year := 0
		if existing.Year != nil {
			year = int(*existing.Year)
		}
		if msg.Year != nil {
			year = int(msg.GetYear())
		}
		remindDays := int(existing.RemindDaysBefore)
		if msg.RemindDaysBefore != nil {
			remindDays = int(msg.GetRemindDaysBefore())
		}

		if err := schedule.ValidateBirthday(day, month, year, remindDays, h.now()); err != nil {
			return invalid("%v", err)
		}

		var yearPtr *int32
		if year > 0 {
			y := int32(year)
			yearPtr = &y
		}

		birthday, innerErr = q.UpdateBirthday(ctx, db.UpdateBirthdayParams{
			ID:                birthdayID,
			FamilyID:          c.familyID,
			Name:              name,
			Day:               int32(day),
			Month:             int32(month),
			Year:              yearPtr,
			RemindDaysBefore:  int32(remindDays),
		})
		if innerErr != nil {
			return innerErr
		}

		// Delete old reminders for this birthday
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: "birthday", ItemID: birthdayID,
		}); innerErr != nil {
			return innerErr
		}

		// Refresh known_members
		if h.familyPub != nil {
			members, innerErr := h.familyPub.ListMembers(ctx, c.user, c.family)
			if innerErr == nil {
				keepUsers := make([]pgtype.UUID, 0, len(members))
				for _, m := range members {
					uid, _ := pgconv.UUID(m.UserID)
					keepUsers = append(keepUsers, uid)
				}
				if len(keepUsers) > 0 {
					_ = q.DeleteKnownMembersExcept(ctx, db.DeleteKnownMembersExceptParams{
						FamilyID: c.familyID,
						Keep:     keepUsers,
					})
				}
			}
		}

		// Create new reminders
		occ := schedule.NextOccurrence(day, month, year, h.now().In(loc))
		if !occ.On.IsZero() {
			occStr := occ.On.Format("2006-01-02")
			remindAts := schedule.BirthdayRemindAt(occ, remindDays, loc, h.now())
			for _, remindAt := range remindAts {
				rows, innerErr := q.ListKnownMembers(ctx, c.familyID)
				if innerErr != nil {
					return innerErr
				}
				for _, m := range rows {
					if innerErr = q.UpsertReminder(ctx, db.UpsertReminderParams{
						FamilyID:   c.familyID,
						UserID:     m.UserID,
						Kind:       "birthday",
						ItemID:     birthdayID,
						Occurrence: occStr,
						RemindAt:   pgTimestamptz(remindAt),
					}); innerErr != nil {
						return innerErr
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	now := h.now()
	loc, _ := time.LoadLocation("UTC")
	if hh, err := h.household(ctx, c); err == nil {
		loc = hh.loc
	}

	return connect.NewResponse(&tasksv1.UpdateBirthdayResponse{Birthday: h.toProtoBirthday(birthday, now, loc)}), nil
}

func (h *Handler) DeleteBirthday(
	ctx context.Context, req *connect.Request[tasksv1.DeleteBirthdayRequest],
) (*connect.Response[tasksv1.DeleteBirthdayResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.touchKnownMember(ctx, c); err != nil {
		return nil, h.internal(ctx, err, "touch known member")
	}

	birthdayID, err := requireUUID("birthday_id", req.Msg.GetBirthdayId())
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		rows, innerErr := q.DeleteBirthday(ctx, db.DeleteBirthdayParams{
			ID: birthdayID, FamilyID: c.familyID,
		})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "delete birthday")
		}
		if rows == 0 {
			return notFound("birthday")
		}

		// Delete pending birthday reminders
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: "birthday", ItemID: birthdayID,
		}); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.DeleteBirthdayResponse{}), nil
}

func (h *Handler) toProtoBirthday(b db.Birthday, now time.Time, loc *time.Location) *tasksv1.Birthday {
	year := int32(0)
	if b.Year != nil {
		year = *b.Year
	}

	occ := schedule.NextOccurrence(int(b.Day), int(b.Month), int(year), now)

	age := int32(0)
	hasAge := false
	if b.Year != nil {
		age = int32(occ.TurningAge)
		hasAge = true
	}

	return &tasksv1.Birthday{
		Id:                  pgconv.UUIDString(b.ID),
		FamilyId:            pgconv.UUIDString(b.FamilyID),
		Name:                b.Name,
		Day:                 b.Day,
		Month:               b.Month,
		Year:                year,
		RemindDaysBefore:    b.RemindDaysBefore,
		NextOn:              occ.On.Format(schedule.DateLayout),
		DaysUntil:           int32(occ.DaysUntil),
		TurningAge:          age,
		CreatedAt:           timestamppb.New(b.CreatedAt.Time),
		UpdatedAt:           timestamppb.New(b.UpdatedAt.Time),
	}
}

func yearPtr(y int) *int32 {
	if y > 0 {
		v := int32(y)
		return &v
	}
	return nil
}
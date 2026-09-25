package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	tasksv1 "github.com/nnc/family-manager/sdk/go/tasks/v1"
	"github.com/nnc/family-manager/services/tasks/db"
	"github.com/nnc/family-manager/services/tasks/internal/family"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
)

const kindBirthday = "birthday"

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

	now, loc := h.now(), h.familyLoc(ctx, c)
	birthdays := make([]*tasksv1.Birthday, 0, len(rows))
	for _, b := range rows {
		birthdays = append(birthdays, toProtoBirthday(b, now, loc))
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
	name, err := birthdayName(msg.GetName())
	if err != nil {
		return nil, err
	}
	day, month, year := int(msg.GetDay()), int(msg.GetMonth()), int(msg.GetYear())
	remindDays := int(msg.GetRemindDaysBefore())

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc
	if err := schedule.ValidateBirthday(day, month, year, remindDays, now.In(loc)); err != nil {
		return nil, invalid("%v", err)
	}

	members := h.fetchMembers(ctx, c, req.Header())

	var birthday db.Birthday
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		var innerErr error
		birthday, innerErr = q.CreateBirthday(ctx, db.CreateBirthdayParams{
			FamilyID:         c.familyID,
			Name:             name,
			Day:              int32(day),
			Month:            int32(month),
			Year:             yearPtr(year),
			RemindDaysBefore: int32(remindDays),
			CreatedByUserID:  c.userID,
		})
		if innerErr != nil {
			return innerErr
		}
		if innerErr = refreshKnownMembers(ctx, q, c, members); innerErr != nil {
			return innerErr
		}
		return scheduleBirthdayReminders(ctx, q, c, birthday, now, loc)
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create birthday")
	}

	return connect.NewResponse(&tasksv1.CreateBirthdayResponse{Birthday: toProtoBirthday(birthday, now, loc)}), nil
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
	var newName string
	if msg.Name != nil {
		if newName, err = birthdayName(msg.GetName()); err != nil {
			return nil, err
		}
	}

	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	now, loc := h.now(), hh.loc
	members := h.fetchMembers(ctx, c, req.Header())

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
			name = newName
		}
		day, month := int(existing.Day), int(existing.Month)
		if msg.Day != nil {
			day = int(msg.GetDay())
		}
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

		if err := schedule.ValidateBirthday(day, month, year, remindDays, now.In(loc)); err != nil {
			return invalid("%v", err)
		}

		birthday, innerErr = q.UpdateBirthday(ctx, db.UpdateBirthdayParams{
			ID:               birthdayID,
			FamilyID:         c.familyID,
			Name:             name,
			Day:              int32(day),
			Month:            int32(month),
			Year:             yearPtr(year),
			RemindDaysBefore: int32(remindDays),
		})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "update birthday")
		}
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: kindBirthday, ItemID: birthdayID,
		}); innerErr != nil {
			return h.internal(ctx, innerErr, "delete birthday reminders")
		}
		if innerErr = refreshKnownMembers(ctx, q, c, members); innerErr != nil {
			return h.internal(ctx, innerErr, "refresh known members")
		}
		if innerErr = scheduleBirthdayReminders(ctx, q, c, birthday, now, loc); innerErr != nil {
			return h.internal(ctx, innerErr, "schedule birthday reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.UpdateBirthdayResponse{Birthday: toProtoBirthday(birthday, now, loc)}), nil
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
		rows, innerErr := q.DeleteBirthday(ctx, db.DeleteBirthdayParams{ID: birthdayID, FamilyID: c.familyID})
		if innerErr != nil {
			return h.internal(ctx, innerErr, "delete birthday")
		}
		if rows == 0 {
			return notFound("birthday")
		}
		if innerErr = q.DeletePendingRemindersForItem(ctx, db.DeletePendingRemindersForItemParams{
			FamilyID: c.familyID, Kind: kindBirthday, ItemID: birthdayID,
		}); innerErr != nil {
			return h.internal(ctx, innerErr, "delete birthday reminders")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&tasksv1.DeleteBirthdayResponse{}), nil
}

func (h *Handler) familyLoc(ctx context.Context, c caller) *time.Location {
	if hh, err := h.household(ctx, c); err == nil {
		return hh.loc
	}
	return time.UTC
}

func (h *Handler) fetchMembers(ctx context.Context, c caller, header http.Header) []family.Member {
	if h.familyPub == nil {
		return nil
	}
	bearer := fmauth.BearerToken(header.Get("Authorization"))
	if bearer == "" {
		return nil
	}
	members, err := h.familyPub.ListMembers(ctx, bearer, c.family)
	if err != nil {
		h.log.WarnContext(ctx, "refresh known members", slog.String("error", err.Error()))
		return nil
	}
	return members
}

func refreshKnownMembers(ctx context.Context, q db.Querier, c caller, members []family.Member) error {
	keep := make([]pgtype.UUID, 0, len(members))
	for _, m := range members {
		uid, err := pgconv.UUID(m.UserID)
		if err != nil {
			continue
		}
		if err := q.UpsertKnownMember(ctx, db.UpsertKnownMemberParams{
			FamilyID: c.familyID, UserID: uid, DisplayName: m.DisplayName, Email: m.Email,
		}); err != nil {
			return err
		}
		keep = append(keep, uid)
	}
	if len(keep) == 0 {
		return nil
	}
	return q.DeleteKnownMembersExcept(ctx, db.DeleteKnownMembersExceptParams{FamilyID: c.familyID, Keep: keep})
}

func scheduleBirthdayReminders(
	ctx context.Context, q db.Querier, c caller, b db.Birthday, now time.Time, loc *time.Location,
) error {
	occ := schedule.NextOccurrence(int(b.Day), int(b.Month), birthYear(b), now.In(loc))
	remindAts := schedule.BirthdayRemindAt(occ, int(b.RemindDaysBefore), loc, now)
	if len(remindAts) == 0 {
		return nil
	}
	members, err := q.ListKnownMembers(ctx, c.familyID)
	if err != nil {
		return err
	}
	for _, remindAt := range remindAts {
		key := birthdayOccurrence(occ, remindAt, loc)
		for _, m := range members {
			if err := q.UpsertReminder(ctx, db.UpsertReminderParams{
				FamilyID:   c.familyID,
				UserID:     m.UserID,
				Kind:       kindBirthday,
				ItemID:     b.ID,
				Occurrence: key,
				RemindAt:   pgTimestamptz(remindAt),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func birthdayOccurrence(occ schedule.Occurrence, remindAt time.Time, loc *time.Location) string {
	on := occ.On.Format(schedule.DateLayout)
	before := schedule.DaysBetween(schedule.Civil(remindAt, loc), occ.On)
	if before <= 0 {
		return on
	}
	return fmt.Sprintf("%s-%dd", on, before)
}

func birthdayName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", invalid("name is required")
	}
	if err := checkText("name", name, maxTitleRunes); err != nil {
		return "", err
	}
	return name, nil
}

func birthYear(b db.Birthday) int {
	if b.Year == nil {
		return 0
	}
	return int(*b.Year)
}

func toProtoBirthday(b db.Birthday, now time.Time, loc *time.Location) *tasksv1.Birthday {
	occ := schedule.NextOccurrence(int(b.Day), int(b.Month), birthYear(b), now.In(loc))
	age := int32(0)
	if occ.HasAge {
		age = int32(occ.TurningAge)
	}
	return &tasksv1.Birthday{
		Id:               pgconv.UUIDString(b.ID),
		FamilyId:         pgconv.UUIDString(b.FamilyID),
		Name:             b.Name,
		Day:              b.Day,
		Month:            b.Month,
		Year:             int32(birthYear(b)),
		RemindDaysBefore: b.RemindDaysBefore,
		NextOn:           occ.On.Format(schedule.DateLayout),
		DaysUntil:        int32(occ.DaysUntil),
		TurningAge:       age,
		CreatedAt:        timestamppb.New(b.CreatedAt.Time),
		UpdatedAt:        timestamppb.New(b.UpdatedAt.Time),
	}
}

func yearPtr(y int) *int32 {
	if y <= 0 {
		return nil
	}
	v := int32(y)
	return &v
}

package pgconv

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UUID(s string) (pgtype.UUID, error) {
	var out pgtype.UUID
	if s == "" {
		return out, nil
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return out, err
	}
	copy(out.Bytes[:], parsed[:])
	out.Valid = true
	return out, nil
}

func MustUUID(s string) pgtype.UUID {
	out, _ := UUID(s)
	return out
}

func UUIDString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func Timestamp(t pgtype.Timestamptz) *timestamppb.Timestamp {
	if !t.Valid {
		return nil
	}
	return timestamppb.New(t.Time)
}

func TimestampFrom(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func Date(s string) (pgtype.Date, error) {
	var out pgtype.Date
	if s == "" {
		return out, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return out, err
	}
	out.Time, out.Valid = t, true
	return out, nil
}

func DateFromToday() pgtype.Date {
	now := time.Now().UTC()
	return pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

func DateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(time.DateOnly)
}

func Text(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func TextString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

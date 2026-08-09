// Package pgconv converts between pgx's Postgres types and the plain Go/protobuf types that
// cross a service boundary. Every service does this at the edge of its handlers; doing it
// twice by hand is how a UUID ends up rendered as "{...}" in one response and "" in another.
package pgconv

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UUID parses a string into a pgtype.UUID. An empty or malformed string yields an invalid
// (NULL) value, which every query treats as "no match" rather than erroring at the driver.
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

// MustUUID is UUID for values already known to be valid (a row just read from the database).
func MustUUID(s string) pgtype.UUID {
	out, _ := UUID(s)
	return out
}

// UUIDString renders a pgtype.UUID; an invalid value becomes "".
func UUIDString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// Timestamp converts to protobuf; an invalid value becomes nil, which protobuf renders as an
// absent field rather than 1970.
func Timestamp(t pgtype.Timestamptz) *timestamppb.Timestamp {
	if !t.Valid {
		return nil
	}
	return timestamppb.New(t.Time)
}

// TimestampFrom wraps a Go time for a query parameter.
func TimestampFrom(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// Date parses YYYY-MM-DD into a pgtype.Date. Calendar dates stay calendar dates: no timezone
// is applied, because "the 31st" must not become "the 1st" for a user in another offset.
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

// DateFromToday returns today's UTC calendar date as a pgtype.Date.
func DateFromToday() pgtype.Date {
	now := time.Now().UTC()
	return pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

// DateString renders a pgtype.Date as YYYY-MM-DD; an invalid value becomes "".
func DateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(time.DateOnly)
}

// Text wraps a string for a nullable column, treating "" as NULL.
func Text(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// TextString unwraps a nullable column, NULL becoming "".
func TextString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

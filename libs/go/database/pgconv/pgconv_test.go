package pgconv

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUUIDRoundTrip(t *testing.T) {
	const in = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	u, err := UUID(in)
	if err != nil {
		t.Fatalf("UUID: %v", err)
	}
	if !u.Valid {
		t.Fatal("expected a valid uuid")
	}
	if got := UUIDString(u); got != in {
		t.Errorf("UUIDString = %q, want %q", got, in)
	}
}

func TestUUIDEmptyIsNullNotError(t *testing.T) {
	u, err := UUID("")
	if err != nil {
		t.Fatalf("UUID(\"\") = %v, want nil error", err)
	}
	if u.Valid {
		t.Error("empty string should produce an invalid (NULL) uuid")
	}
	if got := UUIDString(u); got != "" {
		t.Errorf("UUIDString = %q, want empty", got)
	}
}

func TestUUIDRejectsGarbage(t *testing.T) {
	if _, err := UUID("not-a-uuid"); err == nil {
		t.Fatal("expected an error for a malformed uuid")
	}
}

func TestTimestampNilWhenInvalid(t *testing.T) {
	if got := Timestamp(pgtype.Timestamptz{}); got != nil {
		t.Errorf("Timestamp(invalid) = %v, want nil", got)
	}
	now := time.Date(2026, 3, 12, 10, 30, 0, 0, time.UTC)
	got := Timestamp(TimestampFrom(now))
	if got == nil || !got.AsTime().Equal(now) {
		t.Errorf("Timestamp round trip = %v, want %v", got, now)
	}
}

func TestDateKeepsTheCalendarDay(t *testing.T) {
	d, err := Date("2026-03-31")
	if err != nil {
		t.Fatalf("Date: %v", err)
	}
	if got := DateString(d); got != "2026-03-31" {
		t.Errorf("DateString = %q, want 2026-03-31", got)
	}

	empty, err := Date("")
	if err != nil {
		t.Fatalf("Date(\"\") = %v, want nil error", err)
	}
	if empty.Valid || DateString(empty) != "" {
		t.Error("empty date should be NULL and render as empty")
	}

	if _, err := Date("31/03/2026"); err == nil {
		t.Fatal("expected an error for a non-ISO date")
	}
}

func TestTextTreatsEmptyAsNull(t *testing.T) {
	if Text("").Valid {
		t.Error("empty string should be NULL")
	}
	if got := TextString(Text("note")); got != "note" {
		t.Errorf("TextString = %q, want note", got)
	}
	if got := TextString(pgtype.Text{}); got != "" {
		t.Errorf("TextString(NULL) = %q, want empty", got)
	}
}

package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/services/tasks/db"
	dbfs "github.com/nnc/family-manager/services/tasks/internal/db"
)

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TASKS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TASKS_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	p, err := database.Connect(ctx, database.Config{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	if _, err := database.Migrate(ctx, p, dbfs.Migrations, dbfs.MigrationsDir); err != nil {
		t.Fatal(err)
	}
	return p
}

func uuid(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestInTxCommitsAndRollsBack(t *testing.T) {
	p := pool(t)
	st := New(p)
	ctx := context.Background()
	fam := uuid(t, "aaaaaaaa-0000-4000-8000-000000000001")
	user := uuid(t, "aaaaaaaa-0000-4000-8000-000000000002")
	_, _ = p.Exec(ctx, "DELETE FROM known_members WHERE family_id = $1", fam)

	boom := errors.New("boom")
	err := st.InTx(ctx, func(q db.Querier) error {
		if err := q.TouchKnownMember(ctx, db.TouchKnownMemberParams{FamilyID: fam, UserID: user, Email: "a@x"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	rows, _ := st.Queries().ListKnownMembers(ctx, fam)
	if len(rows) != 0 {
		t.Fatalf("a failed transaction must roll back, found %d rows", len(rows))
	}

	if err := st.InTx(ctx, func(q db.Querier) error {
		return q.TouchKnownMember(ctx, db.TouchKnownMemberParams{FamilyID: fam, UserID: user, Email: "a@x"})
	}); err != nil {
		t.Fatal(err)
	}
	rows, _ = st.Queries().ListKnownMembers(ctx, fam)
	if len(rows) != 1 {
		t.Fatalf("committed rows = %d", len(rows))
	}
	_, _ = p.Exec(ctx, "DELETE FROM known_members WHERE family_id = $1", fam)
}

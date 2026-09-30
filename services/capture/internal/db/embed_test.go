package dbfs

import (
	"regexp"
	"strings"
	"testing"

	"github.com/nnc/family-manager/libs/go/database"
)

func TestMigrationsLoad(t *testing.T) {
	ms, err := database.LoadMigrations(Migrations, MigrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 || ms[0].Version != 1 {
		t.Fatalf("expected migration 1 first, got %+v", ms)
	}
}

var createTable = regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS (\w+) \((.*?)\n\);`)

func migrationBodies(t *testing.T) map[string]string {
	t.Helper()
	entries, err := Migrations.ReadDir(MigrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		body, err := Migrations.ReadFile(MigrationsDir + "/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(body)
	}
	return out
}

func TestEveryTableIsScopedByMember(t *testing.T) {
	found := map[string]string{}
	for _, body := range migrationBodies(t) {
		for _, m := range createTable.FindAllStringSubmatch(body, -1) {
			found[m[1]] = m[2]
		}
	}
	for _, want := range []string{"captured_notifications", "suggestions", "capture_settings", "prefill_rules", "deleted_notification_keys"} {
		if _, ok := found[want]; !ok {
			t.Errorf("table %s missing", want)
		}
	}
	for name, cols := range found {
		if !strings.Contains(cols, "family_id ") {
			t.Errorf("table %s has no family_id column", name)
		}
		if !strings.Contains(cols, "user_id ") {
			t.Errorf("table %s has no user_id column", name)
		}
	}
}

func TestMigrationsAreAdditive(t *testing.T) {
	drop := regexp.MustCompile(`(?i)\bDROP\b`)
	for name, body := range migrationBodies(t) {
		if !strings.HasSuffix(name, ".up.sql") {
			t.Errorf("unexpected migration file %s", name)
		}
		if drop.MatchString(body) {
			t.Errorf("%s contains DROP", name)
		}
	}
}

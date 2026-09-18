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

func TestEveryTableIsScopedByFamily(t *testing.T) {
	body, err := Migrations.ReadFile(MigrationsDir + "/000001_init.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	tables := createTable.FindAllStringSubmatch(string(body), -1)
	if len(tables) < 9 {
		t.Fatalf("expected the nine tables, found %d", len(tables))
	}
	for _, m := range tables {
		if !strings.Contains(m[2], "family_id ") {
			t.Errorf("table %s has no family_id column", m[1])
		}
	}
}

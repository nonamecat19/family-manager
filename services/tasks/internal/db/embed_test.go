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
	var tables [][]string
	entries, err := Migrations.ReadDir(MigrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		body, err := Migrations.ReadFile(MigrationsDir + "/" + e.Name())
		if err != nil {
			t.Fatal(err)
		}
		tables = append(tables, createTable.FindAllStringSubmatch(string(body), -1)...)
	}
	if len(tables) < 10 {
		t.Fatalf("expected at least ten tables, found %d", len(tables))
	}
	exempt := map[string]bool{
		"processed_events": true, // global event idempotency table
	}
	for _, m := range tables {
		if exempt[m[1]] {
			continue
		}
		if !strings.Contains(m[2], "family_id ") {
			t.Errorf("table %s has no family_id column", m[1])
		}
	}
}

package database

import (
	"testing"
	"testing/fstest"
)

func TestLoadMigrationsOrdersByVersionAndSkipsDown(t *testing.T) {
	fsys := fstest.MapFS{
		"m/000002_second.up.sql":  {Data: []byte("SELECT 2;")},
		"m/000001_first.up.sql":   {Data: []byte("SELECT 1;")},
		"m/000001_first.down.sql": {Data: []byte("DROP 1;")},
		"m/000010_tenth.up.sql":   {Data: []byte("SELECT 10;")},
		"m/notes.md":              {Data: []byte("ignored")},
	}

	got, err := LoadMigrations(fsys, "m")
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	want := []int64{1, 2, 10}
	if len(got) != len(want) {
		t.Fatalf("got %d migrations, want %d: %+v", len(got), len(want), got)
	}
	for i, v := range want {
		if got[i].Version != v {
			t.Errorf("migration %d version = %d, want %d", i, got[i].Version, v)
		}
	}
	if got[0].SQL != "SELECT 1;" {
		t.Errorf("first migration SQL = %q", got[0].SQL)
	}
}

func TestLoadMigrationsRejectsDuplicateVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"m/000001_a.up.sql": {Data: []byte("SELECT 1;")},
		"m/000001_b.up.sql": {Data: []byte("SELECT 2;")},
	}
	if _, err := LoadMigrations(fsys, "m"); err == nil {
		t.Fatal("expected an error on duplicate versions")
	}
}

func TestLoadMigrationsRejectsUnnumbered(t *testing.T) {
	fsys := fstest.MapFS{"m/init.up.sql": {Data: []byte("SELECT 1;")}}
	if _, err := LoadMigrations(fsys, "m"); err == nil {
		t.Fatal("expected an error on an unnumbered migration")
	}
}

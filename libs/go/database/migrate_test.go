package database

import (
	"strings"
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

func migrationsFor(t *testing.T, files map[string]string) []Migration {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["m/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	got, err := LoadMigrations(fsys, "m")
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	return got
}

func TestChecksumIgnoresLineEndings(t *testing.T) {
	if Checksum([]byte("SELECT 1;\r\nSELECT 2;\r\n")) != Checksum([]byte("SELECT 1;\nSELECT 2;\n")) {
		t.Fatal("a CRLF checkout must not read as an edited migration")
	}
	if Checksum([]byte("SELECT 1;")) == Checksum([]byte("SELECT 2;")) {
		t.Fatal("different SQL must hash differently")
	}
}

func TestCheckAppliedAcceptsMatchingHistory(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_a.up.sql": "SELECT 1;", "000002_b.up.sql": "SELECT 2;"})

	unrecorded, _, err := CheckApplied(map[int64]string{1: ms[0].Checksum}, ms)
	if err != nil {
		t.Fatalf("CheckApplied: %v", err)
	}
	if len(unrecorded) != 0 {
		t.Fatalf("unrecorded = %+v, want none", unrecorded)
	}
}

func TestCheckAppliedRecordsChecksumsForRowsThatPredateThem(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_a.up.sql": "SELECT 1;", "000002_b.up.sql": "SELECT 2;"})

	unrecorded, _, err := CheckApplied(map[int64]string{1: "", 2: ""}, ms)
	if err != nil {
		t.Fatalf("CheckApplied: %v", err)
	}
	if len(unrecorded) != 2 || unrecorded[0].Version != 1 || unrecorded[1].Version != 2 {
		t.Fatalf("unrecorded = %+v, want versions 1 and 2", unrecorded)
	}
}

func TestCheckAppliedRefusesAnEditedMigration(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_init.up.sql": "CREATE TABLE accounts_v2 ();"})

	_, _, err := CheckApplied(map[int64]string{1: Checksum([]byte("CREATE TABLE accounts ();"))}, ms)
	if err == nil || !strings.Contains(err.Error(), "000001_init.up.sql changed after it was applied") {
		t.Fatalf("err = %v, want an edited-migration refusal", err)
	}
}

func TestCheckAppliedRefusesAMigrationDeletedFromTheMiddle(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_init.up.sql": "SELECT 1;", "000003_c.up.sql": "SELECT 3;"})

	_, _, err := CheckApplied(map[int64]string{1: ms[0].Checksum, 2: "anything", 3: ms[1].Checksum}, ms)
	if err == nil || !strings.Contains(err.Error(), "migration 2 is recorded as applied but its .up.sql file is gone") {
		t.Fatalf("err = %v, want a deleted-migration refusal", err)
	}
}

func TestCheckAppliedLetsARolledBackBuildBoot(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_init.up.sql": "SELECT 1;"})

	unrecorded, ahead, err := CheckApplied(map[int64]string{1: ms[0].Checksum, 2: "from the newer build", 3: ""}, ms)
	if err != nil {
		t.Fatalf("a rollback to an older build must boot: %v", err)
	}
	if len(unrecorded) != 0 || len(ahead) != 2 || ahead[0] != 2 || ahead[1] != 3 {
		t.Fatalf("unrecorded=%+v ahead=%v, want none and [2 3]", unrecorded, ahead)
	}
}

func TestCheckAppliedCatchesASquashOnceChecksumsAreRecorded(t *testing.T) {
	before := migrationsFor(t, map[string]string{"000001_init.up.sql": "CREATE TABLE accounts ();", "000002_budgets.up.sql": "CREATE TABLE budgets ();"})
	squashed := migrationsFor(t, map[string]string{"000001_init.up.sql": "CREATE TABLE accounts (); CREATE TABLE budgets (); CREATE TABLE finance_settings ();"})

	_, _, err := CheckApplied(map[int64]string{1: before[0].Checksum, 2: before[1].Checksum}, squashed)
	if err == nil || !strings.Contains(err.Error(), "000001_init.up.sql changed after it was applied") {
		t.Fatalf("err = %v, want the rewritten 000001 refused", err)
	}
}

func TestCheckAppliedRefusesABuildThatKnowsNoneOfTheHistory(t *testing.T) {
	baseline := migrationsFor(t, map[string]string{"000000_baseline.up.sql": "CREATE TABLE accounts ();"})

	_, _, err := CheckApplied(map[int64]string{1: "x", 2: "y"}, baseline)
	if err == nil || !strings.Contains(err.Error(), "every recorded migration (lowest 1) is newer than this build's highest (0)") {
		t.Fatalf("err = %v, want a refusal to run a baseline over an unknown history", err)
	}
}

func TestCheckAppliedOnAFreshDatabaseRunsEverything(t *testing.T) {
	ms := migrationsFor(t, map[string]string{"000001_init.up.sql": "SELECT 1;"})

	unrecorded, ahead, err := CheckApplied(map[int64]string{}, ms)
	if err != nil || len(unrecorded) != 0 || len(ahead) != 0 {
		t.Fatalf("unrecorded=%v ahead=%v err=%v", unrecorded, ahead, err)
	}
}

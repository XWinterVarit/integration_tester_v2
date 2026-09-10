package v1

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDBClient(t *testing.T) {
	// Use in-memory sqlite
	db := Connect("sqlite3", ":memory:")

	// Setup Table
	fields := []Field{
		{"id", "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"name", "TEXT"},
		{"age", "INTEGER"},
	}
	db.SetupTable("users", true, fields, nil)

	// Test ReplaceData (Insert)
	// Since ID is auto-increment, we might skip it or provide it.
	// ReplaceData implementation depends on how many args vs fields.
	// In db.go: ReplaceData takes values.
	// Let's assume we provide all values for ReplaceData if it constructs "INSERT INTO ... VALUES (?, ?, ...)"
	// db.go implementation constructs insert based on fields stored in client?
	// Wait, DBClient struct doesn't seem to store schema permanently per table unless it caches it?
	// Let's look at db.go logic later if it fails.
	// Actually, ReplaceData implementation:
	// It uses `c.fields` which is set during SetupTable.
	// So we must call SetupTable first (we did).

	db.ReplaceData("users", []interface{}{1, "Alice", 30})

	// Test Fetch
	result := db.Fetch("SELECT name, age FROM users WHERE id = ?", 1)
	if result.Count() != 1 {
		t.Errorf("Expected 1 row, got %d", result.Count())
	}

	row := result.GetRow(0)
	val := row.Get("name")
	if val != "Alice" {
		t.Errorf("Expected Alice, got %v", val)
	}

	row.Expect("age", int64(30)) // Sqlite returns int64 usually

	// Test Update
	db.Update("users", map[string]interface{}{"age": 31}, "id = ?", 1)

	result = db.Fetch("SELECT age FROM users WHERE id = ?", 1)
	result.GetRow(0).Expect("age", int64(31))

	// Test CleanTable
	db.CleanTable("users")
	result = db.Fetch("SELECT * FROM users")
	if result.Count() != 0 {
		t.Errorf("Expected 0 rows after clean, got %d", result.Count())
	}

	// Test DropTable
	db.DropTable("users")
	// Verify drop? (Select should fail)
	// db.Fetch panics on error usually? Or returns empty?
	// implementation: rows, err := c.db.Query... if err != nil -> Fail(...)
	// So it should panic.

	defer func() {
		if r := recover(); r != nil {
			// Expected panic
		} else {
			t.Error("Expected panic when querying dropped table")
		}
	}()
	db.Fetch("SELECT * FROM users")
}

func TestRowResultExpectCond(t *testing.T) {
	// Use in-memory sqlite
	db := Connect("sqlite3", ":memory:")

	fields := []Field{
		{"id", "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"name", "TEXT"},
		{"age", "INTEGER"},
	}
	db.SetupTable("users", true, fields, nil)

	// Insert rows including NULLs
	db.ReplaceData("users", []interface{}{1, "Alice", 30})
	db.ReplaceData("users", []interface{}{2, nil, nil})

	result := db.Fetch("SELECT name, age FROM users WHERE id = ?", 1)
	row := result.GetRow(0)

	// Success cases
	row.ExpectCond("age", ConditionGreaterThan, 20)
	row.ExpectCond("age", ConditionLessThanOrEqual, 30)
	row.ExpectCond("name", ConditionContains, "lic")
	row.ExpectCond("name", ConditionStartsWith, "Al")
	row.ExpectCond("name", ConditionEndsWith, "ce")

	// NULL handling
	nullRow := db.Fetch("SELECT name, age FROM users WHERE id = ?", 2).GetRow(0)
	nullRow.ExpectCond("name", ConditionEqual, nil)
	nullRow.ExpectCond("age", ConditionEqual, nil)

	// Failure cases (should panic)
	assertPanic := func(name string, f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s expected to panic", name)
			} else {
				if _, ok := r.(TestError); !ok {
					t.Errorf("%s panicked with unexpected type: %T", name, r)
				}
			}
		}()
		f()
	}

	assertPanic("condition mismatch", func() { row.ExpectCond("age", ConditionLessThan, 10) })
	assertPanic("missing field", func() { row.ExpectCond("missing", ConditionEqual, 1) })
}

func TestInsertOne(t *testing.T) {
	// Use in-memory sqlite
	db := Connect("sqlite3", ":memory:")

	fields := []Field{
		{"id", "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"name", "TEXT"},
		{"age", "INTEGER"},
	}
	db.SetupTable("users", true, fields, nil)

	// happy path
	db.InsertOne("users", []InsertField{
		{"name", "Alice"},
		{"age", 30},
	})
	result := db.Fetch("SELECT name, age FROM users")
	result.ExpectCount(1)
	row := result.GetRow(0)
	row.Expect("name", "Alice")
	row.Expect("age", int64(30))

	// invalid arg length (should panic)
	assertPanic := func(name string, f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s expected to panic", name)
			} else {
				if _, ok := r.(TestError); !ok {
					t.Errorf("%s panicked with unexpected type: %T", name, r)
				}
			}
		}()
		f()
	}

	assertPanic("no fields", func() { db.InsertOne("users", []InsertField{}) })
	assertPanic("bad field name", func() { db.InsertOne("users", []InsertField{{Key: "", Value: "Bob"}}) })
}

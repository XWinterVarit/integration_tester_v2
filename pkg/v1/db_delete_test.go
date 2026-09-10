package v1

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDBDeleteHelpers(t *testing.T) {
	// in-memory sqlite
	db := Connect("sqlite3", ":memory:")
	defer db.DB.Close()

	db.SetupTable("items", true, []Field{
		{Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{Name: "name", Type: "TEXT"},
	}, nil)

	// seed
	for i := 0; i < 3; i++ {
		db.ReplaceData("items", []interface{}{nil, "item"})
	}

	// delete one
	db.DeleteOne("items", "id = ?", 1)
	res := db.Fetch("SELECT COUNT(*) as cnt FROM items WHERE id = ?", 1)
	res.GetRow(0).Expect("cnt", int64(0))

	// delete with limit 1 (should remove one matching row)
	db.DeleteWithLimit("items", "name = ?", 1, "item")
	res2 := db.Fetch("SELECT COUNT(*) as cnt FROM items", sql.Named("unused", ""))
	// remaining should be 1 (started 3, removed 1 by id, removed 1 by limit)
	res2.GetRow(0).Expect("cnt", int64(1))

	// delete remaining all (limit<=0)
	db.DeleteWithLimit("items", "name = ?", 0, "item")
	res3 := db.Fetch("SELECT COUNT(*) as cnt FROM items", sql.Named("unused", ""))
	res3.GetRow(0).Expect("cnt", int64(0))
}

package fspot

import (
	"code.google.com/p/gosqlite/sqlite"
	"testing"
)

func fspotTestDb() (db Db, conn *sqlite.Conn, err error) {
	conn, err = sqlite.Open("test_db")
	if err != nil {
		return
	}
	db = Db{conn, make(map[int]string)}
	return
}

func TestCalculateTagPaths(t *testing.T) {
	db, conn, err := fspotTestDb()
	if err != nil {
		t.Fatal(err)
	}
  defer conn.Close()
	err = db.CalculateTagPaths()
	if err != nil {
		t.Fatal(err)
	}
}

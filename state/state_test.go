package state

import (
	"errors"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	db, err := open("test.db")
	defer db.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}
	os.Remove("test.db")
}

func TestLog(t *testing.T) {
	db, err := open("test.db")
	defer db.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	record := Log{1, "Foo", errors.New("Err")}
	err = db.Log(record)
	if err != nil {
		t.Fatal(err)
	}
	os.Remove("test.db")

}

func TestMaxIdOnEmpty(t *testing.T) {
	db, err := open("test.db")
	defer db.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	id, err := db.MaxId()
	if err != nil {
		t.Fatal(err)
	}
	if id != 0 {
		t.Fatal("Expected 0 got ", id)
	}
	os.Remove("test.db")

}
func TestMaxId(t *testing.T) {
	db, err := open("test.db")
	defer db.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = db.Init()
	if err != nil {
		t.Fatal(err)
	}

	record := Log{1, "Foo", errors.New("Err")}
	err = db.Log(record)
	if err != nil {
		t.Fatal(err)
	}
	id, err := db.MaxId()
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatal("Expected 1 got ", id)
	}
	os.Remove("test.db")

}

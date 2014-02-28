package state

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Log struct {
	Id        int
	Permanode string
	Error     error
}

type LogChan chan Log

type Db struct {
	Conn *sqlite.Conn
	Chan LogChan
}

func Open() (db Db, err error) {
	dbpath := fmt.Sprintf("%s/.config/fspot2camlistore", os.Getenv("HOME"))
	err = os.MkdirAll(dbpath, 0700)
	if err != nil {
		return
	}
	dbfile := dbpath + "/state.db"
	return open(dbfile)
}

func open(dbfile string) (db Db, err error) {
	Conn, err := sqlite.Open(dbfile)
	return Db{Conn, make(LogChan)}, err
}

func (db *Db) Close() (err error) {
	err = db.Conn.Close()
	return
}

func (db *Db) Init() (err error) {
	err = db.Conn.Exec("CREATE TABLE IF NOT EXISTS fspotcamli (fspot_id INT, perma TEXT, error TEXT)")
	if err != nil {
		return
	}
	return
}

func (db *Db) Log(record Log) (err error) {
	errString := ""
	if record.Error != nil {
		errString = fmt.Sprintf("%v", record.Error)
	}
	err = db.Conn.Exec(fmt.Sprintf("INSERT INTO fspotcamli (fspot_id, perma, error) VALUES (%d, '%s', '%s')", record.Id, record.Permanode, errString))
	return
}

func (db *Db) MaxId() (id int64, err error) {
	stmt, err := db.Conn.Prepare("SELECT MAX(fspot_id) FROM fspotcamli")
	if err != nil {
		return
	}
	if stmt.Next() {
		var s string
		err = stmt.Scan(&s)
		if err != nil {
			return
		}
		if s == "" {
			id = 0
		} else {
			id, err = strconv.ParseInt(s, 0, 64)
		}
	} else {
		err = errors.New("No rows returned on max query")
	}
	return
}

package fspot

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Photo struct {
	Id          int
	Desc        string
	Tags        []string
	AbsTagPaths []string
	Path        string
	Filename    string
	Taken       time.Time
}

type Db struct {
	Conn           *sqlite.Conn
	AbsTagPathMap map[int]string
}

func (db *Db) CalculateTagPaths() (err error) {
	stmt, err := db.Conn.Prepare("SELECT id, name, category_id FROM tags")
	if err != nil {
		return
	}
	for stmt.Next() {
		var id, category_id int
		var name string
		err = stmt.Scan(&id, &name, &category_id)
		if err != nil {
			return
		}
		tagPath := name
		for category_id != 0 {
			var innerStmt *sqlite.Stmt
			innerStmt, err = db.Conn.Prepare(fmt.Sprintf("SELECT name, category_id FROM tags WHERE id=%d", category_id))
			if err != nil {
				return
			}
			if innerStmt.Next() {
				err = innerStmt.Scan(&name, &category_id)
				if err != nil {
					return
				}
				tagPath = name + "/" + tagPath
			}
		}
		db.AbsTagPathMap[id] = tagPath
	}
	return
}

func (db *Db) PhotoLoop(firstPhotoId int, ch chan<- Photo) (err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT id, description, filename, time FROM photos WHERE id >= %d ORDER BY id;", firstPhotoId))
	if err != nil {
		return
	}

	for stmt.Next() {
		var timeStamp int64
		var desc string
    var filename string
		var fspot_id int

		err = stmt.Scan(&fspot_id, &desc, &filename, &timeStamp)
		if err != nil {
			return
		}
		photoDate := time.Unix(timeStamp, 0)

		var tag map[int]string
		tag, err = db.tags(fspot_id)
		if err != nil {
			return
		}
		var path string
		path, err = db.imagePath(fspot_id)
		if err != nil {
			return
		}

    tags := []string{}
    absTagPaths := []string{}
    for k, v := range tag {
      tags = append(tags, v)
      absTagPaths = append(absTagPaths, db.AbsTagPathMap[k])
    }
    sort.Strings(tags)
    sort.Strings(absTagPaths)
    photo := Photo{Id:fspot_id,
                   Desc:desc,
                   Tags: tags,
                   AbsTagPaths: absTagPaths,
                   Filename: filename,
                   Path: path,
                   Taken: photoDate}
    ch <- photo
	}
	return
}

func (db *Db) tags(id int) (tags map[int]string, err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT id, name FROM photo_tags, tags WHERE photo_tags.photo_id=%d AND tags.id=photo_tags.tag_id", id))
	if err != nil {
		return
	}
	tags = make(map[int]string)
	for stmt.Next() {
		var name string
		var id int
		err = stmt.Scan(&id, &name)
		if err != nil {
			return
		}
		tags[id] = name
	}
	return
}

func (db *Db) imagePath(id int) (u string, err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT default_version_id, base_uri, filename FROM photos WHERE id=%d", id))
	if err != nil {
		return
	}
	if !stmt.Next() {
		err = errors.New("Not found")
	}
	var default_version_id int
	var base_uri, filename string
	err = stmt.Scan(&default_version_id, &base_uri, &filename)
	if err != nil {
		return
	}
	if default_version_id != 1 {
		stmt, err = db.Conn.Prepare(fmt.Sprintf("SELECT base_uri, filename FROM photo_versions WHERE version_id=%d  AND photo_id=%d", default_version_id, id))
		if err != nil {
			return
		}
		if !stmt.Next() {
			err = errors.New("Not found")
		}
		err = stmt.Scan(&base_uri, &filename)
		if err != nil {
			return
		}
	}
	u = base_uri + "/" + filename
	u = "/" + strings.TrimLeft(u, "file:/")
	u, err = url.QueryUnescape(u)
	return
}

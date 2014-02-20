package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	camput       = flag.String("camput", "bin/camput", "path to camput binary")
	photoDb      = flag.String("db", "test_db", "path to F-Spot sqlitedb")
	clearCache   = flag.Bool("clear", true, "if set to false the camlistore cache is not cleared")
	firstPhotoId = flag.Int("first", 0, "first photo id to process")
)

func main() {
	flag.Parse()
	conn, err := sqlite.Open(*photoDb)
	if err != nil {
		log.Fatal("Unable to open f-spot source database")
	}
	defer conn.Close()
	db := fspotDb{conn, make(map[int]string)}
	err = db.calculateTagPaths()
	if err != nil {
		log.Fatal("Unable to populate tags paths")
	}
	counter, err := db.photoLoop()
	if err != nil {
		log.Println("Error in photoLoop", err)
		log.Println("You can resume later with the -first flag set higher then ", counter)
	} else {
		log.Println("Finished! Processed ", counter, " files.")
	}
}

type fspotDb struct {
	Conn           *sqlite.Conn
	absoluteTagMap map[int]string
}

func (db *fspotDb) calculateTagPaths() (err error) {
	stmt, err := db.Conn.Prepare("SELECT id, name, category_id FROM tags")
	if err != nil {
		return
	}
	for stmt.Next() {
		var id, category_id int
		var name string
		stmt.Scan(&id, &name, &category_id)
		tagPath := name
		for category_id != 0 {
			var innerStmt *sqlite.Stmt
			innerStmt, err = db.Conn.Prepare(fmt.Sprintf("SELECT name, category_id FROM tags WHERE id=%d", category_id))
			if err != nil {
				return
			}
			if innerStmt.Next() {
				innerStmt.Scan(&name, &category_id)
				tagPath = name + "/" + tagPath
			}
		}
		db.absoluteTagMap[id] = tagPath
	}
	return
}

func (db *fspotDb) photoLoop() (counter int, err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT id, description, time FROM photos WHERE id >= %d ORDER BY id;", *firstPhotoId))
	if err != nil {
		return
	}
	counter = 0
	for stmt.Next() {
		counter++
		var i int
		var timeStamp int64
		var desc string
		stmt.Scan(&i, &desc, &timeStamp)
		photoDate := time.Unix(timeStamp, 0)

		var tag map[int]string
		tag, err = db.tags(i)
		if err != nil {
			return
		}
		var url string
		url, err = db.imageUrl(i)
		if err != nil {
			return
		}
		var camliId string
		camliId, err = putFile(url)
		if err != nil {
			continue
		}
		if camliId != "" {
			err = setAttr(camliId, "fspot_id", fmt.Sprintf("%d", i))
			if err != nil {
				return
			}
			err = setAttr(camliId, "fspot_time", fmt.Sprintf("%v", photoDate))
			if err != nil {
				return
			}
			for k, v := range tag {
				err = addAttr(camliId, "tag", v)
				if err != nil {
					return
				}
				err = addAttr(camliId, "fspot_tag_path", db.absoluteTagMap[k])
				if err != nil {
					return
				}
			}
			if desc != "" {
				err = setAttr(camliId, "description", desc)
				if err != nil {
					return
				}
			}
		}
    if counter % 100 == 0 {
			log.Printf("INFO Put %d files. Clearing camlistore cache", counter)
    }
		if *clearCache {
			home := strings.TrimRight(os.Getenv("HOME"), "/")
			cmd := exec.Command("rm", "-rf", fmt.Sprintf("%s/.cache/camlistore", home))
			err = cmd.Run()
			if err != nil {
				log.Printf("WARNING Problems clearing camlistore cache error: %s", err)
			}
		}
	}
	return
}

func (db *fspotDb) tags(id int) (tags map[int]string, err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT id, name FROM photo_tags, tags WHERE photo_tags.photo_id=%d AND tags.id=photo_tags.tag_id", id))
	if err != nil {
		return
	}
	tags = make(map[int]string)
	for stmt.Next() {
		var name string
		var id int
		stmt.Scan(&id, &name)
		tags[id] = name
	}
	return
}

func (db *fspotDb) imageUrl(id int) (u string, err error) {
	stmt, err := db.Conn.Prepare(fmt.Sprintf("SELECT default_version_id, base_uri, filename FROM photos WHERE id=%d", id))
	if err != nil {
		return
	}
	if !stmt.Next() {
		err = errors.New("Not found")
	}
	var default_version_id int
	var base_uri, filename string
	stmt.Scan(&default_version_id, &base_uri, &filename)
	if default_version_id != 1 {
		stmt, err = db.Conn.Prepare(fmt.Sprintf("SELECT base_uri, filename FROM photo_versions WHERE version_id=%d  AND photo_id=%d", default_version_id, id))
		if err != nil {
			return
		}
		if !stmt.Next() {
			err = errors.New("Not found")
		}
		stmt.Scan(&base_uri, &filename)
	}
	u = base_uri + "/" + filename
	u = "/" + strings.TrimLeft(u, "file:/")
	u, err = url.QueryUnescape(u)
	return
}

func putFile(path string) (camliId string, err error) {
	args := []string{"file", "-filenodes", path}
	cmd := exec.Command(*camput, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR camput file -filenode %s FAILED with: %s", path, data)
		return
	}
	camliIdList := strings.Split(string(data), "\n")
	if len(camliIdList) != 4 {
		camliId = ""
		log.Printf("WARNING camput file -filenode %s output was not 3 lines long (duplicate content)", path)
	} else {
		camliId = camliIdList[0]
	}
	return
}

func setAttr(hash, key, value string) (err error) {
	args := []string{"attr", hash, key, value}
	cmd := exec.Command(*camput, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR camput attr %s %s %s  FAILED with: %s", hash, key, value, data)
	}
	return
}

func addAttr(hash, key, value string) (err error) {
	args := []string{"attr", "-add", hash, key, value}
	cmd := exec.Command(*camput, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR camput attr -add %s %s %s  FAILED with: %s", hash, key, value, data)
	}
	return
}

package main

import (
	"code.google.com/p/gosqlite/sqlite"
	"errors"
	"flag"
	"fmt"
	"net/url"

	"os/exec"
	"strings"
)

var (
	camput  = flag.String("camput", "bin/camput", "path to camput binary")
	photoDb = flag.String("db", "test_db", "path to F-Spot sqlitedb")
)

func main() {

	flag.Parse()
	conn, err := sqlite.Open(*photoDb)
	if err != nil {
		fmt.Println("Unable to open source database")
		return
	}
	defer conn.Close()
	err = loop(conn)
	if err != nil {
		fmt.Println("Error in main loop", err)
	} else {
		fmt.Println("Finished without errors!")
	}
}

func loop(conn *sqlite.Conn) (err error) {
	stmt, err := conn.Prepare("select id from photos;")
	if err != nil {
		return
	}
	for stmt.Next() {
		var i int
		stmt.Scan(&i)
		var tag []string
		tag, err = tags(conn, i)
		if err != nil {
			return
		}
		var url string
		url, err = imageUrl(conn, i)
		if err != nil {
			return
		}
		var camliId string
		camliId, err = putFile(url)
		if err != nil {
      fmt.Println("Problems processing: ", url, " error: ", err)
			continue
		}
		if camliId != "" {
			for _, v := range tag {
				err = addAttr(camliId, "tag", v)
				if err != nil {
					return
				}
			}
		}

	}
	return
}

func tags(conn *sqlite.Conn, id int) (tags []string, err error) {
	stmt, err := conn.Prepare(fmt.Sprintf("select name from photo_tags, tags where photo_tags.photo_id=%d and tags.id=photo_tags.tag_id", id))
	if err != nil {
		return
	}
	tags = make([]string, 0)
	for stmt.Next() {
		var name string
		stmt.Scan(&name)
		fmt.Println("Tag: ", name)
		tags = append(tags, name)
	}
	return
}

func imageUrl(conn *sqlite.Conn, id int) (u string, err error) {
	stmt, err := conn.Prepare(fmt.Sprintf("select default_version_id, base_uri, filename from photos where id=%d", id))
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
		stmt, err = conn.Prepare(fmt.Sprintf("select base_uri, filename from photo_versions where version_id=%d  and photo_id=%d", default_version_id, id))
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
	fmt.Println(args)
	cmd := exec.Command(*camput, args...)
	data, err := cmd.Output()
	if err != nil {
		return
	}
	fmt.Println(string(data))
	camliIdList := strings.Split(string(data), "\n")
	if len(camliIdList) != 4 {
		camliId = ""
	} else {
		camliId = camliIdList[0]
	}
	return
}

func addAttr(hash, key, value string) (err error) {
	args := []string{"attr", "-add", hash, key, value}
	fmt.Println(args)
	cmd := exec.Command(*camput, args...)
	err = cmd.Run()
	return
}

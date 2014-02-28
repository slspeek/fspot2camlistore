package main

import (
	"camlistore.org/pkg/client"
	"camlistore.org/pkg/schema"
	"camlistore.org/pkg/syncutil"
	"code.google.com/p/gosqlite/sqlite"
	"flag"
	"fmt"
	"github.com/slspeek/fspot2camlistore/fspot"
	"github.com/slspeek/fspot2camlistore/state"
	"log"
	"os"
	"sync"
	"time"
)

var (
	photoDb     = flag.String("db", "test_db", "path to F-Spot sqlitedb")
	camliClient = client.NewOrFail()
)

var wgWorkers sync.WaitGroup
var wgLogger sync.WaitGroup

func storePhoto(p fspot.Photo) (permaS string, err error) {
	f, err := os.Open(p.Path)
	if err != nil {
		return
	}
	defer f.Close()

	fileRef, err := schema.WriteFileFromReader(camliClient, p.Filename, f)

	res, err := camliClient.UploadNewPermanode()
	if err != nil {
		return
	}
	perma := res.BlobRef

	claims := []*schema.Builder{}
	claims = append(claims, schema.NewSetAttributeClaim(perma, "camliContent", fileRef.String()))
	claims = append(claims, schema.NewSetAttributeClaim(perma, "fspot_id", fmt.Sprintf("%d", p.Id)))
	claims = append(claims, schema.NewSetAttributeClaim(perma, "fspot_time", fmt.Sprintf("%v", p.Taken)))
	if p.Desc != "" {
		claims = append(claims, schema.NewSetAttributeClaim(perma, "description", p.Desc))
	}
	for _, t := range p.Tags {
		claims = append(claims, schema.NewAddAttributeClaim(perma, "tag", t))
	}
	for _, atp := range p.AbsTagPaths {
		claims = append(claims, schema.NewAddAttributeClaim(perma, "fspot_tag_path", atp))
	}

	grp := syncutil.Group{}
	for _, claimBuilder := range claims {
		claim := claimBuilder.Blob()
		grp.Go(func() error {
			_, err := camliClient.UploadAndSignBlob(claim)
			return err
		})
	}

	return perma.String(), grp.Err()
}

func handlePhotos(ch <-chan fspot.Photo, logger state.LogChan) {
	defer wgWorkers.Done()
	for p := range ch {
		perma, err := storePhoto(p)
		logRecord := state.Log{p.Id, perma, err}
		if err != nil {
			log.Printf("Couldn't store %v: %v", p.Id, err)
		}
		log.Printf("Stored %d as %v", p.Id, perma)
		logger <- logRecord
	}
}

func main() {
	flag.Parse()
	stateDb, err := state.Open()
	if err != nil {
		log.Fatalf("Unable to open state recording database because: %v", err)
	}
	err = stateDb.Init()
	if err != nil {
		log.Fatalf("Unable to create table because: %v", err)
	}

	firstPhotoId, err := stateDb.MaxId()
	if err != nil {
		log.Fatalf("Unable to determine start position because: %v", err)
	}
	firstPhotoId++
	wgLogger.Add(1)
	go func() {
		defer wgLogger.Done()
		for record := range stateDb.Chan {
			stateDb.Log(record)
		}
	}()

	conn, err := sqlite.Open(*photoDb)
	if err != nil {
		log.Fatalf("Unable to open f-spot source database because: %v", err)
	}
	defer conn.Close()
	db := fspot.Db{conn, make(map[int]string)}
	err = db.CalculateTagPaths()
	if err != nil {
		log.Fatalf("Unable to populate tags paths: %v", err)
	}

	ch := make(chan fspot.Photo)

	for i := 0; i < 4; i++ {
		wgWorkers.Add(1)
		go handlePhotos(ch, stateDb.Chan)
	}

	start := time.Now()
	err = db.PhotoLoop(firstPhotoId, ch)
	close(ch)
	if err != nil {
		log.Fatalf("Error reading in photos from fspotdb: %v", err)
	}

	log.Printf("Waiting for queued tasks to complete.")
	wgWorkers.Wait()
	close(stateDb.Chan)
	log.Printf("Waiting for log records to be written.")
	wgLogger.Wait()
	log.Printf("Finished in %v", time.Since(start))
}

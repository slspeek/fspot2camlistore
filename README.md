fspot2camlistore
================

Import your F-Spot collections to Camlistore. This tool adds the following F-Spot photo attributes to the permanode associated with the photo's default version image:

  * fspot_id, the id of the photo in F-Spot
  * fspot_time, the time extracted from exif by F-Spot (propably redundant)
  * tags for all the F-Spot tags associated with this photo
  * fspot_tag_path, all full tag paths associated with the photo
  * description, if it was set in F-Spot

## Installation:

```
go get github.com/slspeek/fspot2camlistore
```

## Running
Point to your f-spot-database with the option -db.
The tool makes of file system copy of this database, and works on the copy.
Typically:

```
./fspot2camlistore -db $HOME/.config/f-spot/photos.db
```
It can be run serveral times over a period of time as it keeps tracks of the work done in ~/.config/fspot2camlistore/state.db.
It will start where it left the last time.

You can check for errors with the command
```
sqlitebrowser $HOME/.config/fspot2camlistore/state.db
```
##Credits
I copied the concurrency ideas and use of the camlistore client api from:
https://github.com/dustin/photo-couch/blob/master/tools/phototocamli/tocamli.go

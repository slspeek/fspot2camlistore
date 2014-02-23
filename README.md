fspot2camlistore
================

Import your F-Spot collections to Camlistore. This tools adds the following F-Spot photo attributes to the permanode associated with the photo's default version image:

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
I recommend making a copy of your fspot-database before running fspot2camli on it, alltought it does not intend to modify it.
Then point to your f-spot-database with the option -db.
Typcally:

```
./fspot2camlistore -db $HOME/.config/f-spot/photos.db
```

##Credits
I copied the concurrency ideas and use of the camlistore client api from:
https://github.com/dustin/photo-couch/blob/master/tools/phototocamli/tocamli.go

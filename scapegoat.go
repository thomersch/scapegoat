package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/thomersch/scapegoat/tags"

	"github.com/thomersch/grandine/lib/geojson"
	"github.com/thomersch/grandine/lib/mvt"
	"github.com/thomersch/grandine/lib/spatial"
	"github.com/thomersch/grandine/lib/tile"
)

type featuresInTile struct {
	Features []spatial.Feature
	Tile     tile.ID
}

var (
	zoom      = 7
	josmURL   = "https://josm.openstreetmap.de/maps?format=geojson"
	tilesPath = "tiles/"
)

func main() {
	loop() // head start
	for range time.Tick(15 * time.Minute) {
		loop()
	}
}

func loop() {
	buf := fetch("state-file")
	if len(buf) == 0 {
		log.Println("Nothing to be done.")
		return
	}
	r := bytes.NewBuffer(buf)
	err := generate(r, tilesPath)
	if err != nil {
		log.Printf("generation failed: %s", err)
	}
}

func lastMod(lastModPath string) string {
	lmdFile, err := os.Open(lastModPath)
	defer lmdFile.Close()
	if err != nil {
		return ""
	}
	buf, err := ioutil.ReadAll(lmdFile)
	if err != nil {
		return ""
	}
	return string(buf)
}

func fetch(lastModPath string) []byte {
	lmd := lastMod(lastModPath)
	hd, err := http.Head(josmURL)
	if err != nil {
		return nil
	}

	if lmd == hd.Header.Get("Last-Modified") {
		return nil
	}

	log.Println("Attempting to fetch new JSON")
	jr, err := http.Get(josmURL)
	if err != nil {
		log.Printf("error while fetching JOSM JSON: %s", err)
		return nil
	}
	defer jr.Body.Close()

	err = ioutil.WriteFile(lastModPath, []byte(jr.Header.Get("Last-Modified")), 0644)
	if err != nil {
		log.Printf("could not write state file: %s", err)
	}

	buf, err := ioutil.ReadAll(jr.Body)
	if err != nil {
		log.Printf("could not read response over http: %s", err)
		return nil
	}
	return buf
}

func generate(src io.Reader, outPath string) error {
	// tile target
	zlPath := filepath.Join(outPath, strconv.Itoa(zoom))
	err := os.MkdirAll(zlPath, 0777)
	if err != nil {
		return err
	}

	// decode GeoJSON
	var (
		fcoll = spatial.FeatureCollection{}
		gj    = &geojson.Codec{}
		ftab  = featureTable([]int{zoom})
	)
	err = gj.Decode(src, &fcoll)
	if err != nil {
		return err
	}

	for i, ft := range fcoll.Features {
		// If we have no geometry, cover the whole planet.
		if ft.Geometry.Typ() == spatial.GeomTypeEmpty {
			fcoll.Features[i].Geometry = spatial.MustNewGeom(spatial.Polygon{{
				{180, 90},
				{-180, 90},
				{-180, -90},
				{180, -90},
			}})
		}

		attr, ok := ft.Properties()["attribution"].(map[string]interface{})
		if !ok {
			continue
		}
		ft.Props["attribution-text"] = attr["text"]
		ft.Props["attribution-url"] = attr["url"]
		delete(ft.Props, "attribution")

		for _, tid := range tile.Coverage(fcoll.Features[i].Geometry.BBox(), zoom) {
			ftab[zoom][tid.X][tid.Y] = append(ftab[zoom][tid.X][tid.Y], fcoll.Features[i])
		}
	}

	var (
		ftChan      = make(chan featuresInTile, 10000)
		encoderDone = startEncoders(ftChan, outPath)
	)

	for x, xl := range ftab[zoom] {
		xPath := filepath.Join(zlPath, strconv.Itoa(x))
		err = os.MkdirAll(xPath, 0777)
		if err != nil {
			log.Fatalf("could not create subpath: %v", err)
		}

		log.Printf("Preparing %v/%v, Queue Depth: %v", zoom, x, len(ftChan))

		for y, yl := range xl {
			var fl = make([]spatial.Feature, 0, len(yl))
			tID := tile.ID{Z: zoom, X: x, Y: y}

			for _, f := range yl {
				for _, g := range f.Geometry.ClipToBBox(tID.BBox()) {
					fl = append(fl, spatial.Feature{Props: f.Props, Geometry: g})
				}
			}

			ftChan <- featuresInTile{Features: fl, Tile: tID}
		}
	}
	close(ftChan)
	log.Println("Waiting for encoders...")

	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			log.Printf("Left in Queue: %v", len(ftChan))
		}
	}()
	<-encoderDone
	ticker.Stop()
	return nil
}

func startEncoders(c <-chan featuresInTile, basepath string) <-chan bool {
	var (
		gjQueue  = make(chan featuresInTile, 100)
		mvtQueue = make(chan featuresInTile, 100)

		wg   sync.WaitGroup
		done = make(chan bool)
	)

	wg.Add(1)
	go func() {
		gfc := &tile.GeoJSONCodec{}
		worker(gjQueue, gfc, basepath, ".geojson")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		var (
			mvtc          = &mvt.Codec{}
			innerMvtQueue = make(chan featuresInTile, 100)
		)

		go func() {
			// MVT does not support nested objects, so we need to post-process here
			for i := range mvtQueue {
				innerMvtQueue <- featuresInTile{Features: tags.FilterFeaturesForID(i.Features), Tile: i.Tile}
			}
			close(innerMvtQueue)
		}()
		worker(innerMvtQueue, mvtc, basepath, ".mvt")
		wg.Done()
	}()

	go func() {
		for ft := range c {
			gjQueue <- ft
			mvtQueue <- ft
		}
		close(gjQueue)
		close(mvtQueue)
		wg.Wait()
		done <- true
	}()

	return done
}

func worker(c <-chan featuresInTile, enc tile.Codec, basepath string, extension string) {
	for ftt := range c {
		t, err := enc.EncodeTile(map[string][]spatial.Feature{"imagery": ftt.Features}, ftt.Tile)
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(tilepath(ftt.Tile, basepath, extension), t, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func tilepath(tID tile.ID, basepath string, extension string) string {
	return filepath.Join(basepath, strconv.Itoa(tID.Z), strconv.Itoa(tID.X), strconv.Itoa(tID.Y)+extension)
}

func pow(x, y int) int {
	var res = 1
	for i := 1; i <= y; i++ {
		res *= x
	}
	return res
}

func featureTable(zls []int) map[int][][][]spatial.Feature {
	r := map[int][][][]spatial.Feature{}
	for _, zl := range zls {
		l := pow(2, zl)
		r[zl] = make([][][]spatial.Feature, l)
		for x := range r[zl] {
			r[zl][x] = make([][]spatial.Feature, l)
		}
	}
	return r
}

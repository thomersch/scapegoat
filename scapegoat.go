package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/thomersch/grandine/lib/geojson"
	"github.com/thomersch/grandine/lib/spatial"
	"github.com/thomersch/grandine/lib/tile"
)

func main() {
	source := flag.String("in", "", "file to read from, supported format")
	target := flag.String("out", "tiles", "path where the tiles will be written")
	zoom := flag.Int("zoom", 7, "zoom level")

	flag.Parse()

	// source
	f, err := os.Open(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// tile target
	zlPath := filepath.Join(*target, strconv.Itoa(*zoom))
	err = os.MkdirAll(zlPath, 0777)
	if err != nil {
		log.Fatal(err)
	}

	// decode GeoJSON
	var (
		fcoll = spatial.FeatureCollection{}
		gj    = &geojson.Codec{}
		ftab  = featureTable([]int{*zoom})
	)
	err = gj.Decode(f, &fcoll)
	if err != nil {
		log.Fatal(err)
	}

	for i, ft := range fcoll.Features {
		if ft.Geometry.Typ() == spatial.GeomTypeEmpty {
			fcoll.Features[i].Geometry = spatial.MustNewGeom(spatial.Polygon{{
				{-180, -90},
				{180, -90},
				{180, 90},
				{-180, 90},
			}})
		}

		attr, ok := ft.Properties()["attribution"].(map[string]interface{})
		if !ok {
			continue
		}
		ft.Props["attribution-text"] = attr["text"]
		ft.Props["attribution-url"] = attr["url"]
		delete(ft.Props, "attribution")

		for _, tid := range tile.Coverage(fcoll.Features[i].Geometry.BBox(), *zoom) {
			ftab[*zoom][tid.X][tid.Y] = append(ftab[*zoom][tid.X][tid.Y], &fcoll.Features[i])
		}
	}

	tileCodec := &tile.GeoJSONCodec{}

	for x, xl := range ftab[*zoom] {
		xPath := filepath.Join(zlPath, strconv.Itoa(x))
		err = os.MkdirAll(xPath, 0777)
		if err != nil {
			log.Fatalf("could not create subpath: %v", err)
		}

		log.Printf("Preparing %v/%v", *zoom, x)

		for y, yl := range xl {
			var fl = make([]spatial.Feature, 0, len(yl))
			tID := tile.ID{Z: *zoom, X: x, Y: y}

			for _, f := range yl {
				fc := *f
				for _, g := range fc.Geometry.ClipToBBox(tID.BBox()) {
					fl = append(fl, spatial.Feature{Props: f.Props, Geometry: g})
				}
			}
			t, err := tileCodec.EncodeTile(map[string][]spatial.Feature{"imagery": fl}, tID)
			if err != nil {
				log.Fatal(err)
			}

			tf, err := os.Create(filepath.Join(xPath, strconv.Itoa(y)+".geojson"))
			if err != nil {
				log.Fatalf("could not create tile file: %v", err)
			}
			tf.Write(t)
			tf.Close()
		}
	}
}

func pow(x, y int) int {
	var res = 1
	for i := 1; i <= y; i++ {
		res *= x
	}
	return res
}

func featureTable(zls []int) map[int][][][]*spatial.Feature {
	r := map[int][][][]*spatial.Feature{}
	for _, zl := range zls {
		l := pow(2, zl)
		r[zl] = make([][][]*spatial.Feature, l)
		for x := range r[zl] {
			r[zl][x] = make([][]*spatial.Feature, l)
		}
	}
	return r
}

package tags

import (
	"strings"

	"github.com/thomersch/grandine/lib/spatial"
)

var (
	webmercator = map[string]bool{
		"EPSG:3857":   true,
		"EPSG:3587":   true,
		"EPSG:3785":   true,
		"EPSG:41001":  true,
		"EPSG:54004":  true,
		"EPSG:102113": true,
		"EPSG:102100": true,
		"EPSG:900913": true,
	}

	wgs84 = "EPSG:4326"
)

// FilterFeaturesForID takes a list of pre-processed features and removes those which do
// not support web mercator or 4326.
func FilterFeaturesForID(fts []spatial.Feature) []spatial.Feature {
	var out = make([]spatial.Feature, 0, len(fts))

	for _, ft := range fts {
		var (
			hasWGSFallback bool
			ftWritten      bool
		)

		projs, ok := ft.Props["available_projections"]
		if !ok {
			out = append(out, ft)
			continue
		}
		for _, proj := range projs.([]interface{}) {
			if webmercator[proj.(string)] {
				setProjTags(ft.Props, proj.(string))
				out = append(out, ft)
				ftWritten = true
				break
			}
			if proj.(string) == wgs84 {
				hasWGSFallback = true
			}
		}

		if hasWGSFallback && !ftWritten {
			setProjTags(ft.Props, wgs84)
			out = append(out, ft)
		}
	}
	return out
}

// setProjTags changes the URL in WMS, sets the projection tag and scrubs the list
// of available projections
func setProjTags(props map[string]interface{}, proj string) {
	if _, ok := props["url"]; ok {
		props["url"] = strings.ReplaceAll(props["url"].(string), "{proj}", proj)
	}
	props["srid"] = proj
	delete(props, "available_projections")
}

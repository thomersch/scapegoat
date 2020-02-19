package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomersch/grandine/lib/spatial"
)

func TestMVTTags(t *testing.T) {
	fts := []spatial.Feature{
		{
			Props: map[string]interface{}{
				"id": 4,
				"available_projections": []interface{}{"EPSG:ASDF", "EPSG:3857"},
				"url": "http://example.com?proj={proj}",
			},
		},
		{
			// Degenerate case where we have a valid projection, but no URL
			Props: map[string]interface{}{
				"id": 5,
				"available_projections": []interface{}{"EPSG:ASDF", "EPSG:3857"},
			},
		},
		{
			// We use 4326 as fallback.
			Props: map[string]interface{}{
				"id": 6,
				"available_projections": []interface{}{"EPSG:FOO", "EPSG:4326", "EPSG:BAR"},
				"url": "http://example.com?proj={proj}",
			},
		},
		{
			// 4326 is present, but there is a preferable web mercator proj available.
			Props: map[string]interface{}{
				"id": 7,
				"available_projections": []interface{}{"EPSG:4326", "EPSG:3857"},
				"url": "http://example.com?proj={proj}",
			},
		},
		{
			// No good projection available. Skip feature alltogether.
			Props: map[string]interface{}{
				"id": 8,
				"available_projections": []interface{}{"EPSG:9999"},
				"url": "http://example.com?proj={proj}",
			},
		},
	}

	newfts := FilterFeaturesForID(fts)

	assert.Equal(t, []spatial.Feature{
		{
			Props: map[string]interface{}{
				"id":   4,
				"srid": "EPSG:3857",
				"url":  "http://example.com?proj=EPSG:3857",
			},
		},
		{
			Props: map[string]interface{}{
				"id":   5,
				"srid": "EPSG:3857",
			},
		},
		{
			Props: map[string]interface{}{
				"id":   6,
				"url":  "http://example.com?proj=EPSG:4326",
				"srid": "EPSG:4326",
			},
		},
		{
			Props: map[string]interface{}{
				"id":   7,
				"url":  "http://example.com?proj=EPSG:3857",
				"srid": "EPSG:3857",
			},
		},
	}, newfts)
}

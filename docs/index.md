# Imagery Index, code name "Scapegoat"

## URL

	http://{ host }/tiles/{z}/{x}/{y}.{ext}

* `z`: Zoom level, we only offer zoom level 7
* `x`/`y`: X/Y tiles, as defined by [Slippy Map Tilenames](https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames)
* `ext`: We offer three file formats:
	* `geojson`: GeoJSON tile with as much information as possible, including different projections
	* `web.geojson`: GeoJSON tile for web clients only with layers which support web mercator or mercator (3857/3587/3785/41001/54004/102113/102100/900913).
	* `mvt`: Mapbox Vector Tile with the same information as `web.geojson`

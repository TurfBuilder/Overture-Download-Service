package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/paulmach/orb/geojson"
)

// Place is one row of the CSV result set.
type Place struct {
	Name          string
	AddressLine1  string
	AddressLine2  string // Overture has no second address line; always empty
	City          string
	StateOrRegion string
	PostalCode    string
	CountryCode   string
	Latitude      float64
	Longitude     float64
}

// fetchPlacesWithin queries the public Overture Maps places theme for all
// places (businesses/POIs) inside the given GeoJSON polygon.
func fetchPlacesWithin(ctx context.Context, release string, area *geojson.Geometry) ([]Place, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}
	defer db.Close()

	// The spatial extension provides the ST_* functions; httpfs reads the
	// Overture parquet files straight out of their public S3 bucket.
	setup := []string{
		"INSTALL spatial",
		"LOAD spatial",
		"INSTALL httpfs",
		"LOAD httpfs",
		"SET s3_region='us-west-2'",
	}
	for _, stmt := range setup {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return nil, fmt.Errorf("duckdb setup %q: %w", stmt, err)
		}
	}

	areaJSON, err := json.Marshal(area)
	if err != nil {
		return nil, fmt.Errorf("re-encoding area: %w", err)
	}

	// Coarse filter on the precomputed bbox columns (fast, pushed down into
	// the parquet scan), then the exact point-in-polygon test. A place's
	// primary address lives in the first element of its addresses list.
	bound := area.Geometry().Bound()
	query := fmt.Sprintf(`
		SELECT name, address, city, region, postcode, country,
		       ST_Y(geom) AS lat, ST_X(geom) AS lon
		FROM (
			SELECT
				names.primary          AS name,
				addresses[1].freeform  AS address,
				addresses[1].locality  AS city,
				addresses[1].region    AS region,
				addresses[1].postcode  AS postcode,
				addresses[1].country   AS country,
				geometry               AS geom
			FROM read_parquet('s3://overturemaps-us-west-2/release/%s/theme=places/type=place/*', hive_partitioning=1)
			WHERE bbox.xmin >= ? AND bbox.xmax <= ?
			  AND bbox.ymin >= ? AND bbox.ymax <= ?
		)
		WHERE ST_Within(geom, ST_GeomFromGeoJSON(?))`, release)

	rows, err := db.QueryContext(ctx, query,
		bound.Min[0], bound.Max[0],
		bound.Min[1], bound.Max[1],
		string(areaJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("querying overture places: %w", err)
	}
	defer rows.Close()

	var places []Place
	for rows.Next() {
		var (
			name, address, city, region, postcode, country sql.NullString
			lat, lon                                       float64
		)
		if err := rows.Scan(&name, &address, &city, &region, &postcode, &country, &lat, &lon); err != nil {
			return nil, fmt.Errorf("scanning place row: %w", err)
		}

		places = append(places, Place{
			Name:          name.String,
			AddressLine1:  address.String,
			City:          city.String,
			StateOrRegion: region.String,
			PostalCode:    postcode.String,
			CountryCode:   country.String,
			Latitude:      lat,
			Longitude:     lon,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading place rows: %w", err)
	}

	return places, nil
}

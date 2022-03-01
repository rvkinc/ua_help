package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type locality struct {
	ID        int    `json:"id"`
	UUID      string `json:"uuid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Meta      struct {
		OsmID             interface{} `json:"osm_id"`
		GoogleMapsPlaceID string      `json:"google_maps_place_id"`
	} `json:"meta"`
	Type string `json:"type"`
	Name struct {
		En string `json:"en"`
		Ru string `json:"ru"`
		Uk string `json:"uk"`
	} `json:"name"`
	PublicName struct {
		En string `json:"en"`
		Ru string `json:"ru"`
		Uk string `json:"uk"`
	} `json:"public_name"`
	PostCode []string `json:"post_code"`
	Katottg  string   `json:"katottg"`
	Koatuu   string   `json:"koatuu"`
	Lng      float64  `json:"lng"`
	Lat      float64  `json:"lat"`
	ParentID int      `json:"parent_id"`
}

//go:embed localities.json
var b []byte

func main() {

	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "localhost", 5432, "postgres", "secret", "postgres")

	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		log.Fatal(err)
	}

	var localities []locality
	if err = json.Unmarshal(b, &localities); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, l := range localities {
		if _, err := tx.Exec(
			"INSERT INTO locality(id, type, name_ru, name_ua, name_eu, public_name_ua, public_name_ru, public_name_eu, parent_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)",
			l.ID,
			l.Type,
			l.Name.Ru,
			l.Name.Uk,
			l.Name.En,
			l.PublicName.Uk,
			l.PublicName.Ru,
			l.PublicName.En,
			l.ParentID,
		); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

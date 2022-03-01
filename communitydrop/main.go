package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"strings"
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
	var localities []locality
	if err := json.Unmarshal(b, &localities); err != nil {
		log.Fatal(err)
	}

	localityMap := make(map[int][]locality)
	communityID := make(map[int]int)
	// fmt.Println(len(localities))

	for _, l := range localities {
		if strings.Contains(l.PublicName.En, "community") {
			communityID[l.ID] = l.ParentID
			continue
		}

		localityMap[l.ParentID] = append(localityMap[l.ParentID], l)
	}

	for com, parent := range communityID {
		v, ok := localityMap[com]
		if ok {
			for loc := range v {
				v[loc].ParentID = parent
			}
			localityMap[com] = v
		}
	}

	f, err := os.Create("locality_2.json")
	if err != nil {
		log.Fatalf("can't create file %s", err)
	}
	defer f.Close()

	ll := make([]locality, 0, 1000)

	for _, v := range localityMap {
		ll = append(ll, v...)
	}
	// fmt.Println(len(ll))

	bb, err := json.Marshal(ll)
	if err != nil {
		log.Fatalf("can't marshal json %s", err)
	}

	_, err = f.Write(bb)
	if err != nil {
		log.Fatalf("can't write into file %s", err)
	}
}

package providerb

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
)

type lotB struct {
	UID         string  `json:"uid"`
	Title       string  `json:"title"`
	Geo         string  `json:"geo"`
	SpacesTotal int     `json:"spaces_total"`
	SpacesFree  int     `json:"spaces_free"`
	Price       float64 `json:"price"`
	PriceCur    string  `json:"price_currency"`
	FullAddress string  `json:"full_address"`
}

var seedLots = []lotB{
	{
		UID:         "B001",
		Title:       "Tiergarten Parkhaus",
		Geo:         "52.5145,13.3501",
		SpacesTotal: 180,
		Price:       3.00,
		PriceCur:    "EUR",
		FullAddress: "Straße des 17. Juni 135",
	},
	{
		UID:         "B002",
		Title:       "Kreuzberg Underground",
		Geo:         "52.4990,13.4030",
		SpacesTotal: 220,
		Price:       2.50,
		PriceCur:    "EUR",
		FullAddress: "Oranienstraße 10",
	},
	{
		UID:         "B003",
		Title:       "Prenzlauer Berg Parking",
		Geo:         "52.5386,13.4244",
		SpacesTotal: 120,
		Price:       2.00,
		PriceCur:    "EUR",
		FullAddress: "Kollwitzstraße 5",
	},
	{
		UID:         "B004",
		Title:       "Friedrichshain East Side",
		Geo:         "52.5163,13.4542",
		SpacesTotal: 90,
		Price:       1.80,
		PriceCur:    "EUR",
		FullAddress: "Warschauer Straße 23",
	},
}

func StartMockServer(port int) string {
	mux := http.NewServeMux()
	mux.HandleFunc("/parking-zones", func(w http.ResponseWriter, r *http.Request) {
		lots := make([]lotB, len(seedLots))
		for i, lot := range seedLots {
			lot.SpacesFree = rand.Intn(lot.SpacesTotal + 1)
			lots[i] = lot
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lots); err != nil {
			log.Printf("provider B: encode response: %v", err)
		}
	})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("provider B: listen on :%d: %v", port, err)
	}

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("provider B: server exited: %v", err)
		}
	}()

	addr := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("Provider B mock server listening on %s", addr)
	return addr
}

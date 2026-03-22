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
		Title:       "North Park",
		Geo:         "55.7700,37.5900",
		SpacesTotal: 80,
		Price:       3.00,
		PriceCur:    "USD",
		FullAddress: "45 North Ave",
	},
	{
		UID:         "B002",
		Title:       "Arbat Underground",
		Geo:         "55.7522,37.5920",
		SpacesTotal: 200,
		Price:       4.00,
		PriceCur:    "USD",
		FullAddress: "10 Old Arbat St",
	},
	{
		UID:         "B003",
		Title:       "Eastern Business Park",
		Geo:         "55.7650,37.7000",
		SpacesTotal: 120,
		Price:       2.00,
		PriceCur:    "USD",
		FullAddress: "88 Business Rd",
	},
	{
		UID:         "B004",
		Title:       "Green Park South",
		Geo:         "55.7300,37.6100",
		SpacesTotal: 60,
		Price:       1.50,
		PriceCur:    "USD",
		FullAddress: "3 Park Lane",
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

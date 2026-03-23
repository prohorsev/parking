package providera

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
)

type lotA struct {
	ParkingID    string    `json:"parking_id"`
	Name         string    `json:"name"`
	Location     locationA `json:"location"`
	TotalCap     int       `json:"total_capacity"`
	Available    int       `json:"available_spots"`
	HourlyRate   float64   `json:"hourly_rate"`
	CurrencyCode string    `json:"currency_code"`
	Street       string    `json:"street"`
}

type locationA struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

var seedLots = []lotA{
	{
		ParkingID:    "a-001",
		Name:         "Mitte Parkhaus",
		Location:     locationA{Lat: 52.5200, Lon: 13.4050},
		TotalCap:     150,
		HourlyRate:   3.50,
		CurrencyCode: "EUR",
		Street:       "Unter den Linden 1",
	},
	{
		ParkingID:    "a-002",
		Name:         "Potsdamer Platz Garage",
		Location:     locationA{Lat: 52.5096, Lon: 13.3760},
		TotalCap:     400,
		HourlyRate:   4.00,
		CurrencyCode: "EUR",
		Street:       "Potsdamer Platz 1",
	},
	{
		ParkingID:    "a-003",
		Name:         "Alexanderplatz Parkplatz",
		Location:     locationA{Lat: 52.5219, Lon: 13.4132},
		TotalCap:     300,
		HourlyRate:   3.00,
		CurrencyCode: "EUR",
		Street:       "Alexanderplatz 7",
	},
	{
		ParkingID:    "a-004",
		Name:         "Charlottenburg Tiefgarage",
		Location:     locationA{Lat: 52.5168, Lon: 13.3040},
		TotalCap:     200,
		HourlyRate:   2.50,
		CurrencyCode: "EUR",
		Street:       "Kurfürstendamm 42",
	},
}

func StartMockServer(port int) string {
	mux := http.NewServeMux()
	mux.HandleFunc("/lots", func(w http.ResponseWriter, r *http.Request) {
		lots := make([]lotA, len(seedLots))
		for i, lot := range seedLots {
			lot.Available = rand.Intn(lot.TotalCap + 1)
			lots[i] = lot
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lots); err != nil {
			log.Printf("provider A: encode response: %v", err)
		}
	})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("provider A: listen on :%d: %v", port, err)
	}

	go func() {
		if err := http.Serve(ln, mux); err != nil {
			log.Printf("provider A: server exited: %v", err)
		}
	}()

	addr := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("Provider A mock server listening on %s", addr)
	return addr
}

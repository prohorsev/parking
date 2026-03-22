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
		Name:         "City Centre Garage",
		Location:     locationA{Lat: 55.7558, Lon: 37.6173},
		TotalCap:     150,
		HourlyRate:   2.50,
		CurrencyCode: "USD",
		Street:       "1 Red Square",
	},
	{
		ParkingID:    "a-002",
		Name:         "Riverside Parking",
		Location:     locationA{Lat: 55.7490, Lon: 37.6200},
		TotalCap:     80,
		HourlyRate:   1.80,
		CurrencyCode: "USD",
		Street:       "14 Embankment St",
	},
	{
		ParkingID:    "a-003",
		Name:         "Mall Parking A",
		Location:     locationA{Lat: 55.7600, Lon: 37.6050},
		TotalCap:     300,
		HourlyRate:   3.00,
		CurrencyCode: "USD",
		Street:       "5 Shopping Ave",
	},
	{
		ParkingID:    "a-004",
		Name:         "Business District Lot",
		Location:     locationA{Lat: 55.7650, Lon: 37.6350},
		TotalCap:     200,
		HourlyRate:   3.50,
		CurrencyCode: "USD",
		Street:       "22 Finance St",
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

package domain

import "time"

type ParkingLot struct {
	ID           string    `json:"id"             db:"id"`
	ExternalID   string    `json:"external_id"    db:"external_id"`
	Provider     string    `json:"provider"       db:"provider"`
	Name         string    `json:"name"           db:"name"`
	Address      string    `json:"address"        db:"address"`
	Lat          float64   `json:"lat"            db:"lat"`
	Lng          float64   `json:"lng"            db:"lng"`
	TotalSpots   int       `json:"total_spots"    db:"total_spots"`
	FreeSpots    int       `json:"free_spots"     db:"free_spots"`
	PricePerHour float64   `json:"price_per_hour" db:"price_per_hour"`
	Currency     string    `json:"currency"       db:"currency"`
	UpdatedAt    time.Time `json:"updated_at"     db:"updated_at"`
}

type ParkingLotWithDistance struct {
	ParkingLot
	DistanceKm float64 `json:"distance_km"`
}

type SearchParams struct {
	Lat      float64
	Lng      float64
	RadiusKm float64
}

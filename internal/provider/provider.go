package provider

import "github.com/parking/api/internal/domain"

type Provider interface {
	Name() string
	FetchAll() ([]domain.ParkingLot, error)
}

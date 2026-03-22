package repository

import "github.com/parking/api/internal/domain"

type Repository interface {
	Upsert(lots []domain.ParkingLot) error
	FindNearby(params domain.SearchParams) ([]domain.ParkingLot, error)
	FindByID(id string) (*domain.ParkingLot, error)
}

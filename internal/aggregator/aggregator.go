package aggregator

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/parking/api/internal/domain"
	"github.com/parking/api/internal/provider"
	"github.com/parking/api/internal/repository"
)

type Aggregator struct {
	repo      repository.Repository
	providers []provider.Provider
}

func New(repo repository.Repository, providers ...provider.Provider) *Aggregator {
	return &Aggregator{repo: repo, providers: providers}
}

func (a *Aggregator) Refresh() error {
	var all []domain.ParkingLot
	var lastErr error

	for _, p := range a.providers {
		lots, err := p.FetchAll()
		if err != nil {
			log.Printf("aggregator: provider %q error: %v", p.Name(), err)
			lastErr = err
			continue
		}
		log.Printf("aggregator: fetched %d lots from %q", len(lots), p.Name())
		all = append(all, lots...)
	}

	if len(all) == 0 {
		return fmt.Errorf("aggregator: all providers failed, last error: %w", lastErr)
	}

	if err := a.repo.Upsert(all); err != nil {
		return fmt.Errorf("aggregator: upsert: %w", err)
	}

	log.Printf("aggregator: upserted %d lots total", len(all))
	return nil
}

func (a *Aggregator) StartRefreshLoop(ctx context.Context, interval time.Duration) {
	if err := a.Refresh(); err != nil {
		log.Printf("aggregator: initial refresh error: %v", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := a.Refresh(); err != nil {
				log.Printf("aggregator: refresh error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Aggregator) GetNearby(params domain.SearchParams) ([]domain.ParkingLotWithDistance, error) {
	lots, err := a.repo.FindNearby(params)
	if err != nil {
		return nil, fmt.Errorf("get nearby: %w", err)
	}

	result := withDistances(params.Lat, params.Lng, lots)
	sort.Slice(result, func(i, j int) bool {
		return result[i].DistanceKm < result[j].DistanceKm
	})
	return result, nil
}

func (a *Aggregator) GetByID(id string) (*domain.ParkingLot, error) {
	lot, err := a.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}
	return lot, nil
}

func withDistances(lat, lng float64, lots []domain.ParkingLot) []domain.ParkingLotWithDistance {
	result := make([]domain.ParkingLotWithDistance, len(lots))
	for i, lot := range lots {
		result[i] = domain.ParkingLotWithDistance{
			ParkingLot: lot,
			DistanceKm: haversine(lat, lng, lot.Lat, lot.Lng),
		}
	}
	return result
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

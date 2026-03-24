package aggregator

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parking/api/internal/domain"
)

type mockRepo struct {
	upsertFn   func(lots []domain.ParkingLot) error
	findNearby func(params domain.SearchParams) ([]domain.ParkingLot, error)
	findByID   func(id string) (*domain.ParkingLot, error)
}

func (m *mockRepo) Upsert(lots []domain.ParkingLot) error {
	return m.upsertFn(lots)
}

func (m *mockRepo) FindNearby(params domain.SearchParams) ([]domain.ParkingLot, error) {
	return m.findNearby(params)
}

func (m *mockRepo) FindByID(id string) (*domain.ParkingLot, error) {
	return m.findByID(id)
}

type mockProvider struct {
	name     string
	fetchAll func() ([]domain.ParkingLot, error)
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) FetchAll() ([]domain.ParkingLot, error) { return m.fetchAll() }

var (
	lotA = domain.ParkingLot{ID: "a", Name: "Lot A", Lat: 52.0, Lng: 13.0}
	lotB = domain.ParkingLot{ID: "b", Name: "Lot B", Lat: 52.1, Lng: 13.1}
	lotC = domain.ParkingLot{ID: "c", Name: "Lot C", Lat: 51.5, Lng: 12.5}
)

func okRepo(upserted *[]domain.ParkingLot) *mockRepo {
	return &mockRepo{
		upsertFn: func(lots []domain.ParkingLot) error {
			*upserted = lots
			return nil
		},
	}
}

func TestRefresh_AllProvidersSucceed(t *testing.T) {
	var upserted []domain.ParkingLot
	repo := okRepo(&upserted)

	p1 := &mockProvider{name: "p1", fetchAll: func() ([]domain.ParkingLot, error) { return []domain.ParkingLot{lotA}, nil }}
	p2 := &mockProvider{name: "p2", fetchAll: func() ([]domain.ParkingLot, error) { return []domain.ParkingLot{lotB, lotC}, nil }}

	agg := New(repo, p1, p2)
	if err := agg.Refresh(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(upserted) != 3 {
		t.Errorf("expected 3 lots upserted, got %d", len(upserted))
	}
}

func TestRefresh_OneProviderFails_OtherSucceeds(t *testing.T) {
	var upserted []domain.ParkingLot
	repo := okRepo(&upserted)

	errProvider := &mockProvider{name: "bad", fetchAll: func() ([]domain.ParkingLot, error) {
		return nil, errors.New("fetch failed")
	}}
	goodProvider := &mockProvider{name: "good", fetchAll: func() ([]domain.ParkingLot, error) {
		return []domain.ParkingLot{lotA}, nil
	}}

	agg := New(repo, errProvider, goodProvider)
	if err := agg.Refresh(); err != nil {
		t.Fatalf("unexpected error when at least one provider succeeds: %v", err)
	}

	if len(upserted) != 1 || upserted[0].ID != "a" {
		t.Errorf("expected lot A from good provider, got %+v", upserted)
	}
}

func TestRefresh_AllProvidersFail_ReturnsError(t *testing.T) {
	upsertCalled := false
	repo := &mockRepo{
		upsertFn: func(lots []domain.ParkingLot) error {
			upsertCalled = true
			return nil
		},
	}

	p1 := &mockProvider{name: "p1", fetchAll: func() ([]domain.ParkingLot, error) {
		return nil, errors.New("err1")
	}}
	p2 := &mockProvider{name: "p2", fetchAll: func() ([]domain.ParkingLot, error) {
		return nil, errors.New("err2")
	}}

	agg := New(repo, p1, p2)
	err := agg.Refresh()
	if err == nil {
		t.Fatal("expected an error when all providers fail")
	}
	if upsertCalled {
		t.Error("upsert should not be called when no lots were fetched")
	}
}

func TestRefresh_UpsertFails_ReturnsError(t *testing.T) {
	upsertErr := errors.New("db down")
	repo := &mockRepo{
		upsertFn: func(lots []domain.ParkingLot) error { return upsertErr },
	}

	p := &mockProvider{name: "p", fetchAll: func() ([]domain.ParkingLot, error) {
		return []domain.ParkingLot{lotA}, nil
	}}

	agg := New(repo, p)
	err := agg.Refresh()
	if err == nil {
		t.Fatal("expected error from upsert")
	}
	if !errors.Is(err, upsertErr) {
		t.Errorf("expected wrapped upsert error, got: %v", err)
	}
}

func TestRefresh_NoProviders_ReturnsError(t *testing.T) {
	repo := &mockRepo{
		upsertFn: func(lots []domain.ParkingLot) error { return nil },
	}
	agg := New(repo)
	if err := agg.Refresh(); err == nil {
		t.Error("expected error when there are no providers")
	}
}

func TestGetNearby_SortedByDistance(t *testing.T) {
	origin := domain.SearchParams{Lat: 52.0, Lng: 13.0, RadiusKm: 100}

	repo := &mockRepo{
		findNearby: func(params domain.SearchParams) ([]domain.ParkingLot, error) {
			return []domain.ParkingLot{lotC, lotB, lotA}, nil
		},
	}

	agg := New(repo)
	results, err := agg.GetNearby(origin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i := 1; i < len(results); i++ {
		if results[i].DistanceKm < results[i-1].DistanceKm {
			t.Errorf("results not sorted by distance: index %d (%f) < index %d (%f)",
				i, results[i].DistanceKm, i-1, results[i-1].DistanceKm)
		}
	}
}

func TestGetNearby_FirstResultIsClosest(t *testing.T) {
	origin := domain.SearchParams{Lat: 52.0, Lng: 13.0}
	repo := &mockRepo{
		findNearby: func(params domain.SearchParams) ([]domain.ParkingLot, error) {
			return []domain.ParkingLot{lotA, lotB}, nil
		},
	}

	agg := New(repo)
	results, err := agg.GetNearby(origin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].ID != "a" {
		t.Errorf("expected lot A (at origin) to be first, got %s", results[0].ID)
	}
	if results[0].DistanceKm > 0.001 {
		t.Errorf("expected near-zero distance for lot at origin, got %f", results[0].DistanceKm)
	}
}

func TestGetNearby_DistanceIsNonNegative(t *testing.T) {
	origin := domain.SearchParams{Lat: 48.8566, Lng: 2.3522}
	repo := &mockRepo{
		findNearby: func(params domain.SearchParams) ([]domain.ParkingLot, error) {
			return []domain.ParkingLot{lotA, lotB, lotC}, nil
		},
	}

	agg := New(repo)
	results, err := agg.GetNearby(origin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range results {
		if r.DistanceKm < 0 {
			t.Errorf("negative distance for lot %s: %f", r.ID, r.DistanceKm)
		}
	}
}

func TestGetNearby_RepoError_Propagated(t *testing.T) {
	repoErr := errors.New("query failed")
	repo := &mockRepo{
		findNearby: func(params domain.SearchParams) ([]domain.ParkingLot, error) {
			return nil, repoErr
		},
	}

	agg := New(repo)
	_, err := agg.GetNearby(domain.SearchParams{})
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected wrapped repo error, got: %v", err)
	}
}

func TestGetNearby_EmptyResult(t *testing.T) {
	repo := &mockRepo{
		findNearby: func(params domain.SearchParams) ([]domain.ParkingLot, error) {
			return []domain.ParkingLot{}, nil
		},
	}

	agg := New(repo)
	results, err := agg.GetNearby(domain.SearchParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestGetByID_ReturnsLot(t *testing.T) {
	want := &domain.ParkingLot{ID: "a", Name: "Lot A"}
	repo := &mockRepo{
		findByID: func(id string) (*domain.ParkingLot, error) {
			if id != "a" {
				t.Errorf("unexpected id: %s", id)
			}
			return want, nil
		},
	}

	agg := New(repo)
	got, err := agg.GetByID("a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected ID %s, got %s", want.ID, got.ID)
	}
}

func TestGetByID_RepoError_Propagated(t *testing.T) {
	notFound := errors.New("not found")
	repo := &mockRepo{
		findByID: func(id string) (*domain.ParkingLot, error) {
			return nil, notFound
		},
	}

	agg := New(repo)
	_, err := agg.GetByID("missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, notFound) {
		t.Errorf("expected wrapped not-found error, got: %v", err)
	}
}

func TestHaversine_SamePoint_IsZero(t *testing.T) {
	d := haversine(48.8566, 2.3522, 48.8566, 2.3522)
	if d != 0 {
		t.Errorf("expected 0 distance for same point, got %f", d)
	}
}

func TestHaversine_KnownDistance(t *testing.T) {
	// Berlin (52.5200, 13.4050) to Paris (48.8566, 2.3522) ≈ 878 km
	d := haversine(52.5200, 13.4050, 48.8566, 2.3522)
	const expected = 878.0
	const tolerancePercent = 0.5
	if math.Abs(d-expected)/expected*100 > tolerancePercent {
		t.Errorf("Berlin→Paris: expected ~%f km, got %f km", expected, d)
	}
}

func TestHaversine_IsSymmetric(t *testing.T) {
	d1 := haversine(52.5200, 13.4050, 48.8566, 2.3522)
	d2 := haversine(48.8566, 2.3522, 52.5200, 13.4050)
	if math.Abs(d1-d2) > 1e-9 {
		t.Errorf("haversine should be symmetric: %f != %f", d1, d2)
	}
}

func TestWithDistances_Length(t *testing.T) {
	lots := []domain.ParkingLot{lotA, lotB, lotC}
	result := withDistances(52.0, 13.0, lots)
	if len(result) != len(lots) {
		t.Errorf("expected %d results, got %d", len(lots), len(result))
	}
}

func TestWithDistances_PreservesLotData(t *testing.T) {
	lots := []domain.ParkingLot{lotA}
	result := withDistances(0, 0, lots)
	if result[0].ID != lotA.ID || result[0].Name != lotA.Name {
		t.Errorf("lot data not preserved: %+v", result[0])
	}
}

func TestStartRefreshLoop_CallsRefreshAndStopsOnCancel(t *testing.T) {
	var callCount atomic.Int32

	repo := &mockRepo{
		upsertFn: func(lots []domain.ParkingLot) error { return nil },
	}
	p := &mockProvider{name: "p", fetchAll: func() ([]domain.ParkingLot, error) {
		callCount.Add(1)
		return []domain.ParkingLot{lotA}, nil
	}}

	agg := New(repo, p)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		agg.StartRefreshLoop(ctx, 20*time.Millisecond)
		close(done)
	}()

	time.Sleep(80 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StartRefreshLoop did not stop after context cancellation")
	}

	if callCount.Load() < 2 {
		t.Errorf("expected at least 2 provider calls, got %d", callCount.Load())
	}
}

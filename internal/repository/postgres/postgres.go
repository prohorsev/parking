package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/parking/api/internal/domain"
)

func Connect(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		query, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := db.Exec(string(query)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
		log.Printf("migration applied: %s", filepath.Base(f))
	}
	return nil
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

const upsertSQL = `
INSERT INTO parking_lots
    (external_id, provider, name, address, lat, lng,
     total_spots, free_spots, price_per_hour, currency, updated_at)
VALUES
    (:external_id, :provider, :name, :address, :lat, :lng,
     :total_spots, :free_spots, :price_per_hour, :currency, :updated_at)
ON CONFLICT (external_id, provider) DO UPDATE SET
    name           = EXCLUDED.name,
    address        = EXCLUDED.address,
    lat            = EXCLUDED.lat,
    lng            = EXCLUDED.lng,
    total_spots    = EXCLUDED.total_spots,
    free_spots     = EXCLUDED.free_spots,
    price_per_hour = EXCLUDED.price_per_hour,
    currency       = EXCLUDED.currency,
    updated_at     = EXCLUDED.updated_at`

func (r *Repository) Upsert(lots []domain.ParkingLot) error {
	if len(lots) == 0 {
		return nil
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, lot := range lots {
		if _, err := tx.NamedExec(upsertSQL, lot); err != nil {
			return fmt.Errorf("upsert %s/%s: %w", lot.Provider, lot.ExternalID, err)
		}
	}
	return tx.Commit()
}

const selectCols = `
    id, external_id, provider, name, address, lat, lng,
    total_spots, free_spots, price_per_hour, currency, updated_at`

func (r *Repository) FindNearby(params domain.SearchParams) ([]domain.ParkingLot, error) {
	latDelta := params.RadiusKm / 111.0
	lngDelta := params.RadiusKm / (111.0 * math.Cos(params.Lat*math.Pi/180))

	q := fmt.Sprintf(`SELECT %s FROM parking_lots
		WHERE lat BETWEEN $1 AND $2
		  AND lng BETWEEN $3 AND $4
		ORDER BY updated_at DESC`, selectCols)

	var lots []domain.ParkingLot
	err := r.db.Select(&lots, q,
		params.Lat-latDelta, params.Lat+latDelta,
		params.Lng-lngDelta, params.Lng+lngDelta,
	)
	if err != nil {
		return nil, fmt.Errorf("find nearby: %w", err)
	}
	return lots, nil
}

func (r *Repository) FindByID(id string) (*domain.ParkingLot, error) {
	q := fmt.Sprintf(`SELECT %s FROM parking_lots WHERE id = $1`, selectCols)
	var lot domain.ParkingLot
	if err := r.db.Get(&lot, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find by id: %w", err)
	}
	return &lot, nil
}

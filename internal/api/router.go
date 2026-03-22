package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/parking/api/internal/aggregator"
)

func NewRouter(agg *aggregator.Aggregator) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := &handler{agg: agg}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/parking", func(r chi.Router) {
		r.Get("/", h.getParking)
		r.Get("/{id}", h.getParkingByID)
	})

	return r
}

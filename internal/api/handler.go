package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/parking/api/internal/aggregator"
	"github.com/parking/api/internal/domain"
)

type handler struct {
	agg *aggregator.Aggregator
}

type nearbyResponse struct {
	Data   any `json:"data"`
	Count  int `json:"count"`
	Params any `json:"params"`
}

type singleResponse struct {
	Data any `json:"data"`
}

type errResponse struct {
	Error string `json:"error"`
}

func (h *handler) getParking(w http.ResponseWriter, r *http.Request) {
	lat, err := queryFloat(r, "lat")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse{`"lat" is required and must be a valid number`})
		return
	}
	lng, err := queryFloat(r, "lng")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse{`"lng" is required and must be a valid number`})
		return
	}

	radius := 5.0
	if v := r.URL.Query().Get("radius"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			radius = parsed
		}
	}

	params := domain.SearchParams{Lat: lat, Lng: lng, RadiusKm: radius}
	lots, err := h.agg.GetNearby(params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResponse{err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, nearbyResponse{
		Data:  lots,
		Count: len(lots),
		Params: map[string]any{
			"lat":       lat,
			"lng":       lng,
			"radius_km": radius,
		},
	})
}

func (h *handler) getParkingByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errResponse{"id is required"})
		return
	}

	lot, err := h.agg.GetByID(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errResponse{err.Error()})
		return
	}
	if lot == nil {
		writeJSON(w, http.StatusNotFound, errResponse{"parking lot not found"})
		return
	}

	writeJSON(w, http.StatusOK, singleResponse{Data: lot})
}

func queryFloat(r *http.Request, key string) (float64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0, fmt.Errorf("missing query param %q", key)
	}
	return strconv.ParseFloat(v, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

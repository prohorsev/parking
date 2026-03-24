package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/parking/api/internal/aggregator"
	"github.com/parking/api/internal/domain"
	"github.com/parking/api/internal/queue"
)

type handler struct {
	agg *aggregator.Aggregator
	pub *queue.Publisher
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

type createParkingReq struct {
	ExternalID   string  `json:"external_id"`
	Provider     string  `json:"provider"`
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	TotalSpots   int     `json:"total_spots"`
	FreeSpots    int     `json:"free_spots"`
	PricePerHour float64 `json:"price_per_hour"`
	Currency     string  `json:"currency"`
}

func (h *handler) createParking(w http.ResponseWriter, r *http.Request) {
	var req createParkingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errResponse{"invalid JSON body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errResponse{`"name" is required`})
		return
	}
	if req.Lat == 0 || req.Lng == 0 {
		writeJSON(w, http.StatusBadRequest, errResponse{`"lat" and "lng" are required`})
		return
	}

	if req.Provider == "" {
		req.Provider = "manual"
	}
	if req.ExternalID == "" {
		req.ExternalID = randomID()
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}

	lot := domain.ParkingLot{
		ExternalID:   req.ExternalID,
		Provider:     req.Provider,
		Name:         req.Name,
		Address:      req.Address,
		Lat:          req.Lat,
		Lng:          req.Lng,
		TotalSpots:   req.TotalSpots,
		FreeSpots:    req.FreeSpots,
		PricePerHour: req.PricePerHour,
		Currency:     req.Currency,
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.pub.Publish(r.Context(), lot); err != nil {
		writeJSON(w, http.StatusInternalServerError, errResponse{"failed to enqueue: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      "queued",
		"external_id": lot.ExternalID,
		"provider":    lot.Provider,
	})
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

func randomID() string {
	b := make([]byte, 8)

	_, _ = rand.Read(b)

	return fmt.Sprintf("%x", b)
}

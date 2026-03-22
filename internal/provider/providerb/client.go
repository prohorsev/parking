package providerb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/parking/api/internal/domain"
)

const providerName = "provider_b"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Name() string { return providerName }

func (c *Client) FetchAll() ([]domain.ParkingLot, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/parking-zones")
	if err != nil {
		return nil, fmt.Errorf("provider B: GET /parking-zones: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider B: unexpected status %d", resp.StatusCode)
	}

	var raw []lotB
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("provider B: decode response: %w", err)
	}

	lots := make([]domain.ParkingLot, 0, len(raw))
	for _, r := range raw {
		lat, lng, err := parseGeo(r.Geo)
		if err != nil {
			fmt.Printf("provider B: skipping %q — %v\n", r.UID, err)
			continue
		}
		lots = append(lots, domain.ParkingLot{
			ExternalID:   r.UID,
			Provider:     providerName,
			Name:         r.Title,
			Address:      r.FullAddress,
			Lat:          lat,
			Lng:          lng,
			TotalSpots:   r.SpacesTotal,
			FreeSpots:    r.SpacesFree,
			PricePerHour: r.Price,
			Currency:     r.PriceCur,
			UpdatedAt:    time.Now().UTC(),
		})
	}
	return lots, nil
}

func parseGeo(geo string) (lat, lng float64, err error) {
	parts := strings.SplitN(geo, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid geo %q: expected \"lat,lng\"", geo)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lat in %q: %w", geo, err)
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid lng in %q: %w", geo, err)
	}
	return lat, lng, nil
}

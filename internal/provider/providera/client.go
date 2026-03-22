package providera

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/parking/api/internal/domain"
)

const providerName = "provider_a"

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
	resp, err := c.httpClient.Get(c.baseURL + "/lots")
	if err != nil {
		return nil, fmt.Errorf("provider A: GET /lots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider A: unexpected status %d", resp.StatusCode)
	}

	var raw []lotA
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("provider A: decode response: %w", err)
	}

	lots := make([]domain.ParkingLot, 0, len(raw))
	for _, r := range raw {
		lots = append(lots, domain.ParkingLot{
			ExternalID:   r.ParkingID,
			Provider:     providerName,
			Name:         r.Name,
			Address:      r.Street,
			Lat:          r.Location.Lat,
			Lng:          r.Location.Lon,
			TotalSpots:   r.TotalCap,
			FreeSpots:    r.Available,
			PricePerHour: r.HourlyRate,
			Currency:     r.CurrencyCode,
			UpdatedAt:    time.Now().UTC(),
		})
	}
	return lots, nil
}

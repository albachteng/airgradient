package client

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

const (
	chicagoLatMin = 41.644
	chicagoLatMax = 42.023
	chicagoLonMin = -87.940
	chicagoLonMax = -87.524
)

type Measure struct {
	LocationID   int      `json:"locationId"`
	LocationName string   `json:"locationName"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Offline      bool     `json:"offline"`
	Pm01         *float64 `json:"pm01"`
	Pm02         *float64 `json:"pm02"`
	Pm10         *float64 `json:"pm10"`
	Pm003Count   *int     `json:"pm003Count"`
	Atmp         *float64 `json:"atmp"`
	Rhum         *int     `json:"rhum"`
	Rco2         *int     `json:"rco2"`
	Tvoc         *float64 `json:"tvoc"`
	Wifi         *int     `json:"wifi"`
	Timestamp    string   `json:"timestamp"`
	TvocIndex    *int     `json:"tvocIndex"`
	NoxIndex     *int     `json:"noxIndex"`
	Heatindex    *int     `json:"heatindex"`
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *Client) get(path string, auth bool) (*http.Response, error) {
	url := c.baseURL + path
	if auth {
		url += "?token=" + c.token
	}
	return c.httpClient.Get(url)
}

func (c *Client) Ping() error {
	resp, err := c.get("/public/api/v1/ping", false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ping: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetOwnMeasure(locationID string) (*Measure, error) {
	resp, err := c.get("/public/api/v1/locations/"+locationID+"/measures/current", true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: check your API token")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetOwnMeasure: unexpected status %d", resp.StatusCode)
	}
	var m Measure
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("GetOwnMeasure: decode: %w", err)
	}
	return &m, nil
}

func (c *Client) GetWorldMeasures() ([]Measure, error) {
	resp, err := c.get("/public/api/v1/world/locations/measures/current", false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetWorldMeasures: unexpected status %d", resp.StatusCode)
	}
	var measures []Measure
	if err := json.NewDecoder(resp.Body).Decode(&measures); err != nil {
		return nil, fmt.Errorf("GetWorldMeasures: decode: %w", err)
	}
	return measures, nil
}

func FilterChicago(measures []Measure) []Measure {
	var result []Measure
	for _, m := range measures {
		if m.Latitude >= chicagoLatMin && m.Latitude <= chicagoLatMax &&
			m.Longitude >= chicagoLonMin && m.Longitude <= chicagoLonMax {
			result = append(result, m)
		}
	}
	return result
}

// SampleN returns a random sample of n measures. If n >= len(measures), all are returned.
func SampleN(measures []Measure, n int) []Measure {
	if n >= len(measures) {
		return measures
	}
	perm := rand.Perm(len(measures))
	result := make([]Measure, n)
	for i := 0; i < n; i++ {
		result[i] = measures[perm[i]]
	}
	return result
}

package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/albachteng/airgradient/internal/client"
)

func TestPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/api/v1/ping" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	if err := c.Ping(); err != nil {
		t.Fatalf("Ping() returned error: %v", err)
	}
}

func TestPingServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	if err := c.Ping(); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestGetOwnMeasure(t *testing.T) {
	want := client.Measure{
		LocationID:   42,
		LocationName: "Living Room",
		Pm02:         floatPtr(5.3),
		Rco2:         intPtr(812),
		Atmp:         floatPtr(21.4),
		Rhum:         intPtr(45),
		Timestamp:    "2026-08-15T12:00:00.000Z",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/api/v1/locations/42/measures/current" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("token") != "test-token" {
			t.Errorf("missing or wrong token: %s", r.URL.Query().Get("token"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	got, err := c.GetOwnMeasure("42")
	if err != nil {
		t.Fatalf("GetOwnMeasure() error: %v", err)
	}
	if got.LocationID != want.LocationID {
		t.Errorf("LocationID: got %d, want %d", got.LocationID, want.LocationID)
	}
	if got.Rco2 == nil || *got.Rco2 != *want.Rco2 {
		t.Errorf("Rco2: got %v, want %v", got.Rco2, want.Rco2)
	}
}

func TestGetOwnMeasureUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "bad-token")
	_, err := c.GetOwnMeasure("42")
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
}

func TestGetWorldMeasures(t *testing.T) {
	measures := []client.Measure{
		{LocationID: 1, LocationName: "Chicago Monitor 1", Latitude: 41.85, Longitude: -87.65},
		{LocationID: 2, LocationName: "Bangkok Monitor", Latitude: 13.75, Longitude: 100.52},
		{LocationID: 3, LocationName: "Chicago Monitor 2", Latitude: 41.90, Longitude: -87.63},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/api/v1/world/locations/measures/current" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(measures)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	got, err := c.GetWorldMeasures()
	if err != nil {
		t.Fatalf("GetWorldMeasures() error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 measures, got %d", len(got))
	}
}

func TestFilterChicago(t *testing.T) {
	measures := []client.Measure{
		{LocationID: 1, Latitude: 41.85, Longitude: -87.65}, // Chicago
		{LocationID: 2, Latitude: 13.75, Longitude: 100.52}, // Bangkok
		{LocationID: 3, Latitude: 41.90, Longitude: -87.63}, // Chicago
		{LocationID: 4, Latitude: 42.50, Longitude: -87.80}, // Outside bbox
		{LocationID: 5, Latitude: 41.70, Longitude: -88.20}, // Outside bbox
	}

	got := client.FilterChicago(measures)
	if len(got) != 2 {
		t.Errorf("expected 2 Chicago measures, got %d", len(got))
	}
	for _, m := range got {
		if m.LocationID != 1 && m.LocationID != 3 {
			t.Errorf("unexpected location in Chicago filter: %d", m.LocationID)
		}
	}
}

func TestSampleN(t *testing.T) {
	measures := make([]client.Measure, 20)
	for i := range measures {
		measures[i] = client.Measure{LocationID: i + 1}
	}

	got := client.SampleN(measures, 5)
	if len(got) != 5 {
		t.Errorf("SampleN(20, 5): expected 5, got %d", len(got))
	}

	// SampleN with n >= len should return all
	got = client.SampleN(measures, 100)
	if len(got) != 20 {
		t.Errorf("SampleN(20, 100): expected 20, got %d", len(got))
	}
}

// helpers

func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

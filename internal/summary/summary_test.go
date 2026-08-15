package summary_test

import (
	"strings"
	"testing"
	"time"

	"github.com/albachteng/airgradient/internal/db"
	"github.com/albachteng/airgradient/internal/summary"
)

type fakeStore struct {
	own     []db.PlacementStats
	chicago []db.LocationStats
	fromGot time.Time
	toGot   time.Time
}

func (f *fakeStore) OwnStatsByPlacement(from, to time.Time) ([]db.PlacementStats, error) {
	f.fromGot, f.toGot = from, to
	return f.own, nil
}

func (f *fakeStore) ChicagoStatsByLocation(from, to time.Time) ([]db.LocationStats, error) {
	return f.chicago, nil
}

var (
	pm25     = 5.3
	co2      = 820.0
	tempC    = 21.4
	humidity = 48.0
)

func TestGeneratePopulatesSummary(t *testing.T) {
	store := &fakeStore{
		own: []db.PlacementStats{
			{Label: "bedroom", ReadingCount: 168, AvgPm25: &pm25, AvgCo2: &co2},
		},
		chicago: []db.LocationStats{
			{LocationID: 100, LocationName: "Lincoln Park", ReadingCount: 42, AvgPm25: floatPtr(4.1)},
		},
	}

	from := time.Now().UTC().Add(-7 * 24 * time.Hour)
	to := time.Now().UTC()

	s, err := summary.Generate(store, from, to)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !s.PeriodStart.Equal(from) {
		t.Errorf("PeriodStart: got %v, want %v", s.PeriodStart, from)
	}
	if !s.PeriodEnd.Equal(to) {
		t.Errorf("PeriodEnd: got %v, want %v", s.PeriodEnd, to)
	}
	if len(s.OwnStats) != 1 {
		t.Fatalf("OwnStats: expected 1, got %d", len(s.OwnStats))
	}
	if len(s.ChicagoStats) != 1 {
		t.Fatalf("ChicagoStats: expected 1, got %d", len(s.ChicagoStats))
	}
}

func TestGeneratePassesTimeRangeToStore(t *testing.T) {
	store := &fakeStore{}
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	summary.Generate(store, from, to)

	if !store.fromGot.Equal(from) {
		t.Errorf("from passed to store: got %v, want %v", store.fromGot, from)
	}
	if !store.toGot.Equal(to) {
		t.Errorf("to passed to store: got %v, want %v", store.toGot, to)
	}
}

func TestRenderHTMLContainsPeriod(t *testing.T) {
	s := summary.Summary{
		PeriodStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC),
	}
	html, err := summary.RenderHTML(s)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(html, "Aug 1") {
		t.Errorf("expected period start in HTML, got:\n%s", html)
	}
	if !strings.Contains(html, "Aug 8") {
		t.Errorf("expected period end in HTML, got:\n%s", html)
	}
}

func TestRenderHTMLContainsOwnStats(t *testing.T) {
	s := summary.Summary{
		PeriodStart: time.Now().UTC().Add(-7 * 24 * time.Hour),
		PeriodEnd:   time.Now().UTC(),
		OwnStats: []db.PlacementStats{
			{Label: "bedroom", ReadingCount: 168, AvgPm25: &pm25, AvgCo2: &co2, AvgTempC: &tempC, AvgHumidity: &humidity},
		},
	}
	html, err := summary.RenderHTML(s)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(html, "bedroom") {
		t.Errorf("expected placement label in HTML")
	}
	if !strings.Contains(html, "5.3") {
		t.Errorf("expected PM2.5 value in HTML")
	}
	if !strings.Contains(html, "820") {
		t.Errorf("expected CO2 value in HTML")
	}
	if !strings.Contains(html, "168") {
		t.Errorf("expected reading count in HTML")
	}
}

func TestRenderHTMLContainsChicagoStats(t *testing.T) {
	s := summary.Summary{
		PeriodStart: time.Now().UTC().Add(-7 * 24 * time.Hour),
		PeriodEnd:   time.Now().UTC(),
		ChicagoStats: []db.LocationStats{
			{LocationID: 100, LocationName: "Lincoln Park", ReadingCount: 42, AvgPm25: floatPtr(4.1)},
			{LocationID: 101, LocationName: "Hyde Park", ReadingCount: 38, AvgPm25: floatPtr(3.7)},
		},
	}
	html, err := summary.RenderHTML(s)
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	if !strings.Contains(html, "Lincoln Park") {
		t.Errorf("expected location name in HTML")
	}
	if !strings.Contains(html, "Hyde Park") {
		t.Errorf("expected location name in HTML")
	}
	if !strings.Contains(html, "4.1") {
		t.Errorf("expected PM2.5 value in HTML")
	}
}

func TestRenderHTMLHandlesNilMetrics(t *testing.T) {
	s := summary.Summary{
		PeriodStart: time.Now().UTC().Add(-7 * 24 * time.Hour),
		PeriodEnd:   time.Now().UTC(),
		OwnStats: []db.PlacementStats{
			{Label: "attic", ReadingCount: 5}, // all avg fields nil
		},
		ChicagoStats: []db.LocationStats{
			{LocationID: 99, LocationName: "Unknown Station", ReadingCount: 2},
		},
	}
	// Should not panic or error
	html, err := summary.RenderHTML(s)
	if err != nil {
		t.Fatalf("RenderHTML with nil metrics: %v", err)
	}
	if !strings.Contains(html, "attic") {
		t.Errorf("expected placement label even with nil metrics")
	}
	// Nil fields should render as a dash rather than "<nil>"
	if strings.Contains(html, "<nil>") {
		t.Errorf("nil metric rendered as <nil> in HTML:\n%s", html)
	}
}

func floatPtr(f float64) *float64 { return &f }

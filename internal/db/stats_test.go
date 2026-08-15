package db_test

import (
	"math"
	"testing"
	"time"

	"github.com/albachteng/airgradient/internal/db"
)

func TestOwnStatsByPlacement(t *testing.T) {
	d := tempDB(t)
	now := time.Now().UTC()

	d.SetPlacement("bedroom", "")
	bedroom, _ := d.CurrentPlacement()

	pm25s := []float64{4.0, 6.0, 8.0}
	for i, v := range pm25s {
		v := v
		co2 := 800 + i*10
		d.SaveReading(db.Reading{
			LocationID:  42,
			Source:      "own",
			Timestamp:   now.Add(-time.Duration(i+1) * time.Minute).Format(time.RFC3339),
			PlacementID: &bedroom.ID,
			Pm02:        &v,
			Rco2:        intPtr(co2),
			Atmp:        floatPtr(21.0),
			Rhum:        intPtr(50),
		})
	}

	stats, err := d.OwnStatsByPlacement(now.Add(-time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("OwnStatsByPlacement: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 placement group, got %d", len(stats))
	}
	s := stats[0]
	if s.Label != "bedroom" {
		t.Errorf("Label: got %q, want \"bedroom\"", s.Label)
	}
	if s.ReadingCount != 3 {
		t.Errorf("ReadingCount: got %d, want 3", s.ReadingCount)
	}
	// avg PM2.5: (4+6+8)/3 = 6.0
	if s.AvgPm25 == nil || math.Abs(*s.AvgPm25-6.0) > 0.01 {
		t.Errorf("AvgPm25: got %v, want ~6.0", s.AvgPm25)
	}
	if s.AvgCo2 == nil || math.Abs(*s.AvgCo2-810.0) > 0.01 {
		t.Errorf("AvgCo2: got %v, want ~810.0", s.AvgCo2)
	}
}

func TestOwnStatsByMultiplePlacements(t *testing.T) {
	d := tempDB(t)
	now := time.Now().UTC()

	d.SetPlacement("bedroom", "")
	bedroom, _ := d.CurrentPlacement()
	for i := 0; i < 2; i++ {
		v := 5.0
		d.SaveReading(db.Reading{
			LocationID:  42,
			Source:      "own",
			Timestamp:   now.Add(-time.Duration(i+1) * time.Minute).Format(time.RFC3339),
			PlacementID: &bedroom.ID,
			Pm02:        &v,
		})
	}

	d.SetPlacement("kitchen", "")
	kitchen, _ := d.CurrentPlacement()
	for i := 0; i < 3; i++ {
		v := 2.0
		d.SaveReading(db.Reading{
			LocationID:  42,
			Source:      "own",
			Timestamp:   now.Add(-time.Duration(i+1) * time.Minute).Format(time.RFC3339),
			PlacementID: &kitchen.ID,
			Pm02:        &v,
		})
	}

	stats, err := d.OwnStatsByPlacement(now.Add(-time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("OwnStatsByPlacement: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 placement groups, got %d", len(stats))
	}
	counts := map[string]int{}
	for _, s := range stats {
		counts[s.Label] = s.ReadingCount
	}
	if counts["bedroom"] != 2 {
		t.Errorf("bedroom: expected 2 readings, got %d", counts["bedroom"])
	}
	if counts["kitchen"] != 3 {
		t.Errorf("kitchen: expected 3 readings, got %d", counts["kitchen"])
	}
}

func TestOwnStatsUnplacedReadings(t *testing.T) {
	d := tempDB(t)
	now := time.Now().UTC()

	// Reading with no placement set
	v := 7.0
	d.SaveReading(db.Reading{
		LocationID: 42,
		Source:     "own",
		Timestamp:  now.Add(-time.Minute).Format(time.RFC3339),
		Pm02:       &v,
	})

	stats, err := d.OwnStatsByPlacement(now.Add(-time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("OwnStatsByPlacement: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 group for unplaced readings, got %d", len(stats))
	}
	if stats[0].Label != "unplaced" {
		t.Errorf("Label: got %q, want \"unplaced\"", stats[0].Label)
	}
}

func TestOwnStatsByPlacementTimeFilter(t *testing.T) {
	d := tempDB(t)
	now := time.Now().UTC()

	d.SetPlacement("office", "")
	office, _ := d.CurrentPlacement()

	v := 5.0
	// One reading in window, one outside
	d.SaveReading(db.Reading{
		LocationID:  42,
		Source:      "own",
		Timestamp:   now.Add(-30 * time.Minute).Format(time.RFC3339),
		PlacementID: &office.ID,
		Pm02:        &v,
	})
	d.SaveReading(db.Reading{
		LocationID:  42,
		Source:      "own",
		Timestamp:   now.Add(-3 * time.Hour).Format(time.RFC3339),
		PlacementID: &office.ID,
		Pm02:        &v,
	})

	stats, err := d.OwnStatsByPlacement(now.Add(-time.Hour), now)
	if err != nil {
		t.Fatalf("OwnStatsByPlacement: %v", err)
	}
	if len(stats) != 1 || stats[0].ReadingCount != 1 {
		t.Errorf("expected 1 reading in window, got stats: %+v", stats)
	}
}

func TestChicagoStatsByLocation(t *testing.T) {
	d := tempDB(t)
	now := time.Now().UTC()

	locApm25 := []float64{3.0, 5.0}
	for i, v := range locApm25 {
		v := v
		d.SaveReading(db.Reading{
			LocationID:   100,
			LocationName: strPtr("Lincoln Park"),
			Source:       "chicago",
			Timestamp:    now.Add(-time.Duration(i+1) * time.Minute).Format(time.RFC3339),
			Pm02:         &v,
			Rco2:         intPtr(420),
		})
	}

	locBpm25 := 8.0
	d.SaveReading(db.Reading{
		LocationID:   101,
		LocationName: strPtr("Hyde Park"),
		Source:       "chicago",
		Timestamp:    now.Add(-time.Minute).Format(time.RFC3339),
		Pm02:         &locBpm25,
	})

	stats, err := d.ChicagoStatsByLocation(now.Add(-time.Hour), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ChicagoStatsByLocation: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 location groups, got %d", len(stats))
	}

	byID := map[int]db.LocationStats{}
	for _, s := range stats {
		byID[s.LocationID] = s
	}

	lpk := byID[100]
	if lpk.LocationName != "Lincoln Park" {
		t.Errorf("LocationName: got %q, want \"Lincoln Park\"", lpk.LocationName)
	}
	if lpk.ReadingCount != 2 {
		t.Errorf("ReadingCount: got %d, want 2", lpk.ReadingCount)
	}
	// avg PM2.5: (3+5)/2 = 4.0
	if lpk.AvgPm25 == nil || math.Abs(*lpk.AvgPm25-4.0) > 0.01 {
		t.Errorf("AvgPm25: got %v, want ~4.0", lpk.AvgPm25)
	}

	hp := byID[101]
	if hp.ReadingCount != 1 {
		t.Errorf("Hyde Park ReadingCount: got %d, want 1", hp.ReadingCount)
	}
}

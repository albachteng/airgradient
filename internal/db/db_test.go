package db_test

import (
	"os"
	"testing"
	"time"

	"github.com/albachteng/airgradient/internal/db"
)

func tempDB(t *testing.T) *db.DB {
	t.Helper()
	f, err := os.CreateTemp("", "airgradient-test-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	d, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenCreatesSchema(t *testing.T) {
	d := tempDB(t)
	if d == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestSaveAndRetrieveReading(t *testing.T) {
	d := tempDB(t)

	r := db.Reading{
		LocationID:   42,
		LocationName: strPtr("Living Room"),
		Source:       "own",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Pm02:         floatPtr(5.3),
		Rco2:         intPtr(812),
		Atmp:         floatPtr(21.4),
		Rhum:         intPtr(45),
	}

	if err := d.SaveReading(r); err != nil {
		t.Fatalf("SaveReading: %v", err)
	}

	since := time.Now().UTC().Add(-time.Minute)
	readings, err := d.ReadingsSince(since)
	if err != nil {
		t.Fatalf("ReadingsSince: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(readings))
	}
	if readings[0].LocationID != 42 {
		t.Errorf("LocationID: got %d, want 42", readings[0].LocationID)
	}
	if readings[0].Rco2 == nil || *readings[0].Rco2 != 812 {
		t.Errorf("Rco2: got %v, want 812", readings[0].Rco2)
	}
}

func TestReadingsSinceFilters(t *testing.T) {
	d := tempDB(t)

	old := db.Reading{
		LocationID: 1,
		Source:     "own",
		Timestamp:  time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	}
	recent := db.Reading{
		LocationID: 2,
		Source:     "chicago",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	d.SaveReading(old)
	d.SaveReading(recent)

	readings, err := d.ReadingsSince(time.Now().UTC().Add(-30 * time.Minute))
	if err != nil {
		t.Fatalf("ReadingsSince: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("expected 1 recent reading, got %d", len(readings))
	}
	if readings[0].LocationID != 2 {
		t.Errorf("expected location 2, got %d", readings[0].LocationID)
	}
}

func TestSetAndCurrentPlacement(t *testing.T) {
	d := tempDB(t)

	if err := d.SetPlacement("bedroom", "north-facing corner"); err != nil {
		t.Fatalf("SetPlacement: %v", err)
	}

	p, err := d.CurrentPlacement()
	if err != nil {
		t.Fatalf("CurrentPlacement: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil placement")
	}
	if p.Label != "bedroom" {
		t.Errorf("Label: got %q, want %q", p.Label, "bedroom")
	}
	if p.Description != "north-facing corner" {
		t.Errorf("Description: got %q, want %q", p.Description, "north-facing corner")
	}
	if p.EndedAt != nil {
		t.Errorf("EndedAt should be nil for current placement")
	}
}

func TestSetPlacementEndsCurrentOne(t *testing.T) {
	d := tempDB(t)

	d.SetPlacement("bedroom", "")
	d.SetPlacement("kitchen", "")

	p, err := d.CurrentPlacement()
	if err != nil {
		t.Fatalf("CurrentPlacement: %v", err)
	}
	if p.Label != "kitchen" {
		t.Errorf("expected current placement to be kitchen, got %q", p.Label)
	}

	all, err := d.ListPlacements()
	if err != nil {
		t.Fatalf("ListPlacements: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 placements, got %d", len(all))
	}

	var bedroom *db.Placement
	for i := range all {
		if all[i].Label == "bedroom" {
			bedroom = &all[i]
		}
	}
	if bedroom == nil {
		t.Fatal("bedroom placement not found")
	}
	if bedroom.EndedAt == nil {
		t.Error("expected bedroom EndedAt to be set after switching to kitchen")
	}
}

func TestCurrentPlacementNoneSet(t *testing.T) {
	d := tempDB(t)

	p, err := d.CurrentPlacement()
	if err != nil {
		t.Fatalf("CurrentPlacement: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil placement when none set, got %+v", p)
	}
}

func TestSaveReadingWithPlacement(t *testing.T) {
	d := tempDB(t)

	d.SetPlacement("living room", "")
	p, _ := d.CurrentPlacement()

	r := db.Reading{
		LocationID:  42,
		Source:      "own",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		PlacementID: &p.ID,
	}

	if err := d.SaveReading(r); err != nil {
		t.Fatalf("SaveReading: %v", err)
	}

	readings, _ := d.ReadingsSince(time.Now().UTC().Add(-time.Minute))
	if readings[0].PlacementID == nil || *readings[0].PlacementID != p.ID {
		t.Errorf("expected placement ID %d on reading", p.ID)
	}
}

// helpers

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

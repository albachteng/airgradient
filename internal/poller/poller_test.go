package poller_test

import (
	"context"
	"testing"
	"time"

	"github.com/albachteng/airgradient/internal/client"
	"github.com/albachteng/airgradient/internal/poller"
)

// fakeStore records calls to SaveReading so tests can assert on them.
type fakeStore struct {
	saved []poller.Reading
}

func (f *fakeStore) SaveReading(r poller.Reading) error {
	f.saved = append(f.saved, r)
	return nil
}

func (f *fakeStore) CurrentPlacement() (*poller.Placement, error) {
	return &poller.Placement{ID: 1, Label: "test-room"}, nil
}

// fakeClient returns deterministic data without hitting the network.
type fakeClient struct {
	ownMeasure    *client.Measure
	worldMeasures []client.Measure
	pingErr       error
}

func (fc *fakeClient) Ping() error { return fc.pingErr }

func (fc *fakeClient) GetOwnMeasure(_ string) (*client.Measure, error) {
	return fc.ownMeasure, nil
}

func (fc *fakeClient) GetWorldMeasures() ([]client.Measure, error) {
	return fc.worldMeasures, nil
}

func TestPollerTicksAndSavesReadings(t *testing.T) {
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{
			LocationID:   42,
			LocationName: "Home",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
		worldMeasures: []client.Measure{
			{LocationID: 100, Latitude: 41.85, Longitude: -87.65},
			{LocationID: 101, Latitude: 41.90, Longitude: -87.63},
		},
	}

	cfg := poller.Config{
		LocationID:         "42",
		ChicagoSampleSize:  2,
	}

	p := poller.New(fc, store, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Tick immediately once, then stop.
	p.RunOnce(ctx)

	// own reading + 2 chicago = 3 total
	if len(store.saved) != 3 {
		t.Errorf("expected 3 saved readings, got %d", len(store.saved))
	}

	var ownCount, chicagoCount int
	for _, r := range store.saved {
		switch r.Source {
		case "own":
			ownCount++
		case "chicago":
			chicagoCount++
		}
	}
	if ownCount != 1 {
		t.Errorf("expected 1 own reading, got %d", ownCount)
	}
	if chicagoCount != 2 {
		t.Errorf("expected 2 chicago readings, got %d", chicagoCount)
	}
}

func TestPollerAttachesCurrentPlacement(t *testing.T) {
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{
			LocationID: 42,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
		},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 0}
	p := poller.New(fc, store, cfg)
	p.RunOnce(context.Background())

	if len(store.saved) != 1 {
		t.Fatalf("expected 1 reading, got %d", len(store.saved))
	}
	r := store.saved[0]
	if r.PlacementID == nil || *r.PlacementID != 1 {
		t.Errorf("expected placement ID 1 on own reading, got %v", r.PlacementID)
	}
}

func TestPollerRunRespectsCancellation(t *testing.T) {
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{LocationID: 42, Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 0}
	p := poller.New(fc, store, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	done := make(chan struct{})
	go func() {
		p.Run(ctx, 100*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// good — poller exited promptly
	case <-time.After(500 * time.Millisecond):
		t.Error("Run() did not exit after context cancellation")
	}
}

package poller_test

import (
	"context"
	"fmt"
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
	ownErr        error
	worldErr      error
}

func (fc *fakeClient) Ping() error { return fc.pingErr }

func (fc *fakeClient) GetOwnMeasure(_ string) (*client.Measure, error) {
	return fc.ownMeasure, fc.ownErr
}

func (fc *fakeClient) GetWorldMeasures() ([]client.Measure, error) {
	return fc.worldMeasures, fc.worldErr
}

// logCapture records Log calls for assertions.
type logCapture struct {
	entries []logEntry
}

type logEntry struct {
	msg    string
	fields map[string]any
}

func (lc *logCapture) Log(msg string, fields map[string]any) {
	lc.entries = append(lc.entries, logEntry{msg: msg, fields: fields})
}

func (lc *logCapture) find(msg string) *logEntry {
	for i := range lc.entries {
		if lc.entries[i].msg == msg {
			return &lc.entries[i]
		}
	}
	return nil
}

func (lc *logCapture) count(msg string) int {
	n := 0
	for _, e := range lc.entries {
		if e.msg == msg {
			n++
		}
	}
	return n
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
		LocationID:        "42",
		ChicagoSampleSize: 2,
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

func TestPollerLogsOwnReadingFields(t *testing.T) {
	pm25, co2, temp, hum := 4.2, 850, 21.5, 48
	lc := &logCapture{}
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{
			LocationID:   42,
			LocationName: "Home",
			Timestamp:    "2026-08-15T12:00:00Z",
			Pm02:         floatPtr(float64(pm25)),
			Rco2:         &co2,
			Atmp:         floatPtr(float64(temp)),
			Rhum:         &hum,
		},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 0, Log: lc.Log}
	poller.New(fc, store, cfg).RunOnce(context.Background())

	entry := lc.find("own reading")
	if entry == nil {
		t.Fatalf("expected \"own reading\" log entry, got entries: %+v", lc.entries)
	}
	assertField(t, entry.fields, "location_id", 42)
	assertField(t, entry.fields, "location_name", "Home")
	assertField(t, entry.fields, "timestamp", "2026-08-15T12:00:00Z")
	assertField(t, entry.fields, "pm2.5", pm25)
	assertField(t, entry.fields, "co2", co2)
	assertField(t, entry.fields, "temp_c", temp)
	assertField(t, entry.fields, "humidity", hum)
}

func TestPollerLogsPlacementOnOwnReading(t *testing.T) {
	lc := &logCapture{}
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{LocationID: 42, Timestamp: "2026-08-15T12:00:00Z"},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 0, Log: lc.Log}
	poller.New(fc, store, cfg).RunOnce(context.Background())

	entry := lc.find("own reading")
	if entry == nil {
		t.Fatalf("expected \"own reading\" log entry")
	}
	// fakeStore always returns placement label "test-room"
	assertField(t, entry.fields, "placement", "test-room")
}

func TestPollerLogsChicagoReadings(t *testing.T) {
	lc := &logCapture{}
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{LocationID: 42, Timestamp: "2026-08-15T12:00:00Z"},
		worldMeasures: []client.Measure{
			{LocationID: 100, LocationName: "Lincoln Park", Latitude: 41.92, Longitude: -87.64},
			{LocationID: 101, LocationName: "Hyde Park", Latitude: 41.80, Longitude: -87.59},
			{LocationID: 102, LocationName: "Wicker Park", Latitude: 41.91, Longitude: -87.68},
		},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 2, Log: lc.Log}
	poller.New(fc, store, cfg).RunOnce(context.Background())

	if n := lc.count("chicago reading"); n != 2 {
		t.Errorf("expected 2 \"chicago reading\" log entries, got %d", n)
	}
}

func TestPollerLogsCycleComplete(t *testing.T) {
	lc := &logCapture{}
	store := &fakeStore{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{LocationID: 42, Timestamp: "2026-08-15T12:00:00Z"},
		worldMeasures: []client.Measure{
			{LocationID: 100, Latitude: 41.85, Longitude: -87.65},
			{LocationID: 101, Latitude: 41.90, Longitude: -87.63},
			{LocationID: 102, Latitude: 41.78, Longitude: -87.60},
		},
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 2, Log: lc.Log}
	poller.New(fc, store, cfg).RunOnce(context.Background())

	entry := lc.find("poll cycle complete")
	if entry == nil {
		t.Fatalf("expected \"poll cycle complete\" log entry, got entries: %+v", lc.entries)
	}
	assertField(t, entry.fields, "chicago_total", 3)
	assertField(t, entry.fields, "chicago_sample", 2)
}

func TestPollerLogsOwnFetchError(t *testing.T) {
	lc := &logCapture{}
	fc := &fakeClient{ownErr: fmt.Errorf("unauthorized")}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 0, Log: lc.Log}
	poller.New(fc, &fakeStore{}, cfg).RunOnce(context.Background())

	entry := lc.find("own measure fetch failed")
	if entry == nil {
		t.Fatalf("expected \"own measure fetch failed\" log entry, got: %+v", lc.entries)
	}
	if entry.fields["error"] == "" {
		t.Error("expected non-empty error field")
	}
}

func TestPollerLogsWorldFetchError(t *testing.T) {
	lc := &logCapture{}
	fc := &fakeClient{
		ownMeasure: &client.Measure{LocationID: 42, Timestamp: "2026-08-15T12:00:00Z"},
		worldErr:   fmt.Errorf("service unavailable"),
	}

	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 5, Log: lc.Log}
	poller.New(fc, &fakeStore{}, cfg).RunOnce(context.Background())

	entry := lc.find("world measures fetch failed")
	if entry == nil {
		t.Fatalf("expected \"world measures fetch failed\" log entry, got: %+v", lc.entries)
	}
	if entry.fields["error"] == "" {
		t.Error("expected non-empty error field")
	}
}

func TestPollerNilLogIsNoOp(t *testing.T) {
	fc := &fakeClient{
		ownMeasure:    &client.Measure{LocationID: 42, Timestamp: "2026-08-15T12:00:00Z"},
		worldMeasures: []client.Measure{{LocationID: 100, Latitude: 41.85, Longitude: -87.65}},
	}
	cfg := poller.Config{LocationID: "42", ChicagoSampleSize: 1} // Log is nil
	// Should not panic
	poller.New(fc, &fakeStore{}, cfg).RunOnce(context.Background())
}

// assertField checks that fields[key] equals want, handling the int/float64 ambiguity
// that arises when map[string]any values come from JSON or mixed numeric types.
func assertField(t *testing.T, fields map[string]any, key string, want any) {
	t.Helper()
	got, ok := fields[key]
	if !ok {
		t.Errorf("field %q missing; fields: %+v", key, fields)
		return
	}
	// Normalise both sides to float64 for numeric comparisons so that int
	// literals in tests match float64 values stored via interface{}.
	if !equalValues(got, want) {
		t.Errorf("field %q: got %v (%T), want %v (%T)", key, got, got, want, want)
	}
}

func equalValues(a, b any) bool {
	af, aOK := toFloat64(a)
	bf, bOK := toFloat64(b)
	if aOK && bOK {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	}
	return 0, false
}

func floatPtr(f float64) *float64 { return &f }

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

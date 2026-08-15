package poller

import (
	"context"
	"time"

	"github.com/albachteng/airgradient/internal/client"
)

type Reading struct {
	LocationID   int
	LocationName *string
	PlacementID  *int
	Source       string
	Timestamp    string
	Pm01         *float64
	Pm02         *float64
	Pm10         *float64
	Pm003Count   *int
	Atmp         *float64
	Rhum         *int
	Rco2         *int
	Tvoc         *float64
	TvocIndex    *int
	NoxIndex     *int
	HeatIndex    *int
	Wifi         *int
}

type Placement struct {
	ID    int
	Label string
}

type Store interface {
	SaveReading(Reading) error
	CurrentPlacement() (*Placement, error)
}

type APIClient interface {
	Ping() error
	GetOwnMeasure(locationID string) (*client.Measure, error)
	GetWorldMeasures() ([]client.Measure, error)
}

type Config struct {
	LocationID        string
	ChicagoSampleSize int
	Log               func(msg string, fields map[string]any) // optional; nil = silent
}

type Poller struct {
	client APIClient
	store  Store
	cfg    Config
}

func New(c APIClient, s Store, cfg Config) *Poller {
	return &Poller{client: c, store: s, cfg: cfg}
}

func (p *Poller) log(msg string, fields map[string]any) {
	if p.cfg.Log != nil {
		p.cfg.Log(msg, fields)
	}
}

func (p *Poller) RunOnce(ctx context.Context) {
	placement, _ := p.store.CurrentPlacement()

	if own, err := p.client.GetOwnMeasure(p.cfg.LocationID); err != nil {
		p.log("own measure fetch failed", map[string]any{"error": err.Error()})
	} else {
		r := measureToReading(own, "own")
		if placement != nil {
			r.PlacementID = &placement.ID
		}
		p.store.SaveReading(r)
		p.log("own reading", readingFields(r, placement))
	}

	if world, err := p.client.GetWorldMeasures(); err != nil {
		p.log("world measures fetch failed", map[string]any{"error": err.Error()})
	} else {
		chicago := client.FilterChicago(world)
		sample := client.SampleN(chicago, p.cfg.ChicagoSampleSize)
		for _, m := range sample {
			m := m
			r := measureToReading(&m, "chicago")
			p.store.SaveReading(r)
			p.log("chicago reading", readingFields(r, nil))
		}
		p.log("poll cycle complete", map[string]any{
			"chicago_total":  len(chicago),
			"chicago_sample": len(sample),
		})
	}
}

func readingFields(r Reading, placement *Placement) map[string]any {
	f := map[string]any{
		"location_id":   r.LocationID,
		"location_name": strVal(r.LocationName),
		"timestamp":     r.Timestamp,
	}
	if r.Pm02 != nil {
		f["pm2.5"] = *r.Pm02
	}
	if r.Rco2 != nil {
		f["co2"] = *r.Rco2
	}
	if r.Atmp != nil {
		f["temp_c"] = *r.Atmp
	}
	if r.Rhum != nil {
		f["humidity"] = *r.Rhum
	}
	if placement != nil {
		f["placement"] = placement.Label
	}
	return f
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.RunOnce(ctx)
		}
	}
}

func measureToReading(m *client.Measure, source string) Reading {
	name := m.LocationName
	return Reading{
		LocationID:   m.LocationID,
		LocationName: &name,
		Source:       source,
		Timestamp:    m.Timestamp,
		Pm01:         m.Pm01,
		Pm02:         m.Pm02,
		Pm10:         m.Pm10,
		Pm003Count:   m.Pm003Count,
		Atmp:         m.Atmp,
		Rhum:         m.Rhum,
		Rco2:         m.Rco2,
		Tvoc:         m.Tvoc,
		TvocIndex:    m.TvocIndex,
		NoxIndex:     m.NoxIndex,
		HeatIndex:    m.Heatindex,
		Wifi:         m.Wifi,
	}
}

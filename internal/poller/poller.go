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
}

type Poller struct {
	client APIClient
	store  Store
	cfg    Config
}

func New(c APIClient, s Store, cfg Config) *Poller {
	return &Poller{client: c, store: s, cfg: cfg}
}

func (p *Poller) RunOnce(ctx context.Context) {
	placement, _ := p.store.CurrentPlacement()

	if own, err := p.client.GetOwnMeasure(p.cfg.LocationID); err == nil {
		r := measureToReading(own, "own")
		if placement != nil {
			r.PlacementID = &placement.ID
		}
		p.store.SaveReading(r)
	}

	if world, err := p.client.GetWorldMeasures(); err == nil {
		chicago := client.FilterChicago(world)
		sample := client.SampleN(chicago, p.cfg.ChicagoSampleSize)
		for _, m := range sample {
			m := m
			p.store.SaveReading(measureToReading(&m, "chicago"))
		}
	}
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

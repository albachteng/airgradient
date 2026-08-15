package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/albachteng/airgradient/internal/client"
	"github.com/albachteng/airgradient/internal/db"
	"github.com/albachteng/airgradient/internal/logger"
	"github.com/albachteng/airgradient/internal/poller"
)

func main() {
	godotenv.Load()

	apiKey := requireEnv("AIRGRADIENT_API_KEY")
	locationID := requireEnv("AIRGRADIENT_LOCATION_ID")
	dbPath := envOr("DB_PATH", "./airgradient.db")
	logDir := envOr("LOG_DIR", "./logs")
	intervalSec := envInt("POLL_INTERVAL_SECONDS", 3600)
	chicagoN := envInt("CHICAGO_SAMPLE_SIZE", 10)

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	l, err := logger.New(logDir)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer l.Close()

	c := client.New("https://api.airgradient.com", apiKey)
	if err := c.Ping(); err != nil {
		log.Fatalf("API ping failed: %v", err)
	}

	store := &dbStore{db: database}
	cfg := poller.Config{
		LocationID:        locationID,
		ChicagoSampleSize: chicagoN,
		Log:               l.Log,
	}
	p := poller.New(c, store, cfg)

	interval := time.Duration(intervalSec) * time.Second
	l.Log("poller starting", map[string]any{
		"interval_seconds": intervalSec,
		"location_id":      locationID,
		"chicago_sample":   chicagoN,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run once immediately, then on the ticker.
	p.RunOnce(ctx)
	p.Run(ctx, interval)

	l.Log("poller stopped", nil)
}

// dbStore adapts db.DB to the poller.Store interface.
type dbStore struct {
	db *db.DB
}

func (s *dbStore) SaveReading(r poller.Reading) error {
	return s.db.SaveReading(db.Reading{
		LocationID:   r.LocationID,
		LocationName: r.LocationName,
		PlacementID:  r.PlacementID,
		Source:       r.Source,
		Timestamp:    r.Timestamp,
		Pm01:         r.Pm01,
		Pm02:         r.Pm02,
		Pm10:         r.Pm10,
		Pm003Count:   r.Pm003Count,
		Atmp:         r.Atmp,
		Rhum:         r.Rhum,
		Rco2:         r.Rco2,
		Tvoc:         r.Tvoc,
		TvocIndex:    r.TvocIndex,
		NoxIndex:     r.NoxIndex,
		HeatIndex:    r.HeatIndex,
		Wifi:         r.Wifi,
	})
}

func (s *dbStore) CurrentPlacement() (*poller.Placement, error) {
	p, err := s.db.CurrentPlacement()
	if err != nil || p == nil {
		return nil, err
	}
	return &poller.Placement{ID: p.ID, Label: p.Label}, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "required env var %s is not set\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

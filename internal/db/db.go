package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Placement struct {
	ID          int
	Label       string
	Description string
	StartedAt   string
	EndedAt     *string
}

type Reading struct {
	ID           int
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

type DB struct {
	db *sql.DB
}

func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	d := &DB{db: sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS placements (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			label       TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			started_at  TEXT NOT NULL DEFAULT (datetime('now')),
			ended_at    TEXT
		);

		CREATE TABLE IF NOT EXISTS readings (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			location_id   INTEGER NOT NULL,
			location_name TEXT,
			placement_id  INTEGER REFERENCES placements(id),
			source        TEXT NOT NULL CHECK(source IN ('own','chicago')),
			timestamp     TEXT NOT NULL,
			pm01          REAL,
			pm02          REAL,
			pm10          REAL,
			pm003_count   INTEGER,
			atmp          REAL,
			rhum          INTEGER,
			rco2          INTEGER,
			tvoc          REAL,
			tvoc_index    INTEGER,
			nox_index     INTEGER,
			heat_index    INTEGER,
			wifi          INTEGER
		);
	`)
	return err
}

func (d *DB) SaveReading(r Reading) error {
	_, err := d.db.Exec(`
		INSERT INTO readings
			(location_id, location_name, placement_id, source, timestamp,
			 pm01, pm02, pm10, pm003_count, atmp, rhum, rco2, tvoc,
			 tvoc_index, nox_index, heat_index, wifi)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.LocationID, r.LocationName, r.PlacementID, r.Source, r.Timestamp,
		r.Pm01, r.Pm02, r.Pm10, r.Pm003Count, r.Atmp, r.Rhum, r.Rco2, r.Tvoc,
		r.TvocIndex, r.NoxIndex, r.HeatIndex, r.Wifi,
	)
	return err
}

func (d *DB) ReadingsSince(t time.Time) ([]Reading, error) {
	rows, err := d.db.Query(`
		SELECT id, location_id, location_name, placement_id, source, timestamp,
		       pm01, pm02, pm10, pm003_count, atmp, rhum, rco2, tvoc,
		       tvoc_index, nox_index, heat_index, wifi
		FROM readings
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
	`, t.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []Reading
	for rows.Next() {
		var r Reading
		if err := rows.Scan(
			&r.ID, &r.LocationID, &r.LocationName, &r.PlacementID, &r.Source, &r.Timestamp,
			&r.Pm01, &r.Pm02, &r.Pm10, &r.Pm003Count, &r.Atmp, &r.Rhum, &r.Rco2, &r.Tvoc,
			&r.TvocIndex, &r.NoxIndex, &r.HeatIndex, &r.Wifi,
		); err != nil {
			return nil, err
		}
		readings = append(readings, r)
	}
	return readings, rows.Err()
}

func (d *DB) SetPlacement(label, description string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`UPDATE placements SET ended_at = datetime('now') WHERE ended_at IS NULL`); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO placements (label, description) VALUES (?, ?)`, label, description); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) CurrentPlacement() (*Placement, error) {
	row := d.db.QueryRow(`
		SELECT id, label, description, started_at, ended_at
		FROM placements
		WHERE ended_at IS NULL
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`)
	var p Placement
	if err := row.Scan(&p.ID, &p.Label, &p.Description, &p.StartedAt, &p.EndedAt); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) ListPlacements() ([]Placement, error) {
	rows, err := d.db.Query(`
		SELECT id, label, description, started_at, ended_at
		FROM placements
		ORDER BY started_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var placements []Placement
	for rows.Next() {
		var p Placement
		if err := rows.Scan(&p.ID, &p.Label, &p.Description, &p.StartedAt, &p.EndedAt); err != nil {
			return nil, err
		}
		placements = append(placements, p)
	}
	return placements, rows.Err()
}

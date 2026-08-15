package db

import (
	"database/sql"
	"time"
)

type PlacementStats struct {
	Label        string
	ReadingCount int
	AvgPm25      *float64
	AvgCo2       *float64
	AvgTempC     *float64
	AvgHumidity  *float64
}

type LocationStats struct {
	LocationID   int
	LocationName string
	ReadingCount int
	AvgPm25      *float64
	AvgCo2       *float64
	AvgTempC     *float64
	AvgHumidity  *float64
}

func (d *DB) OwnStatsByPlacement(from, to time.Time) ([]PlacementStats, error) {
	rows, err := d.db.Query(`
		SELECT
			COALESCE(p.label, 'unplaced') AS label,
			COUNT(*)                      AS reading_count,
			AVG(r.pm02)                   AS avg_pm25,
			AVG(r.rco2)                   AS avg_co2,
			AVG(r.atmp)                   AS avg_temp_c,
			AVG(r.rhum)                   AS avg_humidity
		FROM readings r
		LEFT JOIN placements p ON r.placement_id = p.id
		WHERE r.source = 'own'
		  AND r.timestamp >= ?
		  AND r.timestamp < ?
		GROUP BY COALESCE(p.label, 'unplaced')
		ORDER BY label
	`, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PlacementStats
	for rows.Next() {
		var s PlacementStats
		var pm25, co2, tempC, hum sql.NullFloat64
		if err := rows.Scan(&s.Label, &s.ReadingCount, &pm25, &co2, &tempC, &hum); err != nil {
			return nil, err
		}
		if pm25.Valid {
			s.AvgPm25 = &pm25.Float64
		}
		if co2.Valid {
			s.AvgCo2 = &co2.Float64
		}
		if tempC.Valid {
			s.AvgTempC = &tempC.Float64
		}
		if hum.Valid {
			s.AvgHumidity = &hum.Float64
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (d *DB) ChicagoStatsByLocation(from, to time.Time) ([]LocationStats, error) {
	rows, err := d.db.Query(`
		SELECT
			r.location_id,
			COALESCE(MAX(r.location_name), '') AS location_name,
			COUNT(*)                            AS reading_count,
			AVG(r.pm02)                         AS avg_pm25,
			AVG(r.rco2)                         AS avg_co2,
			AVG(r.atmp)                         AS avg_temp_c,
			AVG(r.rhum)                         AS avg_humidity
		FROM readings r
		WHERE r.source = 'chicago'
		  AND r.timestamp >= ?
		  AND r.timestamp < ?
		GROUP BY r.location_id
		ORDER BY location_name
	`, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []LocationStats
	for rows.Next() {
		var s LocationStats
		var pm25, co2, tempC, hum sql.NullFloat64
		if err := rows.Scan(&s.LocationID, &s.LocationName, &s.ReadingCount, &pm25, &co2, &tempC, &hum); err != nil {
			return nil, err
		}
		if pm25.Valid {
			s.AvgPm25 = &pm25.Float64
		}
		if co2.Valid {
			s.AvgCo2 = &co2.Float64
		}
		if tempC.Valid {
			s.AvgTempC = &tempC.Float64
		}
		if hum.Valid {
			s.AvgHumidity = &hum.Float64
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

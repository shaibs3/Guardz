package db

import (
	"database/sql"
	"time"
)

// GetOrCreatePath inserts a path if it doesn't exist and returns its ID
func GetOrCreatePath(db *sql.DB, path string) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO paths (path) VALUES ($1)
		ON CONFLICT (path) DO UPDATE SET path=EXCLUDED.path
		RETURNING id`, path).Scan(&id)
	return id, err
}

// InsertURLRecord inserts a new URLRecord into the database
func InsertURLRecord(db *sql.DB, rec URLRecord) error {
	_, err := db.Exec(
		`INSERT INTO urls (path_id, url, content, status_code, fetched_at, error)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		rec.PathID, rec.URL, rec.Content, rec.StatusCode, rec.FetchedAt, rec.Error,
	)
	return err
}

// GetURLsByPath returns all URL records for a given path string
func GetURLsByPath(db *sql.DB, path string) ([]URLRecord, error) {
	var records []URLRecord
	rows, err := db.Query(`
		SELECT u.id, u.path_id, u.url, u.content, u.status_code, u.fetched_at, u.error
		FROM urls u
		JOIN paths p ON u.path_id = p.id
		WHERE p.path = $1
		ORDER BY u.fetched_at DESC
	`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rec URLRecord
		var fetchedAt time.Time
		var errStr sql.NullString
		err := rows.Scan(&rec.ID, &rec.PathID, &rec.URL, &rec.Content, &rec.StatusCode, &fetchedAt, &errStr)
		if err != nil {
			return nil, err
		}
		rec.FetchedAt = fetchedAt
		if errStr.Valid {
			rec.Error = &errStr.String
		} else {
			rec.Error = nil
		}
		records = append(records, rec)
	}
	return records, nil
}

package db

import (
	"database/sql"
)

// GetOrCreatePath inserts a path if it doesn't exist and returns its ID
func GetOrCreatePath(db *sql.DB, path string) (int64, error) {
	var id int64
	err := db.QueryRow(`INSERT INTO paths (path) VALUES ($1)
		ON CONFLICT (path) DO UPDATE SET path=EXCLUDED.path
		RETURNING id`, path).Scan(&id)
	return id, err
}

// InsertURLRecord inserts a new URLRecord into the database (only url and path_id)
func InsertURLRecord(db *sql.DB, pathID int64, url string) error {
	_, err := db.Exec(
		`INSERT INTO urls (path_id, url) VALUES ($1, $2)`,
		pathID, url,
	)
	return err
}

// GetURLsByPath returns all URL records for a given path string
func GetURLsByPath(db *sql.DB, path string) ([]URLRecord, error) {
	var records []URLRecord
	rows, err := db.Query(`
		SELECT u.id, u.path_id, u.url
		FROM urls u
		JOIN paths p ON u.path_id = p.id
		WHERE p.path = $1
		ORDER BY u.id ASC
	`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rec URLRecord
		err := rows.Scan(&rec.ID, &rec.PathID, &rec.URL)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

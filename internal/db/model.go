package db

import "time"

// Path represents a unique path
type Path struct {
	ID   int64  `db:"id" json:"id"`
	Path string `db:"path" json:"path"`
}

// URLRecord represents a fetched URL and its content
type URLRecord struct {
	ID         int64     `db:"id" json:"id"`
	PathID     int64     `db:"path_id" json:"path_id"`
	URL        string    `db:"url" json:"url"`
	Content    string    `db:"content" json:"content"`
	StatusCode int       `db:"status_code" json:"status_code"`
	FetchedAt  time.Time `db:"fetched_at" json:"fetched_at"`
	Error      *string   `db:"error" json:"error,omitempty"`
}

// Schema is the SQL schema for the paths and urls tables
const Schema = `
CREATE TABLE IF NOT EXISTS paths (
    id SERIAL PRIMARY KEY,
    path TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    path_id INTEGER REFERENCES paths(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    content TEXT,
    status_code INTEGER,
    fetched_at TIMESTAMP,
    error TEXT
);
`

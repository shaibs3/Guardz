package db_model

// Path represents a unique path
type Path struct {
	ID   uint64 `db_model:"id" json:"id"`
	Path string `db_model:"path" json:"path"`
}

// URLRecord represents a fetched URL and its content
type URLRecord struct {
	ID     uint64 `db_model:"id" json:"id"`
	PathID uint64 `db_model:"path_id" json:"path_id"`
	URL    string `db_model:"url" json:"url"`
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
    url TEXT NOT NULL
);
`

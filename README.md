# Guardz

A Go-based HTTP service that fetches and stores content from multiple URLs concurrently.

## Features

- **Concurrent URL fetching**: Fetches content from multiple URLs simultaneously using goroutines
- **RESTful API**: Simple POST and GET endpoints for submitting URLs and retrieving content
- **Error handling**: Comprehensive error handling with detailed error reporting
- **Thread-safe**: Uses mutex protection for concurrent access to shared data
- **Timeout protection**: 10-second timeout for each HTTP request

## Prerequisites

- Go 1.24.2 or higher
- PostgreSQL database (see below for setup)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd Guardz
```

2. Install dependencies:
```bash
go mod tidy
```

## Running the Service

### Development
```bash
go run cmd/main.go
```

### Production
```bash
go build -o guardz cmd/main.go
./guardz
```

The service will start on port 8080 by default. You can change the port by setting the `PORT` environment variable:

```bash
PORT=3000 go run cmd/main.go
```

## API Endpoints

### POST `/<some_path>`
Submit URLs to fetch content from and associate them with a path.

**Request:**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"urls": ["http://example.com", "http://another.com"]}' \
  http://localhost:8080/mytask
```

- Replace `/mytask` with any path you want.
- Replace the URLs in the JSON array as needed.

**Response:**
```json
{
  "message": "URLs stored successfully",
  "path": "mytask",
  "count": 2
}
```

### GET `/<some_path>`
Retrieve all previously fetched URL contents for a path.

**Request:**
```bash
curl http://localhost:8080/mytask
```

**Response:**
```json
{
  "path": "mytask",
  "results": [
    {
      "id": 1,
      "path_id": 1,
      "url": "http://example.com",
      "content": "...",
      "status_code": 200,
      "fetched_at": "2024-01-01T12:00:00Z",
      "error": null
    },
    ...
  ]
}
```

- Replace `/mytask` with the path you used in the POST request.

## Database Setup

- The application will automatically create the required tables (`paths`, `urls`) in your PostgreSQL database on startup.
- **You must create the database itself (e.g., `torq`) before running the app.**

Example command to create the database:
```bash
createdb -h localhost -U admin torq
```

## Example Environment Variable for DB Connection

```
DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgresql://admin:admin@localhost:5432/torq?sslmode=disable"}}'
```

## Example Usage

1. **Start the service:**
```bash
go run cmd/main.go
```

2. **Submit URLs to fetch:**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"urls": ["http://example.com", "http://another.com"]}' \
  http://localhost:8080/mytask
```

3. **Retrieve fetched content:**
```bash
curl http://localhost:8080/mytask
```

## Limitations

- **In-memory storage**: Not enabled by default; all data is stored in PostgreSQL
- **No persistence if DB is dropped**: Data will be lost if the database is deleted

## Future Enhancements

- Database integration for persistent storage
- Rate limiting and request throttling
- Authentication and authorization
- Content caching with TTL
- Metrics and monitoring
- Docker containerization
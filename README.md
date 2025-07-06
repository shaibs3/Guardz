# Guardz - URL Content Aggregation Service

A Go-based microservice that provides URL content aggregation functionality through a REST API. The service can fetch and aggregate content from multiple URLs with comprehensive security measures, parallel processing, and observability
## Features

- 🌐 **URL Content Aggregation**: Fetch content from multiple URLs in parallel
- 🔒 **Security First**: SSRF protection, URL validation, response size limits
- 🔄 **Redirect Handling**: Automatic redirect following with loop protection
- 📊 **Multiple Content Types**: Support for JSON, images, text, HTML with proper encoding
- 🚦 **Concurrent Request Limiting**: Configurable concurrency to prevent resource exhaustion
- 🚦 **Rate Limiting**: Configurable request rate limiting to prevent abuse
- 🐳 **Docker support** with docker-compose
- 🧪 **Comprehensive testing** with security validation
- 📈 **OpenTelemetry metrics** and observability
- 🔍 **Security scanning** with gosec and govulncheck
- 🚀 **Full CI/CD GitHub Actions pipeline** with Docker Hub image publishing
- 🗄️ **Flexible Database Backend**: PostgreSQL and in-memory providers

## Quick Start

### Prerequisites

- Go 1.24 or higher
- Docker (optional)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/shaibs3/Guardz.git
   cd Guardz
   ```
2. **Set up database configuration**

   **Option 1: Environment Variable**

   **For PostgreSQL:**
   ```bash
   export DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgresql://admin:admin@localhost:5432/guardz?sslmode=disable"}}'
   
   ```

   **Option 2: .env File**
   ```bash
   # Copy the example environment file
   cp .env_example .env
   
   # Or edit the .env file with your configuration
   
   DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgresql://admin:admin@localhost:5432/guardz?sslmode=disable"}}'
  
   RPS_LIMIT=10
   RPS_BURST=20
   PORT=8080
   ```

3. **Run the application**
   ```bash
   # Using docker-compose directly
   docker-compose up --build

   # Or using Make
   make compose-up
   ```

4. **Test the API**
   ```bash
   # Test against localhost (default)
   ./test_script.sh

   # Test against remote server
   ./test_script.sh remote
   ```

5. **Check metrics**
   ```bash
   # Test against localhost (default)
   curl "http://localhost:8080/metrics"
   # Test against remote server
   curl "http://34.55.142.196:8080/metrics"
   ```

## API Documentation

### Store URLs for a Path

**Endpoint:** `POST /{path}`

**Description:** Store a list of URLs associated with a specific path.

**Request Body:**
```json
{
  "urls": [
    "https://httpbin.org/json",
    "https://httpbin.org/image/png",
    "https://httpbin.org/robots.txt"
  ]
}
```

**Example Request:**
```bash
curl -X POST http://localhost:8080/my-path \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/json",
      "https://httpbin.org/image/png"
    ]
  }'
```

**Example Response:**
```json
{
  "message": "URLs stored successfully",
  "path": "my-path",
  "count": 2
}
```

**Error Response (Invalid URLs):**
```json
{
  "message": "URLs stored successfully",
  "path": "my-path",
  "count": 1,
  "invalid_urls": [
    "http://localhost:8080/api: access to localhost is not allowed"
  ],
  "warning": "Some URLs were rejected: 1 valid, 1 invalid"
}
```

### Fetch Content from URLs

**Endpoint:** `GET /{path}`

**Description:** Fetch content from all URLs associated with the specified path.

**Example Request:**
```bash
curl http://localhost:8080/my-path
```

**Example Response:**
```json
{
  "path": "my-path",
  "results": [
    {
      "url": "https://httpbin.org/json",
      "redirected": false,
      "status_code": 200,
      "content_type": "application/json",
      "content": "{\"slideshow\": {\"author\": \"Yours Truly\", \"date\": \"date of publication\", \"slides\": [{\"title\": \"Wake up to WonderWidgets!\", \"type\": \"all\"}, {\"items\": [\"Why <em>WonderWidgets</em> are great\", \"Who <em>buys</em> WonderWidgets\"], \"title\": \"Overview\", \"type\": \"all\"}], \"title\": \"Sample Slide Show\"}}"
    },
    {
      "url": "https://httpbin.org/image/png",
      "redirected": false,
      "status_code": 200,
      "content_type": "image/png",
      "content": "iVBORw0KGgoAAAANSUhEUgAA..."
    }
  ]
}
```

**Response with Redirects:**
```json
{
  "path": "my-path",
  "results": [
    {
      "url": "https://example.com",
      "original_url": "https://example.com",
      "final_url": "https://www.example.com",
      "redirected": true,
      "status_code": 200,
      "content_type": "text/html",
      "content": "<!DOCTYPE html>..."
    }
  ]
}
```

**Response with Errors:**
```json
{
  "path": "my-path",
  "results": [
    {
      "url": "https://invalid-url.com",
      "error": "Get \"https://invalid-url.com\": dial tcp: lookup invalid-url.com: no such host"
    }
  ]
}
```

### Health Check Endpoints

#### Liveness Probe

**Endpoint:** `GET /health/live`

**Example Request:**
```bash
curl "http://localhost:8080/health/live"
```

**Example Response:**
```json
{
  "status": "alive",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "guardz"
}
```

#### Readiness Probe

**Endpoint:** `GET /health/ready`

**Example Request:**
```bash
curl "http://localhost:8080/health/ready"
```

**Example Response:**
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "guardz"
}
```

### Metrics Endpoint

**Endpoint:** `GET /metrics`

**Description:** Exposes Prometheus metrics for monitoring and observability.

**Example Request:**
```bash
curl "http://localhost:8080/metrics"
```

## Configuration

The service supports flexible database configuration using JSON. You can use either PostgreSQL or in-memory database providers.

### Database Configuration

Set the `DB_CONFIG` environment variable with JSON configuration:

**For In-Memory Provider (Development/Testing):**
```bash
export DB_CONFIG='{"dbtype": "memory"}'
```

**For PostgreSQL Provider (Production):**
```bash
export DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgresql://admin:admin@localhost:5432/guardz?sslmode=disable"}}'
```

### Environment Variables

| Variable    | Description                           | Default |
|-------------|---------------------------------------|---------|
| `DB_CONFIG` | JSON configuration for database       | -       |
| `PORT`      | Server port                           | `8080`  |
| `RPS_LIMIT` | Rate limiting (requests per second)   | `100`   |
| `RPS_BURST` | Rate limiting burst                   | `200`   |
| `LOG_LEVEL` | Log level                             | `info`  |

### Rate Limiting Configuration

The service implements configurable rate limiting to prevent abuse and ensure fair usage:

- **RPS_LIMIT**: Maximum requests per second (default: 100)
- **RPS_BURST**: Maximum burst requests allowed (default: 200)

**Example configurations:**

**Conservative rate limiting (for shared environments):**
```bash
export RPS_LIMIT=10
export RPS_BURST=20
```

**High throughput (for dedicated servers):**
```bash
export RPS_LIMIT=1000
export RPS_BURST=2000
```

**Docker Compose configuration:**
```yaml
environment:
  RPS_LIMIT: 100
  RPS_BURST: 200
```

### Available Make Commands

```bash
# Build the application
make build

# Run the application
make run

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Run security scan
make security

# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Stop Docker container
make docker-stop

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test ./internal/lookup -v
```

## Observability

The service exposes comprehensive metrics via Prometheus at `/metrics` endpoint for monitoring and observability.

### Available Metrics

#### HTTP Metrics

- **`http_request_duration_seconds`** (histogram):
  Duration of HTTP requests in seconds. Includes labels for `method`, `path`, and `status_code`.

- **`http_requests_total`** (counter):
  Total number of HTTP requests. Includes labels for `method`, `path`, and `status_code`.

- **`http_error_requests_total`** (counter):
  Total number of HTTP error requests (4xx, 5xx status codes). Includes labels for `method`, `path`, and `status_code`.

- **`http_response_status_total`** (counter):
  Total number of HTTP responses by status code. Includes label for `status_code`.

- **`http_requests_in_flight`** (updown counter):
  Number of HTTP requests currently in flight (active requests).

- **`http_rate_limited_requests_total`** (counter):
  Total number of HTTP requests that were rate limited.

#### Database Metrics

- **`ip_lookup_duration_seconds`** (histogram):
  Duration of database operations in seconds. Useful for monitoring database performance.

- **`ip_lookup_errors_total`** (counter):
  Total number of database operation errors. Useful for alerting on data issues.

#### Business Metrics

The service tracks URL fetching performance and success rates through the HTTP metrics above, providing insights into:
- URL fetch success/failure rates
- Response times for different content types
- Redirect handling performance
- Content size distribution

### Metrics Endpoint

**Endpoint:** `GET /metrics`

**Description:** Exposes Prometheus-formatted metrics for monitoring and alerting.

**Example Request:**
```bash
curl "http://localhost:8080/metrics"
```

**Example Output:**
```
# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/my-path",status_code="200",le="0.1"} 15
http_request_duration_seconds_bucket{method="GET",path="/my-path",status_code="200",le="0.5"} 25
http_request_duration_seconds_bucket{method="GET",path="/my-path",status_code="200",le="1"} 30
http_request_duration_seconds_bucket{method="GET",path="/my-path",status_code="200",le="+Inf"} 30

# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/my-path",status_code="200"} 30
http_requests_total{method="POST",path="/my-path",status_code="201"} 15

# HELP http_rate_limited_requests_total Total number of HTTP requests that were rate limited
# TYPE http_rate_limited_requests_total counter
http_rate_limited_requests_total 5
```





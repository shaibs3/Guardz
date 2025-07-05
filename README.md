# Guardz - URL Content Aggregation Service

A Go-based microservice that provides URL content aggregation functionality through a REST API. The service can fetch and aggregate content from multiple URLs with comprehensive security measures, parallel processing, and observability.

## Features

- 🌐 **URL Content Aggregation**: Fetch content from multiple URLs in parallel
- 🔒 **Security First**: SSRF protection, URL validation, response size limits
- 🔄 **Redirect Handling**: Automatic redirect following with loop protection
- 📊 **Multiple Content Types**: Support for JSON, images, text, HTML with proper encoding
- 🚦 **Concurrent Request Limiting**: Configurable concurrency to prevent resource exhaustion
- 🐳 **Docker support** with docker-compose
- 🧪 **Comprehensive testing** with security validation
- 📈 **OpenTelemetry metrics** and observability
- 🔍 **Security scanning** with gosec and govulncheck
- 🚀 **Full CI/CD GitHub Actions pipeline** with Docker Hub image publishing
- 🗄️ **Flexible Database Backend**: PostgreSQL and in-memory providers

## Quick Start

### Prerequisites

- Go 1.24 or higher
- Docker and docker-compose (optional)
- PostgreSQL (optional, for production)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/shaibs3/Guardz.git
   cd Guardz
   ```

2. **Set up database configuration**
   
   **Option 1: Environment Variable**
   
   **For In-Memory (Development):**
   ```bash
   export DB_CONFIG='{"dbtype": "memory"}'
   ```
   
   **For PostgreSQL:**
   ```bash
   export DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgres://admin:admin@localhost:5432/guardz?sslmode=disable"}}'
   ```
   
   **Option 2: .env File**
   ```bash
   # Copy the example environment file
   cp .env_example .env
   
   # Edit the .env file with your configuration
   # For In-Memory:
   DB_CONFIG='{"dbtype": "memory"}'
   
   # For PostgreSQL:
   # DB_CONFIG='{"dbtype": "postgres", "extra_details": {"conn_str": "postgres://admin:admin@localhost:5432/guardz?sslmode=disable"}}'
   
   PORT=8080
   ```

3. **Run the application**
   ```bash
   # Using Go directly
   go run cmd/main.go

   # Or using Make
   make run
   ```

4. **Test the API**
   ```bash
   # Store URLs
   curl -X POST http://localhost:8080/test-path \
     -H "Content-Type: application/json" \
     -d '{"urls": ["https://httpbin.org/json", "https://httpbin.org/image/png"]}'

   # Fetch content
   curl http://localhost:8080/test-path
   ```

### Using Docker

1. **Start with docker-compose (includes PostgreSQL)**
   ```bash
   docker-compose up -d
   ```

2. **Or build and run manually**
   ```bash
   # Build the Docker image
   make docker-build

   # Run the container
   make docker-run
   ```

3. **Test the API**
   ```bash
   curl -X POST http://localhost:8080/test-path \
     -H "Content-Type: application/json" \
     -d '{"urls": ["https://httpbin.org/json"]}'
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

## Configuration

### Database Configuration

The service supports flexible database configuration using JSON:

#### Supported Database Types

- `"memory"` - In-memory provider (for development/testing)
- `"postgres"` - PostgreSQL database provider (for production)

#### Configuration Examples

**In-Memory Provider:**
```json
{
  "dbtype": "memory"
}
```

**PostgreSQL Provider:**
```json
{
  "dbtype": "postgres",
  "extra_details": {
    "conn_str": "postgres://username:password@localhost:5432/guardz?sslmode=disable"
  }
}
```

### Environment Variables

| Variable    | Description                           | Default |
|-------------|---------------------------------------|---------|
| `DB_CONFIG` | JSON configuration for database       | -       |
| `PORT`      | Server port                           | `8080`  |
| `LOG_LEVEL` | Log level                             | `info`  |

## Security Features

### URL Validation & SSRF Protection

- **Scheme Restriction**: Only `http` and `https` schemes allowed
- **Private IP Blocking**: Blocks access to localhost, private IP ranges
- **URL Format Validation**: Ensures URLs are properly formatted

### Response Size Limits

- **1MB Limit**: All responses are limited to 1MB to prevent memory exhaustion
- **Truncation Warning**: Clear indication when responses are truncated

### Concurrent Request Limiting

- **Max 10 Concurrent**: Prevents resource exhaustion attacks
- **Semaphore-based**: Efficient concurrency control

### Redirect Protection

- **Max 10 Redirects**: Prevents infinite redirect loops
- **Redirect Tracking**: Full visibility into redirect chains

## Available Make Commands

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

# Run vulnerability check
make vulncheck

# Build Docker image
make docker-build

# Run Docker container
make docker-run

# Stop Docker container
make docker-stop

# Clean build artifacts
make clean

# Clean database
make db-clean

# Run load testing
make test-load

# Show all available commands
make help
```

## Testing

### Run All Tests
```bash
make test
```

### Run Security Tests
```bash
# Run security validation tests
go test -v ./internal/handlers/ -run "TestDynamicHandler_Security"

# Run content type tests
go test -v ./internal/handlers/ -run "TestDynamicHandler_MultipleContentTypes"
```

### Load Testing
```bash
# Run the load test script
make test-load
```

### Manual Testing Examples

**Test with different content types:**
```bash
# Store URLs with different content types
curl -X POST http://localhost:8080/content-test \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://httpbin.org/json",
      "https://httpbin.org/image/png",
      "https://httpbin.org/robots.txt"
    ]
  }'

# Fetch and verify content
curl http://localhost:8080/content-test
```

**Test security features:**
```bash
# Try to store invalid URLs (should be rejected)
curl -X POST http://localhost:8080/security-test \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "http://localhost:8080/api",
      "file:///etc/passwd",
      "https://httpbin.org/json"
    ]
  }'
```

## Docker Compose Setup

The project includes a `docker-compose.yml` file for easy development setup:

```bash
# Start all services (app + PostgreSQL)
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Clean up
docker-compose down -v
```

## CI/CD Pipeline

The project includes a comprehensive GitHub Actions pipeline with:

- **Testing**: Unit tests with coverage
- **Linting**: Code quality checks
- **Security Scanning**: gosec and govulncheck
- **Docker Build**: Automated image building
- **Deployment**: SSH-based deployment to remote servers

## Observability

### Metrics

The service exposes OpenTelemetry metrics for monitoring:

- **Request Duration**: Histogram of request processing times
- **Request Count**: Total number of requests by method/path/status
- **Error Rate**: Count of error responses
- **URL Fetch Metrics**: Duration and success rate of URL fetching

### Logging

Structured logging with Zap logger including:
- Request/response logging
- Error tracking
- Security event logging
- Performance metrics

## Security Considerations

### Implemented Security Measures

1. **Input Validation**: All URLs are validated before processing
2. **SSRF Protection**: Blocks access to private IP ranges and localhost
3. **Response Size Limits**: Prevents memory exhaustion attacks
4. **Concurrent Request Limiting**: Prevents resource exhaustion
5. **Redirect Loop Protection**: Limits redirect chains
6. **Content Type Handling**: Safe handling of binary and text content
7. **Error Sanitization**: Prevents information leakage

### Security Testing

The project includes comprehensive security tests:
- SSRF attack prevention
- Invalid URL rejection
- Response size limit validation
- Concurrent request limiting
- Redirect loop detection

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License.
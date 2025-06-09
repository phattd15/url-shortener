# URL Shortener Service

A high-performance URL shortener service built with Go, Gin framework, PostgreSQL, and Redis caching with comprehensive Swagger API documentation.

## Features

- **URL Shortening**: Convert long URLs into short, shareable links
- **URL Redirection**: Redirect short URLs to original destinations
- **Click Tracking**: Track the number of clicks for each short URL
- **URL Expiration**: Optional expiration dates for URLs
- **Statistics**: Get detailed stats for each short URL
- **Health Check**: Monitor service health including database and cache
- **API Documentation**: Comprehensive Swagger/OpenAPI documentation
- **Redis Caching**: High-performance caching layer to reduce database load
- **Graceful Degradation**: Service continues to function even if Redis is unavailable

## Performance Features

- **Redis Cache Layer**: Caches URL mappings, click counts, and statistics
- **Async Click Tracking**: Non-blocking click count updates
- **Cache-First Strategy**: Always checks cache before hitting the database
- **Automatic Cache Invalidation**: Smart cache invalidation when data changes
- **Fallback Mechanism**: Falls back to database when cache is unavailable

## API Documentation

Once the service is running, you can access the interactive Swagger documentation at:
- **Swagger UI**: http://localhost:8080/swagger/index.html

## Quick Start

### Option 1: Using Make (Recommended for Development)
```bash
# Clone the repository
git clone <repository-url>
cd url-shortener

# Complete development setup (installs tools, starts DB, generates docs)
make dev-setup

# Run the application
make run

# The service will be available at http://localhost:8080
# Swagger docs at http://localhost:8080/swagger/index.html
```

### Option 2: Using Docker Compose (Full Stack)
```bash
# Clone the repository
git clone <repository-url>
cd url-shortener

# Start database, Redis, and application
docker-compose up -d
# or
make docker-run

# The service will be available at http://localhost:8080
# Swagger docs at http://localhost:8080/swagger/index.html
```

## Development Commands

The project includes a Makefile with common development tasks:

```bash
# View all available commands
make help

# Development workflow
make dev-setup          # One-time setup for development
make run                 # Run the application
make test                # Run tests
make swagger-gen         # Regenerate Swagger docs

# Database management
make dev-db              # Start development database
make stop-db             # Stop development database

# Docker workflow
make docker-build        # Build Docker image
make docker-run          # Run with Docker Compose
make docker-stop         # Stop Docker containers
make logs                # View application logs

# Build
make build               # Build development binary
make prod-build          # Build production binary
```

## API Endpoints

### Create Short URL
```
POST /shorten
Content-Type: application/json

{
  "url": "https://example.com/very/long/url",
  "expires_in": 30  // optional, in days
}
```

**Response:**
```json
{
  "short_url": "http://localhost:8080/abc123",
  "original_url": "https://example.com/very/long/url",
  "short_code": "abc123",
  "expires_at": "2024-02-15T10:30:00Z"
}
```

### Redirect Short URL
```
GET /{shortCode}
```
Redirects to the original URL and increments click count.

### Get URL Statistics
```
GET /stats/{shortCode}
```

**Response:**
```json
{
  "original_url": "https://example.com/very/long/url",
  "short_code": "abc123",
  "click_count": 42,
  "created_at": "2024-01-15T10:30:00Z",
  "expires_at": "2024-02-15T10:30:00Z"
}
```

### Health Check
```
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "service": "url-shortener",
  "database": {"healthy": true},
  "cache": {"healthy": true}
}
```

Health check status can be:
- `healthy`: All services operational
- `degraded`: Database healthy but cache unavailable
- `unhealthy`: Database unavailable (service non-functional)

## Manual Setup

### Prerequisites
- Go 1.23 or higher
- Docker and Docker Compose (for PostgreSQL and Redis)
- PostgreSQL (if not using Docker)
- Redis (optional, for caching)
- Make (optional, for using Makefile commands)

### 1. Clone and Install Dependencies
```bash
git clone <repository-url>
cd url-shortener
go mod tidy
```

### 2. Generate Swagger Documentation (if modified)
```bash
# Install swag CLI tool
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init -g cmd/server/main.go
```

### 3. Start Services
Using Docker Compose (recommended):
```bash
docker-compose up -d postgres redis
```

Or use your own services and set environment variables:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=urlshortener
export REDIS_ADDR=localhost:6379
export REDIS_PASSWORD=""
export REDIS_DB=0
```

### 4. Run the Application
```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`

### 5. Test the Service
```bash
# Create a short URL
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/gin-gonic/gin"}'

# Test redirection (replace abc123 with actual short code)
curl -L http://localhost:8080/abc123

# Get statistics
curl http://localhost:8080/stats/abc123

# Check health
curl http://localhost:8080/health

# View API documentation
open http://localhost:8080/swagger/index.html
```

## Configuration

Environment variables:

### Server Configuration
- `PORT`: Server port (default: 8080)
- `GIN_MODE`: Gin mode (default: debug, set to release for production)

### Database Configuration
- `DB_HOST`: Database host (default: localhost)
- `DB_PORT`: Database port (default: 5432)
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (default: password)
- `DB_NAME`: Database name (default: urlshortener)

### Redis Configuration
- `REDIS_ADDR`: Redis address (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password (default: "")
- `REDIS_DB`: Redis database number (default: 0)

**Note**: If Redis is unavailable, the service will operate without caching, falling back to database-only operations.

## Development

### Project Structure
```
url-shortener/
├── cmd/
│   └── server/
│       └── main.go         # Application entry point
├── cache/                  # Redis cache layer
│   └── redis.go           # Cache operations and client
├── docs/                   # Auto-generated Swagger documentation
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── database/
│   └── database.go         # Database connection and setup
├── models/
│   └── url.go             # Data models and request/response types
├── handlers/
│   └── url.go             # HTTP handlers with Swagger annotations
├── utils/
│   └── shortener.go       # Utility functions
├── go.mod                 # Go module file
├── Makefile               # Development commands
├── docker-compose.yml     # Full stack setup
├── Dockerfile             # Application container
└── README.md              # This file
```

### Database Schema
The application uses GORM for database operations. The URL table includes:
- `id`: Primary key
- `original_url`: The original long URL
- `short_code`: The generated short code (6 character alphanumeric)
- `click_count`: Number of times the URL was accessed
- `expires_at`: Optional expiration timestamp
- `created_at`, `updated_at`, `deleted_at`: GORM timestamps

### Cache Strategy
- **URL Mappings**: Cached for 24 hours
- **Statistics**: Cached for 5 minutes
- **Click Counts**: Real-time updates in cache, periodic sync to database
- **Original URL Lookups**: Cached to avoid duplicate short codes

### Adding New API Endpoints

1. Add handler function with Swagger annotations in `handlers/`
2. Register route in `cmd/server/main.go`
3. Regenerate Swagger docs: `make swagger-gen`

Example Swagger annotations:
```go
// FunctionName godoc
// @Summary Brief description
// @Description Detailed description
// @Tags tag-name
// @Accept json
// @Produce json
// @Param param-name path string true "Description"
// @Success 200 {object} ResponseType
// @Failure 400 {object} map[string]string
// @Router /endpoint [method]
```

## Deployment Options

### Option 1: Docker Compose (Recommended)
```bash
# Full stack deployment with PostgreSQL and Redis
make docker-run
# or
docker-compose up -d
```

### Option 2: Build and Run Manually
```bash
# Build binary
make build
# or
go build -o url-shortener cmd/server/main.go

# Run binary
./bin/url-shortener
```

### Option 3: Docker Container Only
```bash
# Build image
make docker-build

# Run container (requires external PostgreSQL and Redis)
docker run -p 8080:8080 \
  -e DB_HOST=your-postgres-host \
  -e DB_PASSWORD=your-password \
  -e REDIS_ADDR=your-redis-host:6379 \
  url-shortener
```

## Production Considerations

- Set `GIN_MODE=release` for production
- Use environment variables for sensitive configuration
- Set up proper logging and monitoring
- Consider using a reverse proxy (nginx) for SSL termination
- Implement rate limiting for production use
- Configure Redis persistence and clustering for high availability
- Set up Redis password authentication in production
- Regularly backup the PostgreSQL database
- Monitor both database and cache performance
- Consider implementing cache warming strategies for frequently accessed URLs

## Performance Optimization

The service implements several performance optimizations:

1. **Cache-First Strategy**: Always checks Redis before database
2. **Async Operations**: Click tracking doesn't block redirects
3. **Smart Caching**: Different TTLs for different data types
4. **Graceful Degradation**: Continues functioning without Redis
5. **Connection Pooling**: Efficient database and Redis connections

## API Testing

You can use the interactive Swagger UI for testing all endpoints:
1. Start the service: `make run`
2. Open http://localhost:8080/swagger/index.html
3. Use the "Try it out" feature for each endpoint

Alternatively, you can use the provided curl examples or any API testing tool like Postman, Insomnia, or httpie.

## Monitoring

The health check endpoint provides detailed status information about all service components, making it easy to integrate with monitoring systems like Prometheus, Datadog, or custom health check services.
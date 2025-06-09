.PHONY: build run test clean docker-build docker-run swagger-gen dev-db stop-db dev-cache stop-cache

# Build the application
build:
	go build -o bin/url-shortener cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf docs/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Generate Swagger documentation
swagger-gen:
	swag init -g cmd/server/main.go

# Install Swagger CLI tool
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

# Start development database
dev-db:
	docker-compose up -d postgres

# Start development cache (Redis)
dev-cache:
	docker-compose up -d redis

# Start both database and cache
dev-services:
	docker-compose up -d postgres redis

# Stop development database
stop-db:
	docker-compose stop postgres

# Stop development cache
stop-cache:
	docker-compose stop redis

# Stop all services
stop-services:
	docker-compose down

# Build Docker image
docker-build:
	docker build -t url-shortener .

# Run with Docker Compose (full stack)
docker-run:
	docker-compose up -d

# Stop Docker containers
docker-stop:
	docker-compose down

# View logs
logs:
	docker-compose logs -f app

# View Redis logs
redis-logs:
	docker-compose logs -f redis

# Redis CLI access
redis-cli:
	docker-compose exec redis redis-cli

# Development setup (install tools, start services, generate docs)
dev-setup: swagger-install deps swagger-gen dev-services
	@echo "Development environment ready!"
	@echo "Services started: PostgreSQL and Redis"
	@echo "Run 'make run' to start the server"
	@echo "Visit http://localhost:8080/swagger/index.html for API docs"

# Production build
prod-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/url-shortener cmd/server/main.go

# Help target
help:
	@echo "Available targets:"
	@echo "  build           - Build the application binary"
	@echo "  run             - Run the application in development mode"
	@echo "  test            - Run tests"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Install and tidy dependencies"
	@echo "  swagger-gen     - Generate Swagger documentation"
	@echo "  swagger-install - Install Swagger CLI tool"
	@echo ""
	@echo "Development Services:"
	@echo "  dev-db          - Start development database (PostgreSQL)"
	@echo "  dev-cache       - Start development cache (Redis)"
	@echo "  dev-services    - Start both database and cache"
	@echo "  stop-db         - Stop development database"
	@echo "  stop-cache      - Stop development cache"
	@echo "  stop-services   - Stop all development services"
	@echo ""
	@echo "Docker Commands:"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-run      - Run with Docker Compose (full stack)"
	@echo "  docker-stop     - Stop Docker containers"
	@echo "  logs            - View application logs"
	@echo "  redis-logs      - View Redis logs"
	@echo "  redis-cli       - Access Redis CLI"
	@echo ""
	@echo "Setup & Build:"
	@echo "  dev-setup       - Complete development environment setup"
	@echo "  prod-build      - Build for production"
	@echo "  help            - Show this help message" 
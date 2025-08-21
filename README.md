# Laundry Scheduler

Living with shared laundry can be a bit annoying. This application can be shared with other apartments to keep track of who's currently in the laundry from the comfort of their own couch!

## Quick Start

### Using Docker (Recommended)

```bash
# Pull and run a specific version (recommended for production)
docker run -p 8080:8080 ghcr.io/captainsloths/laundry-scheduler:v1

# Or using docker-compose with versioned tag
docker-compose up -d
```

### Local Development

```bash
# Build and run locally
go run main.go

# Or using Docker for development
docker-compose -f docker-compose.dev.yml up --build
```

## Features

- **Responsive Web Interface** - Works on desktop and mobile
- **Dark/Light Mode Toggle** - Choose your preferred theme
- **Real-time Queue Management** - See who's doing laundry and when
- **Auto-cleanup** - Completed items are automatically removed
- **Docker Support** - Easy deployment with containers

## Development

### Prerequisites
- Go 1.24+
- Docker (optional)

### Local Setup
1. Clone the repository
2. Run `go mod download`
3. Run `go run main.go`
4. Open http://localhost:8080

### Docker Development
```bash
# Build and run with hot reload for static files
docker-compose -f docker-compose.dev.yml up --build
```

## Deployment

### Docker
```bash
# Production deployment (use specific version)
docker run -d -p 8080:8080 --name laundry-scheduler \
  ghcr.io/captainsloths/laundry-scheduler:v1
```

### Docker Compose
```yaml
version: '3.8'
services:
  laundry-scheduler:
    image: ghcr.io/captainsloths/laundry-scheduler:v1  # Use specific version
    ports:
      - "8080:8080"
    restart: unless-stopped
```

## Container Registry

Images are automatically built and published to GitHub Container Registry:

- **Versioned**: `ghcr.io/captainsloths/laundry-scheduler:v1`, `v2`, `v3`, etc.
- **Latest**: `ghcr.io/captainsloths/laundry-scheduler:latest` (development only)

**Production Warning**: Always use versioned tags in production. The `latest` tag can change unexpectedly and break your deployment.

## Configuration

The application runs on port 8080 by default. No additional configuration required!



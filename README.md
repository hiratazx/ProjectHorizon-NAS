# ProjectHorizon

A TrueNAS-inspired NAS dashboard that can be deployed on existing Linux systems, similar to CasaOS.

## Features

- 📊 **Dashboard** - Real-time system monitoring with widgets for CPU, Memory, Storage, and Network
- 🐳 **Docker Management** - View, start, stop, and restart containers
- 💾 **Storage Browser** - Browse files and view disk usage
- 🌙 **Dark Theme** - TrueNAS-inspired premium dark UI
- 🔧 **Easy Install** - CasaOS-style installation with systemd service

## Quick Start using Docker

```bash
# Pull the latest image
docker pull hiratazx/projecthorizon:latest

# Run the container (Example: Port 8080, sharing your home directory)
docker run -d \
  --name projecthorizon \
  -v /path/to/your/date:/media:rw \
  -p 8080:8080 \
  hiratazx/projecthorizon:latest

# Access at http://localhost:$TARGET_PORT
```
If you want to build the Docker Image, [see this](#build-the-docker-image).

## Build

```bash
# Build
make build

# Run
./build/horizon

# Access at http://localhost:8080
```

## Installation

```bash
# Install as system service
sudo make install

# Service will auto-start and be available at http://your-ip:8080

# Uninstall
sudo make uninstall
```

## Development

```bash
# Run in development mode (with hot reload)
make dev

# Build for all platforms
make build-all

# Create distribution tarball
make dist
```

## Build the Docker Image

```bash
# Build
docker build . -t hiratazx/projecthorizon
# You can change the 'hiratazx/projecthorizon' whatever you want
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/system/info` | System information |
| `GET /api/system/cpu` | CPU usage |
| `GET /api/system/memory` | Memory usage |
| `GET /api/system/network` | Network interfaces |
| `GET /api/docker/info` | Docker overview |
| `GET /api/docker/containers` | List containers |
| `POST /api/docker/containers/:id/:action` | Container actions |
| `GET /api/storage/usage` | Disk usage |
| `GET /api/storage/browse?path=` | Browse files |


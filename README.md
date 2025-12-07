# Kubernetes Monitoring Tool

Monitor your Kubernetes cluster with real-time alerts and a web dashboard.

## Quick Start

```bash
docker-compose up -d
open http://localhost:8080
```

## What It Does

- Monitors Kubernetes pods and nodes
- Alerts on issues (crashes, high CPU/memory, restarts)
- Real-time web dashboard with live updates
- Email notifications
- Stores alerts in PostgreSQL

## Architecture & Data Flow

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                           │
│                    (Pods, Nodes, Metrics Server)                    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Watches & Collects
                               ↓
┌─────────────────────────────────────────────────────────────────────┐
│                         Data Collectors                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────────┐  │
│  │ Pod Watcher  │  │ Node Watcher │  │ Metrics Watcher (CPU/   │  │
│  │              │  │              │  │ Memory via metrics API) │  │
│  └──────────────┘  └──────────────┘  └─────────────────────────┘  │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Sends Events
                               ↓
┌─────────────────────────────────────────────────────────────────────┐
│                         Alert Engine                                │
│  • Evaluates rules (CPU > 80%, CrashLoopBackOff, etc.)             │
│  • Determines severity (Critical, High, Medium)                     │
│  • Creates/Updates alert records                                    │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ Publishes Alerts
                               ↓
┌─────────────────────────────────────────────────────────────────────┐
│                          Event Bus (Pub/Sub)                        │
│              Distributes alerts to multiple observers               │
└────────┬──────────────────────┬─────────────────────┬───────────────┘
         │                      │                     │
         ↓                      ↓                     ↓
┌────────────────┐    ┌─────────────────┐   ┌──────────────────┐
│   PostgreSQL   │    │  WebSocket Hub  │   │ Email Notifier   │
│   Repository   │    │                 │   │    (Optional)    │
│                │    │  • Maintains    │   │                  │
│  • Stores all  │    │    connections  │   │  • Sends SMTP    │
│    alerts      │    │  • Broadcasts   │   │    alerts to     │
│  • Provides    │    │    to clients   │   │    team          │
│    history     │    │                 │   │                  │
└────────────────┘    └────────┬────────┘   └──────────────────┘
                               │ Real-time Updates
                               ↓
                     ┌──────────────────────┐
                     │   Web Dashboard      │
                     │   (Alpine.js UI)     │
                     │                      │
                     │  • Shows alerts      │
                     │  • Live updates      │
                     │  • Alert history     │
                     └──────────────────────┘
```

### How It Works

1. **Collection Phase**
   - Three watchers continuously monitor Kubernetes cluster
   - Pod Watcher: Tracks pod status changes (Running, Failed, CrashLoopBackOff)
   - Node Watcher: Monitors node conditions (Ready, MemoryPressure, DiskPressure)
   - Metrics Watcher: Polls metrics-server every 60s for CPU/Memory usage

2. **Evaluation Phase**
   - Alert Engine receives events from collectors
   - Applies configurable rules (CPU > 80%, restarts > 3, etc.)
   - Assigns severity levels (Critical, High, Medium)
   - Creates or updates alert records with timestamps

3. **Distribution Phase**
   - Event Bus publishes alerts to three observers in parallel
   - PostgreSQL stores alerts for historical analysis
   - WebSocket Hub pushes to all connected dashboard clients
   - Email Notifier sends notifications (if enabled)

4. **Presentation Phase**
   - Web UI receives real-time updates via WebSocket
   - REST API provides alert queries and statistics
   - Dashboard shows current status and alert history

## Features

- **Real-time monitoring** - Watches pods, nodes, and resource usage
- **Smart alerts** - Detects crashes, OOM kills, high resource usage
- **Web dashboard** - Clean UI with live WebSocket updates
- **Email alerts** - Optional SMTP notifications
- **Easy configuration** - Environment variables or config file
- **Docker ready** - Run with docker-compose

## Local Development

**Requirements:** Go 1.24+, PostgreSQL, kubectl configured

```bash
# 1. Build
make build

# 2. Configure (optional)
cp .env.example .env

# 3. Run
POSTGRES_PASSWORD=postgres ./bin/monitoring-tool
```

## Configuration

Set via environment variables or `.env` file:

```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_PASSWORD=postgres

# Alert thresholds (percentage)
ALERT_POD_CPU_THRESHOLD=80
ALERT_POD_MEMORY_THRESHOLD=85

# Email (optional)
EMAIL_ENABLED=false
SMTP_HOST=smtp.gmail.com
SMTP_PASSWORD=your-password
SMTP_TO=alerts@example.com
```

See [.env.example](.env.example) for all options.

## API Endpoints

**Alerts**
- `GET /api/alerts/recent` - Last 50 alerts
- `GET /api/alerts/count` - Total count
- `GET /api/alerts/active/count` - Active alerts

**Other**
- `GET /` - Web dashboard
- `GET /health` - Health check
- `GET /ws` - WebSocket for live updates

## Alert Types

**Pod Issues**
- CrashLoopBackOff (Critical)
- OOMKilled (Critical)
- ImagePullBackOff (High)
- High CPU/Memory usage
- Excessive restarts

**Node Issues**
- NotReady (Critical)
- Memory/Disk pressure (High)
- High CPU/Memory usage

## Docker Deployment

```bash
# Build and run
docker build -t monitoring-tool .
docker run -d -p 8080:8080 \
  -e POSTGRES_PASSWORD=yourpass \
  -v ~/.kube:/root/.kube:ro \
  monitoring-tool
```

## Kubernetes Deployment

```bash
# Create secrets
kubectl create secret generic monitoring-secrets \
  --from-literal=postgres-password=yourpass

# Deploy
kubectl apply -f k8s/

# Set K8S_IN_CLUSTER=true when running inside cluster
```

## Project Structure

```
monitoring-tool/
├── cmd/monitoring-tool/    # Main application
├── internal/
│   ├── api/               # HTTP handlers
│   ├── collector/         # K8s watchers
│   ├── processor/         # Alert engine
│   ├── storage/           # Database
│   └── websocket/         # Real-time updates
├── web/static/            # Web UI
├── configs/config.yaml    # Default config
└── docker-compose.yml     # Local setup
```

## Development

```bash
make test              # Run tests
make build             # Build binary
make docker-up         # Start with Docker
```

## Troubleshooting

**Can't connect to database**
```bash
docker ps | grep postgres
env | grep POSTGRES
```

**Can't access Kubernetes**
```bash
kubectl cluster-info
kubectl auth can-i get pods
```

**Email not working**
- Set `EMAIL_ENABLED=true`
- Use App Password for Gmail
- Check `LOG_LEVEL=debug` for errors

## Tech Stack

Go 1.24, Gin, PostgreSQL, Kubernetes Client-Go, WebSockets, Alpine.js

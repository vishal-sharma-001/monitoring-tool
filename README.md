# Kubernetes Monitoring Tool

Production-ready Kubernetes monitoring with real-time alerts, PostgreSQL persistence, and WebSocket updates.

## Features

- Real-time Kubernetes monitoring (Pods, Nodes, Metrics)
- Rule-based alert engine with configurable thresholds
- WebSocket push notifications + optional email alerts
- PostgreSQL with automatic migrations
- RESTful API + responsive web dashboard
- Environment-driven configuration (12-factor app)
- Docker and Kubernetes ready

## Quick Start

**Docker Compose (Recommended)**
```bash
docker-compose up -d
open http://localhost:8080
```

**Local Development**

Requirements: Go 1.24+, PostgreSQL 13+, kubeconfig access

```bash
# Build and run
make build
cp .env.example .env  # Configure if needed
POSTGRES_PASSWORD=postgres ./bin/monitoring-tool
```

## Configuration

Layered system: `config.yaml` → `.env` → `environment variables`

**Essential Variables**
```bash
# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_PASSWORD=postgres
POSTGRES_DB=monitoring_db

# Kubernetes
K8S_IN_CLUSTER=false        # true for in-cluster
KUBECONFIG=~/.kube/config

# Alert Thresholds (defaults shown)
ALERT_POD_CPU_THRESHOLD=80
ALERT_POD_MEMORY_THRESHOLD=85
ALERT_NODE_CPU_THRESHOLD=80
ALERT_NODE_MEMORY_THRESHOLD=85

# Email (optional)
EMAIL_ENABLED=false
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_PASSWORD=your-app-password
SMTP_TO=team@example.com
```

See [.env.example](.env.example) for all 30+ variables.

## Architecture

```
K8s Cluster → Collectors → Alert Engine → Event Bus
                                              ↓
                         ┌────────────────────┼────────────────────┐
                         ↓                    ↓                    ↓
                   PostgreSQL           WebSocket Hub        Email Notifier
                                              ↓
                                          Web UI
```

## API Reference

**Alerts**
- `GET /api/alerts/recent` - Last 50 alerts
- `GET /api/alerts/count` - Total count
- `GET /api/alerts/active/count` - Active alerts
- `GET /api/alerts/severity/counts` - Counts by severity

**Health**
- `GET /health` - Health check
- `GET /api/info` - System info
- `GET /ws` - WebSocket connection
- `GET /` - Web UI

**Example Response**
```json
{
  "alerts": [{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "firing",
    "severity": "critical",
    "message": "Pod web-app-7d8f9c-xyz is in CrashLoopBackOff",
    "source": "k8s_pod",
    "labels": {"namespace": "production", "pod": "web-app-7d8f9c-xyz"},
    "triggered_at": "2025-12-07T13:30:00Z"
  }]
}
```

## Alert Rules

**Pod Monitoring**
- CrashLoopBackOff (Critical), OOMKilled (Critical)
- ImagePullBackOff (High), Pending >5min (Medium)
- High restarts (High), High CPU/Memory (Medium/High)

**Node Monitoring**
- NotReady (Critical), NetworkUnavailable (Critical)
- Memory/Disk/PID Pressure (High/Medium)
- High CPU/Memory (High)

Default thresholds: 80% CPU, 85% Memory, 3 restarts

## Development

**Testing**
```bash
make test                # Run all tests
make test-coverage       # With coverage report
```

**Building**
```bash
make build               # Development build
make clean && make build # Clean build
```

**Database:** Auto-migrates on startup (GORM). Manual migrations: `make migrate-up`

## Deployment

**Docker**
```bash
docker build -t monitoring-tool:latest .
docker run -d -p 8080:8080 \
  -e POSTGRES_PASSWORD=yourpass \
  -v ~/.kube:/root/.kube:ro \
  monitoring-tool:latest
```

**Kubernetes**
```bash
kubectl create secret generic monitoring-secrets \
  --from-literal=postgres-password=yourpass
kubectl apply -f k8s/
```

Set `K8S_IN_CLUSTER=true` for in-cluster authentication.

## Project Structure

```
monitoring-tool/
├── cmd/monitoring-tool/    # Entry point (main.go, init.go)
├── internal/
│   ├── api/               # HTTP handlers & routes
│   ├── collector/         # K8s watchers (pods, nodes, metrics)
│   ├── processor/         # Alert engine & event bus
│   ├── repository/        # Data access layer
│   ├── storage/           # Database & migrations
│   └── websocket/         # Real-time hub
├── web/static/            # UI (Alpine.js + Tailwind)
├── configs/config.yaml    # Default config
├── docker-compose.yml     # Local stack
└── .env.example          # Config template
```

## Tech Stack

Go 1.24, Gin, GORM, PostgreSQL 16, Kubernetes Client-Go, Gorilla WebSocket, Zerolog, Alpine.js, Tailwind CSS

## Troubleshooting

**Database connection fails**
```bash
docker ps | grep postgres                    # Check if running
psql -h localhost -p 5432 -U postgres -d monitoring_db
env | grep POSTGRES                          # Verify env vars
```

**Kubernetes connection issues**
```bash
kubectl cluster-info                         # Verify access
kubectl auth can-i get pods                  # Check permissions
kubectl top nodes                            # Test metrics-server
```

**WebSocket not connecting**
- Check port 8080 is accessible
- Verify no firewall blocking
- Check browser console for errors

**Email not sending**
- Set `EMAIL_ENABLED=true`
- Use App Password for Gmail
- Check SMTP port (587/465) is allowed
- Enable debug logging: `LOG_LEVEL=debug`

## Performance

- Worker pool for concurrent processing
- WebSocket: 500-message buffer, per-client mutex
- Event Bus: 200-event buffer, parallel observers
- Database: 25 max connections, 5 idle
- Alert evaluation: Every 60s (configurable)

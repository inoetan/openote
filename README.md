# Openote

Rundeck-like visual job scheduler — define, schedule, and monitor jobs through a clean web UI.

## Tech Stack

| Layer       | Technology              |
|-------------|-------------------------|
| API         | Go 1.22                 |
| Frontend    | React + TypeScript      |
| Database    | PostgreSQL 16           |
| Queue/Cache | Redis 7                 |
| Proxy       | nginx                   |
| Container   | Docker / Docker Compose |

## Quick Start

```bash
# Clone and start the full stack
git clone https://github.com/your-org/openote.git
cd openote
docker compose -f infra/docker-compose.yml up
```

The app will be available at <http://localhost>.

### Development (hot-reload)

```bash
docker compose \
  -f infra/docker-compose.yml \
  -f infra/docker-compose.dev.yml \
  up
```

## Architecture

Openote is split into four components that each live in their own directory:

```
openote/
├── api/          # Go REST + WebSocket API server
├── worker/       # Go background job executor
├── frontend/     # React single-page application
└── infra/        # Docker, nginx, and CI/CD configs
```

| Component    | Responsibility                                                  |
|--------------|-----------------------------------------------------------------|
| **api**      | Exposes REST endpoints and WebSocket streams; owns auth (JWT)   |
| **worker**   | Picks up jobs from Redis queues and executes them               |
| **frontend** | Visual scheduler UI — job definitions, triggers, execution logs |
| **infra**    | Compose stacks, Dockerfiles, nginx configs, GitHub Actions      |

## Configuration

All runtime configuration is passed through environment variables:

| Variable      | Description                          | Default (dev)                                      |
|---------------|--------------------------------------|----------------------------------------------------|
| `DB_DSN`      | PostgreSQL connection string         | `postgres://openote:openote@postgres:5432/openote` |
| `REDIS_URL`   | Redis connection URL                 | `redis://redis:6379`                               |
| `JWT_SECRET`  | Secret used to sign JWT tokens       | `changeme-in-production`                           |

## Docs

Additional documentation lives in [`docs/`](docs/).

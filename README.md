# Go Cloud IDE

The system provisions a dedicated `code-server` container per workspace with a persistent Docker volume for isolation and durability. Each workspace gets a unique URL, with a routing layer that keeps access stable through the app. A background worker handles idle cleanup, and container-level CPU and memory limits keep usage fair.

## Setup

- Make sure Docker is running
- Run `docker compose up --build`
- Open `http://localhost:8090`
- Workspace password: `dev123`

## What I Built

- Go API to create, list, and delete workspaces
- Small HTMX UI to launch and view active workspaces
- Docker-managed `code-server` containers with isolated volumes
- SQLite's storage for workspace metadata
- Idle cleanup worker that removes workspaces after 30 minutes

## API

- `POST /workspaces`
- `GET /workspaces`
- `DELETE /delete?id=<id>`
- `GET /ws/<id>`

## Trade-offs

- Kept the architecture simple with one service and basic routing
- Used polling and redirects instead of realtime session management
- Fixed resource limits and one workspace image for speed of delivery
- Minimal auth: shared workspace password only

## With More Time

- Add proper auth and per-user workspaces
- Clean up Docker volumes on deletion
- Support configurable images, limits, and timeout settings

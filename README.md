# Go Cloud IDE

The system provisions a dedicated `code-server` container per workspace with a persistent Docker volume for isolation and durability. Each workspace gets a unique URL, with a routing layer that keeps access stable through the app. A background worker handles idle cleanup, and container-level CPU and memory limits keep usage fair.

## Setup

- Make sure Docker is running
- Run `docker compose up --build`
- Open `http://localhost:8090`
- Workspace password: `dev123`

## What I Built

- Go API to create, list, start, stop, restart, and delete workspaces
- HTMX + Bootstrap UI to launch and manage workspaces
- Docker-managed `code-server` containers with isolated volumes
- SQLite storage for workspace metadata with status tracking
- Idle cleanup worker that stops inactive workspaces after 30 minutes
- Startup reconciler that syncs database state with Docker containers
- Heartbeat endpoint to keep workspaces active during use

## API

- `POST /workspaces` - Create a new workspace
- `GET /workspaces` - List all workspaces
- `POST /workspaces/start?id=<id>` - Start a stopped workspace
- `POST /workspaces/stop?id=<id>` - Stop a running workspace
- `POST /workspaces/restart?id=<id>` - Restart a workspace
- `DELETE /delete?id=<id>` - Delete a workspace
- `GET /heartbeat?id=<id>` - Update last active timestamp
- `GET /ws/<id>` - Proxy to workspace code-server
- `GET /` - Web UI
- `GET /ui/workspaces` - Workspace list fragment (HTMX)

## Trade-offs

- Kept the architecture simple with one service and basic routing
- Used polling and redirects instead of realtime session management
- Fixed resource limits and one workspace image for speed of delivery
- Minimal auth: shared workspace password only

## With More Time

- Add proper auth and per-user workspaces with access control
- Clean up Docker volumes when deleting workspaces
- Support configurable images, CPU/memory limits, and idle timeout settings
- Add WebSocket-based real-time status updates instead of polling
- Implement workspace auto-save and backup features
- Add metrics and monitoring dashboard

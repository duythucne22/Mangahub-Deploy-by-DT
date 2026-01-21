# Mangahub Testing Guide (Manual + Automation)

This guide documents manual API tests and basic automation checks for the HTTP/gRPC/WebSocket/UDP/TCP stack and the TUI app.

## 1) Pre-reqs
- Server running (HTTP + gRPC + TCP + UDP).
- Database schema applied and seed loaded from [seed.sql](seed.sql).
- TUI config file (optional): [configs/tui.yaml](configs/tui.yaml) or default.

## 2) Manual API Tests (HTTP)

### Auth
- Register: POST /api/v1/auth/register
- Login: POST /api/v1/auth/login

Expected: token in response data.

### Manga Browse/Search/Trending
- List: GET /api/v1/manga?page=1&limit=20
- Filter by genre: GET /api/v1/manga?genre=action
- Filter by status: GET /api/v1/manga?status=ongoing
- Search: GET /api/v1/manga/search?q=one%20piece
- Trending: GET /api/v1/manga/trending?limit=10

Expected: list results with pagination, filters applied, and trending ordered by weekly score.

### Manga Create (Admin)
- POST /api/v1/manga

Expected:
- HTTP 201
- Activity log (type: manga_update)
- UDP notification broadcast
- TCP stats event emitted

### Comments
- Create: POST /api/v1/manga/:id/comments
- Like: POST /api/v1/manga/:id/comments/:comment_id/like
- List: GET /api/v1/manga/:id/comments

Expected:
- Activity log (comment)
- TCP stats event emitted

## 3) gRPC Search (Streaming)
Use `StreamSearch` with FTS query. Expected to stream results and use `search_vector`.

## 4) WebSocket Chat
Connect to:
- ws://<host>:<port>/ws/manga/:manga_id?token=<JWT>

Expected:
- Auth validated
- Room join/leave messages
- Persistence to `chat_messages`
- TCP stats events for chat

## 5) UDP Notifications
Listen on UDP port (default 4000). Trigger manga create to receive a notification.

Expected:
- UDP broadcast packet
- Notification logged into `notifications` table

## 6) TCP Stats
Send framed JSON event to TCP port (default 6000):

Event types allowed: `comment`, `chat`, `manga_update`.

Expected:
- `manga_stats` updated
- Activity logged unless source is `http`

## 7) TUI Smoke Test

### Setup
- Confirm TUI config: [configs/tui.yaml](configs/tui.yaml) or use defaults.
- Ensure server addresses match config in [internal/tui/config/config.go](internal/tui/config/config.go).

### Actions
- Launch TUI and login
- Browse, search, view detail
- Enter chat view and send a message
- View stats and lists

Expected:
- All views load without errors
- Search uses gRPC if available, otherwise HTTP
- Chat connects and persists messages

## 8) Automation (Basic)

- `go test ./...` (unit checks)
- Add targeted tests to cover:
  - `MangaService.List` filters
  - `UDP` broadcast path
  - `TCP` event processing
  - TUI API client calls

## 9) SQL Debug Helpers

Use `psql` inside docker to validate data changes in:
- `manga`
- `manga_genres`
- `activity_feed`
- `notifications`
- `manga_stats`
- `chat_messages`


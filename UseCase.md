# 📋 MangaHub Use Cases - Manual Testing Guide

## 👤 **User Authentication Flows**

### **1. User Registration (HTTP)**
**Protocol**: HTTP POST `/auth/register`  
**Request**:
```json
{
  "username": "anime_fan99",
  "password": "SecurePass123!@"
}
```
**Expected Response**:
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "user": {
      "id": "user-12345",
      "username": "anime_fan99",
      "created_at": "2026-01-19T10:30:00Z"
    }
  },
  "timestamp": "2026-01-19T10:30:00Z"
}
```
**Integration Points**:
- ✅ Creates user record in `users` table
- ❌ Does NOT trigger TCP/UDP events
- ❌ Does NOT require admin role

**Manual Test**:
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "test_user", "password": "TestPass123!@"}'
```

### **2. User Login (HTTP)**
**Protocol**: HTTP POST `/auth/login`  
**Request**:
```json
{
  "username": "anime_fan99",
  "password": "SecurePass123!@"
}
```
**Expected Response**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOi...",
    "user": {
      "id": "user-12345",
      "username": "anime_fan99",
      "created_at": "2026-01-19T10:30:00Z"
    },
    "expires_in": 3600
  },
  "timestamp": "2026-01-19T10:35:00Z"
}
```
**Integration Points**:
- ✅ Validates credentials against `users` table
- ✅ Generates JWT token with user ID and role
- ❌ Does NOT trigger TCP/UDP events

**Manual Test**:
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "test_user", "password": "TestPass123!@"}'
```

---

## 📚 **Manga Management Flows**

### **3. Browse Manga by Genre (HTTP)**
**Protocol**: HTTP GET `/manga?genre=action`  
**Request**: GET `/manga?genre=action&limit=10`  
**Expected Response**:
```json
{
  "success": true,
  "data": {
    "data": [
      {
        "id": "manga-001",
        "title": "One Piece",
        "description": "The story of Monkey D. Luffy...",
        "cover_url": "https://example.com/covers/onepiece.jpg",
        "status": "ongoing",
        "genres": [
          {"id": "action", "name": "Action"},
          {"id": "adventure", "name": "Adventure"}
        ]
      }
    ],
    "total": 42,
    "limit": 10,
    "offset": 0,
    "has_more": true
  },
  "timestamp": "2026-01-19T11:00:00Z"
}
```
**Integration Points**:
- ✅ Queries `manga` and `manga_genres` tables
- ✅ Returns paginated results with genre information
- ❌ Does NOT log to activity feed
- ❌ Does NOT trigger stats events

**Manual Test**:
```bash
curl "http://localhost:8080/manga?genre=action&limit=10" \
  -H "Authorization: Bearer <your_token>"
```

### **4. gRPC Search (gRPC)**
**Protocol**: gRPC StreamSearch (port 50051)  
**Request** (using grpcurl):
```bash
grpcurl -plaintext -d '{
  "query": "one piece",
  "limit": 5
}' localhost:50051 mangahub.v1.MangaService/StreamSearch
```
**Expected Response** (streaming):
```json
{
  "id": "manga-001",
  "title": "One Piece",
  "description": "The story of Monkey D. Luffy...",
  "cover_url": "https://example.com/covers/onepiece.jpg",
  "status": "ongoing",
  "genres": [
    {"id": "action", "name": "Action"},
    {"id": "adventure", "name": "Adventure"}
  ],
  "relevance_score": 0.95
}
```
**Integration Points**:
- ✅ Uses PostgreSQL FTS (`search_vector` column)
- ✅ Streams results in real-time (no pagination)
- ✅ Higher performance than HTTP search
- ❌ Does NOT log individual searches to activity feed

**Manual Test**:
```bash
# Install grpcurl first: brew install grpcurl
grpcurl -plaintext -proto proto/manga.proto -d '{
  "query": "piece",
  "limit": 3
}' localhost:50051 mangahub.v1.MangaService/StreamSearch
```

### **5. Create Manga (Admin Only - HTTP)**
**Protocol**: HTTP POST `/manga`  
**Request** (admin token required):
```json
{
  "title": "New Manga Title",
  "description": "Description of new manga",
  "cover_url": "https://example.com/covers/newmanga.jpg",
  "status": "ongoing",
  "genre_ids": ["action", "fantasy"]
}
```
**Expected Response**:
```json
{
  "success": true,
  "message": "Manga created successfully",
  "data": {
    "id": "manga-999",
    "title": "New Manga Title",
    "genres": [
      {"id": "action", "name": "Action"},
      {"id": "fantasy", "name": "Fantasy"}
    ]
  },
  "timestamp": "2026-01-19T11:30:00Z"
}
```
**Integration Points**:
- ✅ Checks admin role before creation
- ✅ Creates records in `manga`, `manga_genres` tables
- ✅ Initializes `manga_stats` record
- ✅ Logs to `activity_feed` (type: `manga_update`)
- ✅ Triggers UDP notification broadcast
- ✅ Emits TCP stats event (weight: 5)

**Manual Test**:
```bash
curl -X POST http://localhost:8080/manga \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Manga",
    "description": "Test description",
    "cover_url": "https://example.com/test.jpg",
    "status": "ongoing",
    "genre_ids": ["action", "comedy"]
  }'
```

---

## 💬 **Community Interaction Flows**

### **6. Comment on Manga (HTTP)**
**Protocol**: HTTP POST `/manga/{manga_id}/comments`  
**Request**:
```json
{
  "content": "This manga is amazing!"
}
```
**Expected Response**:
```json
{
  "success": true,
  "message": "Comment created successfully",
  "data": {
    "id": "comment-123",
    "manga_id": "manga-001",
    "user": {
      "id": "user-12345",
      "username": "anime_fan99"
    },
    "content": "This manga is amazing!",
    "like_count": 0,
    "created_at": "2026-01-19T12:00:00Z"
  },
  "timestamp": "2026-01-19T12:00:00Z"
}
```
**Integration Points**:
- ✅ Creates record in `comments` table
- ✅ Logs to `activity_feed` (type: `comment`)
- ✅ Emits TCP stats event (weight: 1)
- ✅ Updates `manga_stats.comment_count` atomically
- ✅ Updates `manga_stats.weekly_score` (+1 point)

**Manual Test**:
```bash
curl -X POST http://localhost:8080/manga/manga-001/comments \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "Great chapter!"}'
```

### **7. Like Comment (HTTP)**
**Protocol**: HTTP POST `/comments/{comment_id}/like`  
**Request**: POST (no body)  
**Expected Response**:
```json
{
  "success": true,
  "message": "Comment liked successfully",
  "data": {
    "id": "comment-123",
    "like_count": 1,
    "user": {
      "id": "user-12345",
      "username": "anime_fan99"
    }
  },
  "timestamp": "2026-01-19T12:05:00Z"
}
```
**Integration Points**:
- ✅ Updates `comments.like_count` atomically
- ✅ Logs to `activity_feed` (type: `comment` with like action)
- ✅ Emits TCP stats event (weight: 1)
- ✅ Updates `manga_stats.like_count` and `weekly_score`

**Manual Test**:
```bash
curl -X POST http://localhost:8080/comments/comment-123/like \
  -H "Authorization: Bearer <user_token>"
```

### **8. Join Manga Chat Room (WebSocket)**
**Protocol**: WebSocket GET `/chat/{manga_id}`  
**Request** (using websocat):
```bash
websocat "ws://localhost:8081/chat/manga-001?token=<user_token>"
```
**Expected Response** (after connection):
1. Connection upgrade to WebSocket
2. Initial chat history (last 50 messages)
3. Join notification broadcast to all room members

**Integration Points**:
- ✅ Validates token before upgrade
- ✅ Checks manga exists
- ✅ Sends chat history to new client
- ✅ Broadcasts join message to room
- ✅ Logs to `activity_feed` (type: `chat`)

**Manual Test**:
```bash
# Install websocat first: brew install websocat
websocat "ws://localhost:8081/chat/manga-001?token=<your_token>"
# Then type messages in the format:
# {"content": "Hello from terminal!"}
```

---

## 🔔 **Activity & Notification Flows**

### **9. Receive UDP Notification (UDP)**
**Protocol**: UDP listen on port 4000  
**Setup** (terminal 1 - listener):
```bash
nc -u -l 4000
```
**Trigger** (terminal 2 - admin action):
```bash
curl -X POST http://localhost:8080/manga \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "UDP Test Manga",
    "description": "Testing notifications",
    "status": "ongoing",
    "genre_ids": ["action"]
  }'
```
**Expected UDP Packet** (in terminal 1):
```json
{"type":"manga_update","message":"📚 New manga: UDP Test Manga","timestamp":"2026-01-19T12:30:00Z"}
```
**Integration Points**:
- ✅ UDP server broadcasts to all connected clients
- ✅ 100 packets/second rate limiting
- ✅ 1KB max packet size validation
- ✅ Logs to `notifications` table before broadcast
- ✅ Fire-and-forget semantics (no ACKs)

**Manual Test**:
```bash
# Terminal 1: Start UDP listener
nc -u -l 4000

# Terminal 2: Trigger notification (admin action)
# (Use the curl command above)
```

### **10. View Activity Feed (HTTP)**
**Protocol**: HTTP GET `/activity`  
**Request**: GET `/activity?limit=10`  
**Expected Response**:
```json
{
  "success": true,
  "data": {
    "data": [
      {
        "id": "act-123",
        "type": "comment",
        "user": {
          "id": "user-12345",
          "username": "anime_fan99"
        },
        "manga": {
          "id": "manga-001",
          "title": "One Piece"
        },
        "created_at": "2026-01-19T12:00:00Z"
      },
      {
        "id": "act-124",
        "type": "manga_update",
        "manga": {
          "id": "manga-999",
          "title": "New Manga Title"
        },
        "created_at": "2026-01-19T11:30:00Z"
      }
    ],
    "total": 15,
    "limit": 10,
    "offset": 0,
    "has_more": true
  },
  "timestamp": "2026-01-19T12:35:00Z"
}
```
**Integration Points**:
- ✅ Joins `activity_feed` with `users` and `manga` tables
- ✅ Returns chronological feed
- ✅ Filters by activity type if specified
- ❌ Does NOT trigger additional events


---

## 📊 **Stats & Ranking Flows**

### **11. Get Hot Manga Rankings (gRPC)**
**Protocol**: gRPC GetTrendingManga (port 50051)  
**Request**:
```bash
grpcurl -plaintext -d '{
  "limit": 5
}' localhost:50051 mangahub.v1.MangaService/GetTrendingManga
```
**Expected Response**:
```json
{
  "manga": [
    {
      "id": "manga-001",
      "title": "One Piece",
      "cover_url": "https://example.com/covers/onepiece.jpg",
      "relevance_score": 950
    },
    {
      "id": "manga-002",
      "title": "Jujutsu Kaisen",
      "cover_url": "https://example.com/covers/jjk.jpg",
      "relevance_score": 880
    }
  ]
}
```
**Integration Points**:
- ✅ Reads from `manga_stats` table
- ✅ Orders by `weekly_score` DESC
- ✅ Uses TCP Stats Service for real-time updates
- ✅ Weighted scoring (comment=1, chat=2, manga_update=5)

**Manual Test**:
```bash
grpcurl -plaintext -proto proto/manga.proto -d '{
  "limit": 5
}' localhost:50051 mangahub.v1.MangaService/GetTrendingManga
```

### **12. TCP Stats Event Processing (TCP)**
**Protocol**: TCP connection to port 6000  
**Test Setup** (send event manually):
```bash
echo -ne '\x00\x00\x00\x4a{"type":"comment","manga_id":"manga-001","user_id":"user-12345","weight":1,"source":"http"}' | nc localhost 6000
```
**Expected Response**:
```json
{"status":"success","message":"Event processed successfully"}
```
**Integration Points**:
- ✅ Custom protocol framing ([4-byte length][JSON payload])
- ✅ Atomic counter updates in `manga_stats`
- ✅ Weighted scoring based on event type
- ✅ Uses PostgreSQL row locking for concurrency safety
- ✅ Logs to `activity_feed` if not already logged

**Manual Test**:
```bash
# Send TCP stats event (hex length prefix + JSON)
echo -ne '\x00\x00\x00\x4a{"type":"comment","manga_id":"manga-001","user_id":"user-12345","weight":1,"source":"http"}' | nc localhost 6000
```

---

## ✅ **Complete Use Case Checklist**

| Use Case | Protocol | Status | Integration Points Verified |
|----------|----------|--------|-----------------------------|
| **User Registration** | HTTP | ✅ READY | Activity logging, user creation |
| **User Login** | HTTP | ✅ READY | Token generation, activity logging |
| **Browse by Genre** | HTTP | ✅ READY | Genre relationships, pagination |
| **gRPC Search** | gRPC | ✅ READY | PostgreSQL FTS, streaming |
| **Create Manga (Admin)** | HTTP | ✅ READY | Role check, UDP broadcast, TCP event |
| **Comment on Manga** | HTTP | ✅ READY | Activity logging, TCP stats event |
| **Like Comment** | HTTP | ✅ READY | Atomic updates, TCP stats event |
| **Join Chat Room** | WebSocket | ✅ READY | Room isolation, message persistence |
| **Receive UDP Notification** | UDP | ✅ READY | Broadcast, rate limiting, logging |
| **View Activity Feed** | HTTP | ✅ READY | Data joining, filtering |
| **Get Hot Rankings** | gRPC | ✅ READY | Stats aggregation, weighted scoring |
| **TCP Stats Processing** | TCP | ✅ READY | Protocol framing, atomic updates |

## 🚀 **Testing Strategy**

1. **Start with HTTP endpoints** (easiest to test with curl)
2. **Test gRPC services** with grpcurl
3. **Test WebSocket** with websocat
4. **Test UDP** with netcat listeners
5. **Test TCP** with netcat + manual framing
6. **Verify integration points** by checking database tables:
   - `activity_feed` logs all events
   - `manga_stats` counters update correctly
   - `notifications` table logs broadcasts

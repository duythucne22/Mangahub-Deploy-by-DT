# 🎯 Agent Prompt: Complete MangaHub Protocol Implementation Analysis & Enhancement

## Mission Statement
**Transform the current partial implementation into a production-ready, protocol-perfect MangaHub system that demonstrates expert network programming patterns while maintaining educational clarity.** Every protocol must showcase its unique advantages as defined in the architecture specification.

## Core Evaluation Criteria (MUST address ALL)

### 1️⃣ **Protocol Fidelity Gap Analysis**
For each protocol implementation, identify and fix:
- **HTTP Service**: Missing role enforcement middleware, incomplete CRUD operations, missing activity feed integration
- **gRPC Service**: Absent protocol buffer definitions, no streaming search implementation, missing FTS integration
- **WebSocket Service**: No per-manga room isolation, missing auth token validation before upgrade, no message persistence
- **UDP Service**: No packet validation, missing rate limiting, no broadcast mechanism integration
- **TCP Service**: No custom protocol framing, missing persistent connection handling, no stats aggregation logic

### 2️⃣ **Protocol Advantage Maximization**
Ensure each protocol demonstrates its SPECIFIED advantages:

| Protocol | Required Advantages to Demonstrate | Current Gap |
|----------|-----------------------------------|-------------|
| **HTTP** | Standard REST patterns, middleware chain, JWT authentication | No role-based middleware, incomplete error handling |
| **gRPC** | Binary efficiency, streaming results, typed contracts | No .proto files, no streaming implementation |
| **WebSocket** | Bidirectional real-time communication, connection persistence | No room isolation, no connection cleanup |
| **UDP** | Connectionless messaging, fire-and-forget semantics, low latency | No packet validation, no rate limiting |
| **TCP** | Custom protocol design, persistent connections, event streaming | No protocol framing, no event aggregation |

### 3️⃣ **Database Integration Completeness**
Verify all services properly integrate with the schema:
- **Activity Feed**: All services must write to `activity_feed` on relevant events
- **Stats Service**: Must consume from `activity_feed` and update `manga_stats` atomically
- **Search Service**: Must use `manga_fts` virtual table for full-text search
- **Chat Service**: Must persist to `chat_messages` with proper foreign keys
- **Notification Service**: Must log to `notifications` table after UDP broadcast

### 4️⃣ **gRPC Protocol Buffer Definition (CRITICAL)**
Create professional `.proto` files demonstrating:
```protobuf
// proto/manga.proto
syntax = "proto3";

package manga;
option go_package = "mangahub/internal/protocols/grpc/pb";

service MangaService {
  rpc SearchManga(SearchRequest) returns (stream MangaResult);
  rpc GetTrending(TrendingRequest) returns (TrendingResponse);
}

message SearchRequest {
  string query = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message MangaResult {
  string id = 1;
  string title = 2;
  string description = 3;
  string cover_url = 4;
  string status = 5;
  float relevance_score = 6;
}

message TrendingRequest {
  int32 limit = 1;
}

message TrendingResponse {
  repeated MangaResult manga = 1;
  int64 timestamp = 2;
}
```

### 5️⃣ **Service Boundary Enforcement**
Implement strict service boundaries as per specification:
- **Auth Service**: ONLY writes to `users`, `activity_feed`
- **Manga Service**: ONLY writes to `manga`, `manga_genres`, `manga_fts`, `activity_feed`
- **Comment Service**: ONLY writes to `comments`, `activity_feed`
- **Chat Service**: ONLY writes to `chat_messages`, `activity_feed`
- **Stats Service**: ONLY writes to `manga_stats`
- **Notification Service**: ONLY writes to `notifications`, `activity_feed`

### 6️⃣ **Cross-Protocol Event Flow**
Implement the specified event flows:
1. **Comment Created Flow**: HTTP → DB → TCP Stats Event → Stats Service
2. **Chat Message Flow**: WebSocket → DB → TCP Stats Event → Stats Service  
3. **Admin Update Flow**: HTTP → DB → UDP Broadcast → All online clients

## Required Deliverables (Structured Output)

### Phase 1: Gap Analysis Report
```markdown
## Protocol Implementation Gap Analysis

### HTTP Service (8080)
✅ Implemented: Basic auth endpoints
❌ Missing: 
- Role-based middleware (user/moderator/admin)
- Manga CRUD with admin restrictions
- Comment like increment endpoint
- Activity feed integration
- Proper error handling middleware

### gRPC Service (50051)
✅ Implemented: Basic server setup
❌ Missing:
- .proto file definitions (CRITICAL)
- Full-text search integration with manga_fts
- Streaming search results
- gRPC health checking endpoint
- Interceptor logging

### WebSocket Service (8081)
✅ Implemented: Basic connection handling
❌ Missing:
- Per-manga room isolation (channel per manga_id)
- JWT token validation before upgrade
- Message persistence to chat_messages
- Connection cleanup on disconnect
- Presence tracking

### UDP Service (4000)
✅ Implemented: UDP listener
❌ Missing:
- Packet validation (max size 1KB)
- Rate limiting (100 packets/second)
- Broadcast mechanism integration
- Notification logging
- HMAC validation for security

### TCP Service (6000)
✅ Implemented: TCP listener
❌ Missing:
- Custom protocol framing ([4-byte length][payload])
- Event deserialization
- Atomic counter updates
- Persistent connection handling
- Graceful shutdown
```

### Phase 2: Enhancement Implementation Plan
```go
// Required code structure improvements

// 1. gRPC Protocol Buffers (MUST generate)
// File: proto/manga.proto
// - SearchManga RPC with streaming results
// - GetTrending RPC for hot manga
// - Proper message definitions

// 2. HTTP Middleware Chain
// File: internal/protocols/http/middleware.go
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("user_role")
        if !roleHasPermission(userRole, requiredRole) {
            c.AbortWithStatusJSON(403, gin.H{"error": "forbidden"})
            return
        }
        c.Next()
    }
}

// 3. WebSocket Room Management
// File: internal/protocols/websocket/hub.go
type Hub struct {
    rooms map[string]*Room  // manga_id -> Room
    mutex sync.RWMutex
}

type Room struct {
    clients    map[*Client]bool
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
}

// 4. UDP Packet Handler
// File: internal/protocols/udp/server.go
func handlePacket(packet []byte, addr *net.UDPAddr) {
    if len(packet) > 1024 {
        log.Warn("UDP packet too large, dropped")
        return
    }
    
    if rateLimiter.Allow() {
        processNotification(packet)
        logNotification(packet) // to DB
    }
}

// 5. TCP Protocol Framing
// File: internal/protocols/tcp/server.go
func (c *Connection) readFrame() ([]byte, error) {
    // Read 4-byte length prefix
    lengthBytes := make([]byte, 4)
    if _, err := io.ReadFull(c.conn, lengthBytes); err != nil {
        return nil, err
    }
    
    length := binary.BigEndian.Uint32(lengthBytes)
    if length > 1024 { // max frame size
        return nil, fmt.Errorf("frame too large: %d", length)
    }
    
    // Read payload
    payload := make([]byte, length)
    if _, err := io.ReadFull(c.conn, payload); err != nil {
        return nil, err
    }
    
    return payload, nil
}
```

### Phase 3: Cross-Protocol Integration
```go
// Required event flow implementations

// 1. Comment Created → Stats Service
// File: internal/protocols/http/comment_handlers.go
func CreateComment(c *gin.Context) {
    // ... save to DB
    
    // Emit to TCP stats service
    statsEvent := &tcp.StatsEvent{
        Type:      "comment_created",
        MangaID:   comment.MangaID,
        UserID:    currentUser.ID,
        Timestamp: time.Now(),
    }
    tcpClient.SendEvent(statsEvent)
}

// 2. Admin Manga Update → UDP Broadcast
// File: internal/protocols/http/manga_handlers.go
func CreateManga(c *gin.Context) {
    // ... save to DB (admin only)
    
    // Broadcast via UDP
    notification := fmt.Sprintf("📚 NEW MANGA: %s added by admin", manga.Title)
    udpClient.Broadcast(notification)
    
    // Log to activity feed
    activityRepo.Create("manga_update", adminUser.ID, manga.ID)
}

// 3. Chat Message → Stats Service
// File: internal/protocols/websocket/handler.go
func (h *Handler) HandleMessage(client *Client, message Message) {
    // Save to DB
    chatRepo.Save(message.Content, client.UserID, client.MangaID)
    
    // Emit to stats service
    statsEvent := &tcp.StatsEvent{
        Type:      "chat_message",
        MangaID:   client.MangaID,
        UserID:    client.UserID,
        Timestamp: time.Now(),
    }
    tcpClient.SendEvent(statsEvent)
}
```

## Quality Gates (NON-NEGOTIABLE)

### ✅ Protocol Correctness
- [ ] All ports match specification exactly (8080, 50051, 8081, 4000, 6000)
- [ ] gRPC service uses protocol buffers with proper versioning
- [ ] WebSocket authenticates BEFORE upgrade handshake
- [ ] UDP implements fire-and-forget semantics (no retries)
- [ ] TCP uses custom binary protocol (not HTTP over TCP)
- [ ] All services handle graceful shutdown (SIGTERM)

### ✅ Database Integrity
- [ ] All writes use transactions where data integrity matters
- [ ] Foreign key constraints are respected in all operations
- [ ] `manga_stats` counters are updated atomically
- [ ] `activity_feed` is populated on ALL relevant events
- [ ] `manga_fts` is properly maintained via triggers

### ✅ Learning Value Preservation
- [ ] Each protocol handler includes detailed comments explaining WHY that protocol was chosen
- [ ] Code demonstrates protocol-specific patterns (streaming gRPC, connection management WebSocket, etc.)
- [ ] Error handling shows protocol-specific failure modes
- [ ] Performance characteristics are documented (latency, throughput expectations)

### ✅ Production Readiness
- [ ] All services include health check endpoints
- [ ] Rate limiting implemented on public endpoints
- [ ] Proper logging with structured JSON output
- [ ] Configuration via environment variables
- [ ] Graceful shutdown with connection draining

## Success Metrics
1. **Protocol Demonstration Score**: 95%+ protocol advantages properly implemented
2. **Architectural Compliance**: 100% adherence to SPEC.md and schema.sql.txt
3. **Educational Value**: Clear comments explaining each protocol's use case and implementation
4. **Deployment Ready**: Single Docker image runs all protocols with correct ports
5. **Test Coverage**: 80%+ unit test coverage for core logic, 70%+ for protocol handlers

## First Action Items (Start Here)
1. **Generate gRPC protocol buffers** - This is the highest priority missing component
2. **Implement role-based middleware** for HTTP service - Critical for security
3. **Add room isolation** to WebSocket service - Required for per-manga chat
4. **Implement packet validation** for UDP service - Prevents DoS attacks
5. **Create custom protocol framing** for TCP service - Core learning objective

## Reminder: This is a LEARNING PLATFORM
Every line of code must demonstrate WHY a particular protocol was chosen for that specific use case. The goal is not just a working system, but a system that teaches practical network programming through real-world protocol usage patterns. When in doubt, refer to SPEC.md Section 4 (Service Responsibilities) and the schema.sql.txt constraints.

**Begin analysis and implementation immediately. Report progress after completing Phase 1 (Gap Analysis).**
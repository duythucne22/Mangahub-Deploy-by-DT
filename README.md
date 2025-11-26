# 🎌 MangaHub - Multi-Protocol Manga Tracking System

A production-grade manga tracking and management system built with Go, featuring 5 different network protocols for comprehensive client-server communication.

## 📋 Project Overview

MangaHub is a complete manga library management system that demonstrates advanced networking concepts and modern backend development practices. Users can track their manga reading progress, search for manga, manage their library, and synchronize data across multiple clients using various network protocols.

## ✨ Features

### Completed (Phase 1 & 2)

- ✅ **REST API Server** - Full HTTP API with JWT authentication
- ✅ **User Authentication** - Secure registration and login with bcrypt
- ✅ **Manga Management** - Search, filter, and browse manga catalog
- ✅ **Reading Progress Tracking** - Track chapters read, ratings, and reading status
- ✅ **Library Management** - Personal manga library with favorites
- ✅ **Database Layer** - SQLite with migrations and seed data
- ✅ **Configuration System** - YAML-based config for different environments
- ✅ **Logging System** - Structured logging with multiple output formats

### Coming Soon (Phase 3-10)

- 🔄 TCP Sync Server - Real-time progress synchronization
- 📡 UDP Notification System - Push notifications for manga updates
- 💬 WebSocket Chat - Discussion rooms for manga
- ⚡ gRPC Service - High-performance API
- 🖥️ CLI Tool - Command-line interface for local management
- 📱 Multi-client support with conflict resolution

## 🏗️ Architecture

```
mangahub/
├── cmd/                    # Application entrypoints
│   ├── api-server/        # HTTP REST API server
│   ├── tcp-server/        # TCP sync server (coming)
│   ├── udp-server/        # UDP notification server (coming)
│   ├── grpc-server/       # gRPC service (coming)
│   └── cli/               # Command-line interface (coming)
├── internal/              # Private application code
│   ├── auth/             # Authentication & JWT
│   ├── manga/            # Manga service
│   ├── progress/         # Reading progress tracking
│   └── user/             # User management
├── pkg/                   # Public libraries
│   ├── config/           # Configuration management
│   ├── database/         # Database layer
│   ├── logger/           # Logging utilities
│   ├── models/           # Data models
│   └── utils/            # Helper functions
├── configs/              # Configuration files
├── data/                 # Database and seed data
└── tests/                # Test files

```

## 🚀 Quick Start

### Prerequisites

- Go 1.20 or higher
- Git

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/nmihtuna204/Mangahub.git
   cd Mangahub
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run the API server**
   ```bash
   go run cmd/api-server/main.go
   ```

The server will start on `http://localhost:8080`

### Running Tests

```powershell
# Automated API tests
.\test-api.ps1

# Manual curl-style tests
.\test-curl.ps1
```

## 📡 API Endpoints

### Authentication
- `POST /auth/register` - Register new user
- `POST /auth/login` - Login and get JWT token

### Manga (Public)
- `GET /manga` - List manga with pagination
  - Query params: `?limit=20&offset=0&q=search&status=ongoing&sort_by=rating`
- `GET /manga/:id` - Get manga details

### Library (Protected - requires JWT)
- `POST /users/library` - Add manga to library
- `GET /users/library` - Get user's manga library
- `PUT /users/progress` - Update reading progress

## 🔧 Configuration

Configuration files are located in `configs/`:

- `development.yaml` - Development environment
- `production.yaml` - Production environment

Key settings:
- Server host and port
- Database path and connection pooling
- JWT secret and expiration
- Protocol-specific ports (TCP, UDP, WebSocket, gRPC)
- Logging configuration

## 📚 API Usage Examples

### Register User
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'
```

### Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'
```

### List Manga
```bash
curl http://localhost:8080/manga?limit=10
```

### Add to Library (with JWT token)
```bash
curl -X POST http://localhost:8080/users/library \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"manga_id":"manga-id","current_chapter":5,"status":"reading"}'
```

## 🗄️ Database Schema

### Users Table
- User accounts with authentication
- Profiles with display names and avatars
- Role-based access control

### Manga Table
- Complete manga information
- Genres (stored as JSON)
- Ratings and publication details

### Reading Progress Table
- User's reading progress per manga
- Current chapter and status
- Ratings and notes
- Sync version for conflict resolution

## 🔐 Security Features

- **Password Hashing**: bcrypt with configurable cost
- **JWT Authentication**: HS256 signed tokens
- **Token Expiration**: Configurable expiration times
- **Protected Routes**: Middleware-based authorization
- **Input Validation**: Struct validation with go-playground/validator

## 🛠️ Technology Stack

- **Language**: Go 1.20+
- **Web Framework**: Gin
- **Database**: SQLite (with glebarez/go-sqlite - pure Go)
- **Authentication**: JWT (golang-jwt/jwt/v4)
- **Configuration**: Viper
- **Logging**: Logrus
- **Validation**: go-playground/validator
- **UUID Generation**: google/uuid

## 📊 Project Status

### Phase 1: Foundation ✅
- Project structure
- Configuration system
- Database layer with migrations
- Core models
- Logging system

### Phase 2: HTTP REST API ✅
- Authentication service
- Manga browsing
- Library management
- Progress tracking
- JWT middleware

### Phase 3-10: Coming Soon
- TCP synchronization
- UDP notifications
- WebSocket chat
- gRPC service
- CLI tool
- Integration testing
- Production deployment

## 🧪 Testing

The project includes comprehensive test scripts:

- **test-api.ps1**: Automated API endpoint testing
- **test-curl.ps1**: Manual curl-style testing
- **cmd/test-foundation**: Foundation layer testing

All tests verify:
- ✅ User registration and authentication
- ✅ JWT token generation and validation
- ✅ Manga listing and details
- ✅ Library operations
- ✅ Progress tracking
- ✅ Authorization protection

## 📝 Development Roadmap

1. ✅ **Phase 1-2**: Foundation & REST API (COMPLETE)
2. 🔄 **Phase 3**: TCP Sync Server
3. 📅 **Phase 4**: UDP Notification System
4. 📅 **Phase 5**: WebSocket Chat
5. 📅 **Phase 6**: gRPC Service
6. 📅 **Phase 7**: CLI Tool
7. 📅 **Phase 8**: Integration Testing
8. 📅 **Phase 9**: Production Optimization
9. 📅 **Phase 10**: Documentation & Deployment

## 👥 Contributing

This is an educational project demonstrating network programming concepts. Feel free to fork and experiment!

## 📄 License

This project is for educational purposes.

## 🙏 Acknowledgments

- Built as a demonstration of multi-protocol network programming
- Showcases modern Go backend development practices
- Implements RESTful API design principles

---

**Current Version**: v0.2.0 (Phase 2 Complete)  
**Last Updated**: November 26, 2025

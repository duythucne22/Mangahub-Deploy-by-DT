# 🛠️ MangaHub Configuration Specification (config.md)

## I. Service Port Mapping Matrix

| Service               | Protocol      | Container Port | Host Port | Binding Interface | Security Notes                     |
|-----------------------|---------------|----------------|-----------|-------------------|------------------------------------|
| API Gateway           | HTTPS         | 443            | 443       | 0.0.0.0           | TLS termination                    |
| Auth & Core API       | HTTP/2        | 8080           | 8080      | 127.0.0.1         | Behind gateway; mTLS in prod       |
| Search Service        | gRPC          | 50051          | 50051     | 127.0.0.1         | Service mesh mTLS                  |
| Chat Service          | WebSocket     | 8081           | 8081      | 127.0.0.1         | Pre-auth HTTP handshake            |
| UDP Notification Bus  | UDP           | 4000           | 4000      | 10.0.0.0/24       | Firewall-restricted to app servers |
| Stats Aggregator      | TCP (custom)  | 6000           | 6000      | 127.0.0.1         | Internal only; no external access  |
| PostgreSQL            | TCP           | 5432           | 5432      | 10.0.0.0/24       | VPC-only; SSL required             |
| Redis (Cache)         | TCP           | 6379           | 6379      | 127.0.0.1         | Memory-only; no persistence        |
| Prometheus            | HTTP          | 9090           | 9090      | 127.0.0.1         | Scraped by central monitoring      |
| Admin Console         | HTTPS         | 8082           | 8443      | 10.0.0.5/32       | IP-restricted; RBAC enforced       |

> **Binding Principle**: Production services bind to internal networks only. Public exposure occurs exclusively through API Gateway with TLS termination.

## II. Service-Specific Environment Configuration

### Global Configuration (All Services)
```env
# Environment Topology
APP_ENV=development  # [development|staging|production]
DEPLOY_REGION=us-east-1
CLUSTER_NAME=mangahub-prod

# Observability
LOG_LEVEL=info  # [debug|info|warn|error]
LOG_FORMAT=json  # [json|text]
METRICS_ENABLED=true
TRACE_SAMPLING_RATE=0.1  # 10% sampling

# Secrets Management
SECRETS_MANAGER=hashicorp_vault  # [hashicorp_vault|aws_secretsmanager|plaintext]
VAULT_ADDR=https://vault.internal:8200
VAULT_TOKEN_FILE=/run/secrets/vault-token
```

### Database Configuration
```env
# Primary Datastore
DB_TYPE=postgres
DB_HOST=postgres-primary
DB_PORT=5432
DB_NAME=mangahub
DB_USER=${DB_USER}  # Resolved via secrets manager
DB_PASSWORD=${DB_PASSWORD}  # NEVER hardcode
DB_MAX_CONNECTIONS=25  # Per service instance
DB_CONNECTION_TIMEOUT=5s
DB_STATEMENT_TIMEOUT=30s

# Optional Caching Layer
REDIS_URL=redis://redis-cache:6379/0
REDIS_READ_TIMEOUT=100ms
REDIS_WRITE_TIMEOUT=100ms
REDIS_MAX_RETRIES=2
```

### Service-Specific Ports
```env
# API Service (Auth & Core)
HTTP_PORT=8080
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=30s
JWT_SECRET_KEY=${JWT_SECRET}  # 256-bit minimum
SESSION_DURATION=24h

# Search Service
GRPC_PORT=50051
GRPC_MAX_MESSAGE_SIZE=4MB
FTS_REINDEX_INTERVAL=1h

# Chat Service
WS_PORT=8081
WS_PING_INTERVAL=30s
WS_MAX_MESSAGE_SIZE=8192  # 8KB
WS_PRESENCE_UPDATE_INTERVAL=10s

# Notification Service
UDP_PORT=4000
UDP_MAX_PACKET_SIZE=1024  # 1KB
UDP_BROADCAST_TTL=1  # Local network only

# Stats Service
STATS_TCP_PORT=6000
STATS_FLUSH_INTERVAL=5s
STATS_COUNTER_DECAY_RATE=0.95  # For weekly_score calculation
```

## III. Security Configuration Matrix

| Secret Type           | Development          | Staging               | Production               | Rotation Policy     |
|-----------------------|----------------------|-----------------------|--------------------------|---------------------|
| JWT Signing Key       | Hardcoded (test only)| Vault-mounted         | HSM-backed key           | Quarterly           |
| DB Credentials        | Docker secret        | Vault dynamic secret  | IAM database auth        | Daily               |
| HMAC Key (UDP)        | .env file            | Vault                 | KMS-generated            | Weekly              |
| TLS Certificates      | Self-signed          | Let's Encrypt         | ACM/certificate manager  | 90 days             |
| Admin Credentials     | docker-compose init  | SSO federation        | RBAC with breakglass     | Per session         |

> **Critical Rule**: Production secrets MUST NEVER appear in version control. Use init containers to inject secrets at runtime.

## IV. Configuration Loading Sequence
1. **Compile-time defaults** (sane production defaults embedded in binary)
2. **File-based config** (`/etc/mangahub/config.yaml` - mounted via ConfigMap)
3. **Environment variables** (override file config)
4. **Secrets injection** (from Vault/KMS during service startup)
5. **Dynamic reload** (for non-sensitive config via `/config/reload` endpoint)

## V. Production Hardening Checklist
- [ ] All services run as non-root users with read-only filesystems
- [ ] Database connections use TLS with certificate verification
- [ ] UDP service implements rate limiting at kernel level (iptables)
- [ ] Stats TCP service uses mutual TLS for service authentication
- [ ] Secrets injected via tmpfs volumes (never on disk)
- [ ] Configuration changes require peer-reviewed pull requests
- [ ] Audit logs for all configuration changes retained for 365 days

## VI. Configuration Validation Protocol
1. **Pre-flight checks** on service startup:
   ```go
   func validateConfig() error {
     if os.Getenv("APP_ENV") == "production" && !strings.HasPrefix(os.Getenv("DB_DSN"), "postgres://") {
       return errors.New("PRODUCTION: DB_DSN must use TLS")
     }
     if len(os.Getenv("JWT_SECRET_KEY")) < 32 {
       return errors.New("JWT secret must be >=32 bytes")
     }
     // Additional service-specific validations
   }
   ```
2. **Configuration drift detection**: Hash-based config versioning in metrics
3. **Safe defaults**: Services fail fast on invalid configuration (never degrade)
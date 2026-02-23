# User Service Architecture Documentation

## 📊 Codebase Statistics

### Overview
- **Total Go Files**: 28
- **Source Files**: 24 (non-test)
- **Test Files**: 4
- **Total Lines of Code**: ~4,285 lines
- **Architecture**: Clean Architecture (Hexagonal/Ports & Adapters)

### Lines of Code by Layer
```
Layer                    Files    Lines    Purpose
────────────────────────────────────────────────────────────────
cmd/                     1        ~580     Service bootstrapping
internal/config/         1        ~171     Configuration management
internal/domain/         1        ~72      Business entities
internal/application/    1        ~270     Business logic/use cases
internal/infrastructure/ 14       ~1,800   External dependencies
internal/interfaces/     6        ~1,392   HTTP handlers & middleware
────────────────────────────────────────────────────────────────
TOTAL                    24       ~4,285
```

## 🏗️ Architecture Pattern: Clean Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Layer                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │           Middleware Chain (Cross-Cutting)            │  │
│  │  Rate Limit → Auth → Logging → Metrics → Security    │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ↓                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              HTTP Handlers (Controllers)              │  │
│  │    Register | Login | GetUser | UpdateProfile        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                   Application Layer                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                   UserService                         │  │
│  │  Business Logic Orchestration & Use Cases             │  │
│  │  • Register  • Login  • GetUser  • UpdateUser         │  │
│  └───────────────────────────────────────────────────────┘  │
│                            ↓                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         Interfaces (Dependency Inversion)             │  │
│  │  UserRepository | UserCache | PasswordValidator      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                     Domain Layer                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    User Entity                        │  │
│  │  Core business rules, no framework dependencies      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│                 Infrastructure Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  PostgreSQL  │  │    Redis     │  │     Auth     │      │
│  │  Repository  │  │    Cache     │  │  JWT/Bcrypt  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Sanitizer   │  │   Logger     │  │  Prometheus  │      │
│  │  Security    │  │   Zerolog    │  │   Metrics    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## 📂 Directory Structure

```
user-service/
│
├── cmd/
│   └── main.go                              # 🚀 Entry point, bootstrapping
│
├── internal/
│   │
│   ├── domain/                              # 🎯 Core business entities
│   │   └── user.go                          # User entity (ID, Email, etc.)
│   │
│   ├── application/                         # 💼 Business logic layer
│   │   ├── user_service.go                  # Use cases orchestration
│   │   └── user_service_test.go             # Unit tests
│   │
│   ├── interfaces/                          # 🌐 External interfaces
│   │   └── http/
│   │       ├── handlers/
│   │       │   └── user_handler.go          # HTTP request handlers
│   │       │
│   │       └── middleware/
│   │           ├── auth.go                  # JWT authentication
│   │           ├── logging.go               # Request logging
│   │           ├── ratelimit.go             # Rate limiting (in-memory)
│   │           ├── redis_ratelimit.go       # Rate limiting (Redis)
│   │           ├── metrics.go               # Prometheus metrics
│   │           ├── security.go              # Security headers
│   │           └── cors.go                  # CORS support
│   │
│   ├── infrastructure/                      # 🔧 External systems
│   │   ├── postgres/
│   │   │   ├── connection.go                # DB connection & pooling
│   │   │   ├── user_repository.go           # Data access implementation
│   │   │   ├── user_model.go                # DB model mapping
│   │   │   ├── transaction.go               # Transaction manager
│   │   │   ├── migrations.go                # Migration runner
│   │   │   └── errors.go                    # DB error handling
│   │   │
│   │   ├── redis/
│   │   │   ├── client.go                    # Redis operations
│   │   │   └── user_cache.go                # Cache implementation
│   │   │
│   │   ├── auth/
│   │   │   ├── jwt.go                       # JWT token management
│   │   │   └── password.go                  # Bcrypt & validation
│   │   │
│   │   ├── security/
│   │   │   └── sanitizer.go                 # Input sanitization
│   │   │
│   │   └── logger/
│   │       └── logger.go                    # Logging setup
│   │
│   └── config/
│       └── config.go                        # Configuration loading
│
├── migrations/                              # 📦 Database migrations
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
│
├── .env                                     # ⚙️ Environment variables
├── Dockerfile                               # 🐳 Container image
├── Makefile                                 # 🛠️ Build automation
├── go.mod                                   # 📦 Dependencies
└── test.sh                                  # 🧪 Test script
```

## 🔄 Request Flow Diagrams

### 1. User Registration Flow

```
┌─────────┐
│ Client  │
└────┬────┘
     │ POST /api/v1/auth/register
     │ {username, email, password}
     ↓
┌────────────────────────────────────────────┐
│         Rate Limit Middleware              │
│  ✓ Check: 5 requests/min per IP           │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│        Logging Middleware                  │
│  • Generate Request ID (UUID)              │
│  • Log request metadata                    │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│        Security Middleware                 │
│  • Add security headers                    │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│      Register Handler                      │
│  1. Decode JSON                            │
│  2. Validate input                         │
│     ├─ Required fields?                    │
│     ├─ Email format valid?                 │
│     ├─ Min/max lengths?                    │
│     └─ Password strength?                  │
│  3. Sanitize input                         │
│     ├─ Username (alphanumeric only)        │
│     └─ Email (lowercase, trim)             │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│         UserService.Register()             │
│                                            │
│  1. Validate password strength             │
│     └─ Min 8 chars, uppercase, lowercase,  │
│        numbers, special chars              │
│                                            │
│  2. Check email uniqueness                 │
│     └─ Repository.GetByEmail()             │
│                                            │
│  3. Hash password                          │
│     └─ bcrypt.GenerateFromPassword()       │
│                                            │
│  4. Create user in transaction             │
│     └─ TransactionManager.WithTransaction()│
│        └─ Repository.Create()              │
│                                            │
│  5. Invalidate cache (if exists)           │
│     └─ Cache.Delete()                      │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       PostgreSQL Database                  │
│  INSERT INTO users (...)                   │
│  • Unique constraint on email              │
│  • Auto-generate UUID for ID               │
│  • Set created_at timestamp                │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       Prometheus Metrics                   │
│  user_registrations_total++                │
│  http_requests_total{status=201}++         │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│           Response                         │
│  HTTP 201 Created                          │
│  {                                         │
│    "id": "uuid",                           │
│    "username": "...",                      │
│    "email": "...",                         │
│    "created_at": "..."                     │
│  }                                         │
└────────────────────────────────────────────┘
```

### 2. User Login Flow

```
┌─────────┐
│ Client  │
└────┬────┘
     │ POST /api/v1/auth/login
     │ {email, password}
     ↓
┌────────────────────────────────────────────┐
│      Rate Limit Middleware                 │
│  ✓ Check: 10 requests/min per IP          │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│      Login Handler                         │
│  1. Validate input                         │
│  2. Sanitize email                         │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│         UserService.Login()                │
│                                            │
│  1. Find user by email                     │
│     └─ Repository.GetByEmail()             │
│        └─ SELECT * FROM users              │
│           WHERE email = $1                 │
│           AND deleted_at IS NULL           │
│                                            │
│  2. Verify password                        │
│     └─ bcrypt.CompareHashAndPassword()     │
│                                            │
│  3. Update last login timestamp            │
│     └─ Repository.Update()                 │
│                                            │
│  4. Return user entity                     │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│      JWT Token Generation                  │
│  JWTManager.GenerateToken(userID)          │
│  • Claims: {userID, exp, iat, iss}         │
│  • Algorithm: HS256                        │
│  • Expiration: 24h (configurable)          │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       Prometheus Metrics                   │
│  user_logins_total++                       │
│  http_requests_total{status=200}++         │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│           Response                         │
│  HTTP 200 OK                               │
│  {                                         │
│    "user": {...},                          │
│    "token": "eyJhbGc..."                   │
│  }                                         │
└────────────────────────────────────────────┘

     ↓ (on error)
┌────────────────────────────────────────────┐
│       Error Handling                       │
│  • Invalid credentials                     │
│  • User not found                          │
│  • Metrics: user_login_failures_total++    │
│  HTTP 401 Unauthorized                     │
└────────────────────────────────────────────┘
```

### 3. Get Current User Flow (Protected Route)

```
┌─────────┐
│ Client  │
└────┬────┘
     │ GET /api/v1/users/me
     │ Headers: {Authorization: "Bearer <token>"}
     ↓
┌────────────────────────────────────────────┐
│       Auth Middleware                      │
│  1. Extract Bearer token from header       │
│  2. Validate JWT token                     │
│     └─ JWTManager.ValidateToken()          │
│        ├─ Verify signature (HS256)         │
│        ├─ Check expiration                 │
│        └─ Validate issuer                  │
│  3. Extract user ID from claims            │
│  4. Inject into request context            │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│      Get Current User Handler              │
│  1. Get user ID from context               │
│     └─ GetUserID(r)                        │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│         UserService.GetUser()              │
│                                            │
│  ┌──────────────────────────────────────┐ │
│  │   Cache-Aside Pattern                │ │
│  └──────────────────────────────────────┘ │
│                                            │
│  1. Try cache first                        │
│     └─ Cache.Get(userID)                   │
│                                            │
│     ├─ Cache HIT ✓                         │
│     │  └─ Return cached user               │
│     │                                      │
│     └─ Cache MISS ✗                        │
│        ├─ Repository.GetByID()             │
│        │  └─ SELECT * FROM users           │
│        │     WHERE id = $1                 │
│        │                                   │
│        └─ Cache.Set(user, TTL=5m)          │
│           └─ Return user                   │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       Response                             │
│  HTTP 200 OK                               │
│  {                                         │
│    "id": "...",                            │
│    "username": "...",                      │
│    "email": "...",                         │
│    // password excluded                   │
│  }                                         │
└────────────────────────────────────────────┘
```

### 4. Update Profile Flow (Protected + Rate Limited)

```
┌─────────┐
│ Client  │
└────┬────┘
     │ PUT /api/v1/users/profile
     │ Headers: {Authorization: "Bearer <token>"}
     │ Body: {first_name, last_name, phone, birthday, gender}
     ↓
┌────────────────────────────────────────────┐
│       Auth Middleware                      │
│  • Validate JWT                            │
│  • Extract user ID → context               │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│   User-Based Rate Limit Middleware         │
│  • Key: "user:{userID}"                    │
│  • Limit: 10 requests/min                  │
│  • Uses Redis if available                 │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│      Update Profile Handler                │
│  1. Validate input                         │
│     ├─ Phone: E.164 format                 │
│     ├─ Gender: enum (male/female/other)    │
│     └─ Birthday: date format               │
│  2. Sanitize each field                    │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       UserService.UpdateUser()             │
│                                            │
│  1. Fetch current user                     │
│     └─ Repository.GetByID()                │
│                                            │
│  2. If email changed                       │
│     └─ Check uniqueness                    │
│        └─ Repository.GetByEmail()          │
│                                            │
│  3. Update in transaction                  │
│     └─ TransactionManager.WithTransaction()│
│        └─ Repository.Update()              │
│                                            │
│  4. Invalidate cache                       │
│     ├─ Cache.Delete(userID)                │
│     ├─ Cache.Delete(oldEmail)              │
│     └─ Cache.Delete(newEmail)              │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       PostgreSQL                           │
│  UPDATE users SET                          │
│    first_name = $1,                        │
│    last_name = $2,                         │
│    phone = $3,                             │
│    updated_at = NOW()                      │
│  WHERE id = $4                             │
└────────────────┬───────────────────────────┘
                 ↓
┌────────────────────────────────────────────┐
│       Response                             │
│  HTTP 200 OK                               │
│  {updated_user}                            │
└────────────────────────────────────────────┘
```

## 🔐 Security Features

### Authentication & Authorization
- **JWT-based authentication** with HS256 signing
- **Bearer token** validation in middleware
- **Bcrypt password hashing** with default cost
- **Password strength validation** (8+ chars, complexity rules)

### Input Validation & Sanitization
```go
// Multi-layer defense
1. Struct validation tags     // Required, min/max, email format
2. Custom validators           // Password strength, business rules
3. Input sanitizer            // XSS, SQL injection, path traversal
4. Type-safe parsing          // Prevent type confusion
```

### Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

### Rate Limiting
- **IP-based rate limiting** with trusted proxy support
- **User-based rate limiting** for authenticated endpoints
- **Per-route limits** (login: 10/min, register: 5/min)
- **Distributed rate limiting** with Redis (optional)

### Data Protection
- **Soft deletes** (data preservation)
- **Password exclusion** in API responses
- **Email uniqueness** constraint
- **HTTPS enforcement** (in production)

## 📊 Observability

### Logging (Zerolog)
```go
// Structured logging with context
{
  "level": "info",
  "request_id": "uuid-v4",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "duration_ms": 45.2,
  "ip": "127.0.0.1",
  "user_agent": "..."
}
```

### Metrics (Prometheus)
```
# HTTP metrics
http_requests_total{method, endpoint, status}
http_request_duration_seconds{method, endpoint}
http_requests_in_flight

# Business metrics
user_registrations_total
user_logins_total
user_login_failures_total
user_password_changes_total
```

### Health Check
```
GET /health
{
  "status": "healthy",
  "timestamp": "2024-12-27T...",
  "services": {
    "database": "connected",
    "redis": "connected"  // or "disconnected"
  }
}
```

## 🗄️ Database Design

### Users Table Schema
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    gender VARCHAR(10),
    birthday DATE,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP  -- Soft delete
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### Soft Delete Pattern
```go
// Query automatically excludes soft-deleted records
db.Where("deleted_at IS NULL").Find(&users)

// Soft delete
db.Model(&user).Update("deleted_at", time.Now())

// Hard delete (admin only)
db.Unscoped().Delete(&user)
```

## 🚀 Performance Optimizations

### Caching Strategy
```
1. Cache Key Format:
   - "user:id:{uuid}"
   - "user:email:{email}"

2. TTL: 5 minutes (configurable)

3. Invalidation Strategy:
   - On update: Delete by ID and email
   - On delete: Delete by ID and email
   - On email change: Delete old and new email keys
```

### Connection Pooling
```go
MaxIdleConns:    10   // Idle connections in pool
MaxOpenConns:    100  // Maximum open connections
ConnMaxLifetime: 5m   // Connection reuse time
ConnMaxIdleTime: 1m   // Idle connection timeout
```

### Database Query Optimization
- **Indexed email lookups** for authentication
- **Prepared statements** via GORM
- **Pagination support** (offset + limit)
- **Select specific columns** where possible
- **Batch operations** for cache invalidation

## 🔧 Configuration

### Environment Variables
```bash
# Server
PORT=8081
JWT_SECRET=your-secret-key
JWT_EXPIRE=24h

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=user_service_db
DB_MAX_IDLE_CONNS=10
DB_MAX_OPEN_CONNS=100

# Redis (optional)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Cache
CACHE_USER_TTL=5m

# Rate Limiting
RATE_LIMIT_GLOBAL=100.0        # requests/sec
RATE_LIMIT_LOGIN=0.167         # 10/min
RATE_LIMIT_REGISTER=0.083      # 5/min

# Proxy Trust (Security)
TRUST_PROXIES=false
TRUSTED_PROXY_CIDRS=127.0.0.1/32
```

## 🧪 Testing

### Test Coverage
- **Application Layer**: 71.4%
- **Infrastructure/Auth**: Password validation tests
- **Infrastructure/Security**: Sanitizer tests
- **Middleware**: Rate limiting tests

### Test Patterns
```go
// Table-driven tests
tests := []struct {
    name     string
    input    Input
    expected Output
    wantErr  bool
}{...}

// Mocking with interfaces
mockRepo := &MockUserRepository{}
service := NewUserService(mockRepo, ...)
```

## 📦 Deployment

### Docker Image
```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder
# Build static binary
FROM gcr.io/distroless/static-debian12:nonroot
# Minimal attack surface, non-root user
```

### Health Check
```bash
# Container health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ["/user-service", "-healthcheck"]
```

### Graceful Shutdown
```go
// Wait for connections to drain (15s timeout)
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## 🎯 API Endpoints

```
Public Endpoints:
  POST   /api/v1/auth/register     Rate: 5/min
  POST   /api/v1/auth/login        Rate: 10/min

Protected Endpoints:
  GET    /api/v1/users/me
  PUT    /api/v1/users/profile     Rate: 10/min (per user)
  POST   /api/v1/users/change-password
  DELETE /api/v1/users/delete
  GET    /api/v1/users             (Admin/List)

System Endpoints:
  GET    /health
  GET    /metrics                  (Prometheus)
```

## 🔑 Key Design Decisions

1. **Clean Architecture** - Separation of concerns, testability
2. **Interface-based dependencies** - Loose coupling, mockable
3. **Cache-aside pattern** - Performance without complexity
4. **Soft deletes** - Data preservation, audit trail
5. **JWT stateless auth** - Scalability, no session storage
6. **Graceful degradation** - Service works without Redis
7. **Structured logging** - Observability, debugging
8. **Prometheus metrics** - Production monitoring
9. **Rate limiting** - DDoS protection, abuse prevention
10. **Multi-layer validation** - Security in depth

## 📚 Further Reading

- [Internal Documentation](./SECURITY_FIX_X_FORWARDED_FOR.md) - Security fixes
- [Makefile](./Makefile) - Build and run commands
- [Migrations](./migrations/) - Database schema evolution
- [Tests](./test.sh) - Test execution guide

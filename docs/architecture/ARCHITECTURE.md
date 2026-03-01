# DatRi Architecture

> A visual and written breakdown of how DatRi is structured internally and how it serves clients.

---

## Architecture Diagram

![DatRi Architecture](./architecture.png)

---

## Overview

DatRi is a **schema-driven API gateway**. It connects to a database, introspects its schema at startup, and automatically generates API endpoints — no code required from the user.

The architecture follows a clean **layered design** where every layer talks to **interfaces**, never to concrete implementations. This is what allows DatRi to support Postgres, MySQL, MinIO, and S3 without changing anything in the server layer.

---

## Layer-by-Layer Breakdown

### 1. Client Layer
External consumers of the DatRi API:
- `curl`, Postman, frontend apps, mobile apps
- Communicate over **REST**, **GraphQL**, **gRPC**, or **WebSocket**

---

### 2. `cmd/datri/` — Entry Point
The CLI binary that boots the application.

**Responsibilities:**
- Parses CLI flags: `--db`, `--port`, `--config`
- Loads `datri.yaml` configuration
- Initialises all internal components
- Wires everything together and starts the server(s)

```bash
datri serve --db "postgres://user:pass@localhost/mydb" --port 8080
```

---

### 3. `internal/config/` — Configuration
Reads and validates `datri.yaml`.

```yaml
server:
  port: 8080
  protocols:
    - rest

database:
  type: postgres
  dsn: "postgres://localhost/mydb"

auth:
  enabled: true
  type: jwt
  secret: your-secret-key

cors:
  enabled: true
  origins: ["*"]
```

---

### 4. `internal/logger/` — Observability
Enterprise-grade structured logging using **zerolog**.
- JSON and console output modes
- Multiple log levels: `debug`, `info`, `warn`, `error`, `fatal`
- HTTP middleware integration
- Used by **every layer** in the application

---

### 5. `internal/server/` — Protocol Layer
Each protocol has its own sub-package but all share the **same handler logic**.

| Sub-package | Library | Status |
|---|---|---|
| `rest/` | `chi` router | 🔨 Build Next |
| `graphql/` | `gqlgen` | 🔜 v2.0 |
| `grpc/` | `google.golang.org/grpc` | 🔜 v4.0 |
| `websocket/` | `gorilla/websocket` | 🔜 v3.0 |

**Shared Handler Logic** (protocol-agnostic):
1. Validates incoming request
2. Builds a parameterized SQL query using `SelectBuilder`
3. Calls the `database.DB` interface
4. Serializes DB rows into the response format (JSON / protobuf)

---

### 6. `internal/database/` — Data Layer ✅ Built

The core data abstraction. All server layers talk to the `DB` interface — never to `postgres` or `mysql` packages directly.

| File | Purpose |
|---|---|
| `interface.go` | `DB`, `Rows`, `Row` interfaces |
| `schema.go` | `Schema`, `TableInfo`, `ColumnInfo`, `ForeignKey` types |
| `query.go` | `SelectBuilder` — SQL injection-safe, dialect-aware query builder |
| `row.go` | Row scanning helpers |
| `config.go` | Connection pool configuration |
| `postgres/` | Postgres driver implementation |
| `mysql/` | MySQL driver implementation |

---

### 7. `internal/filestore/` — File Storage Layer ✅ Built

Abstracts object storage backends behind the `Store` interface.

| Component | Status |
|---|---|
| `store.go` — `Store` interface | ✅ Built |
| `minio/` — MinIO backend | ✅ Built |
| S3 backend | 🔜 Future |
| Azure Blob backend | 🔜 Future |

---

### 8. `internal/auth/` — Security Layer
Pluggable authentication middleware.

| Feature | Status |
|---|---|
| JWT authentication | 🔜 Coming Soon |
| API Key auth | 🔜 Coming Soon |
| OAuth 2.0 | 🔜 Future |

---

## The Core Design Principle

```
Every layer above talks to INTERFACES, not implementations.

internal/server/rest  →  database.DB       (not postgres.DB directly)
internal/server/rest  →  filestore.Store   (not minio.Store directly)
```

This means:
- Swapping Postgres → MySQL requires **zero changes** in the server layer
- Swapping MinIO → S3 requires **zero changes** in the server layer
- Adding a new protocol (gRPC) requires **zero changes** in the database layer

---

## Request Flow (REST Example)

```
Client
  │
  │  GET /users?limit=10
  ▼
chi Router  (internal/server/rest)
  │
  │  Route matched: GET /{table}
  ▼
Handler
  │  1. Extract table name: "users"
  │  2. Build query via SelectBuilder:
  │     SELECT * FROM "users" LIMIT $1  → args: [10]
  │  3. Call db.Query(ctx, sql, args...)
  ▼
database.DB  (Postgres / MySQL driver)
  │
  │  Execute SQL → return Rows
  ▼
Handler
  │  4. Scan rows into []map[string]any
  │  5. Write JSON response
  ▼
Client  ←  {"data": [{"id":1,"name":"Alice"}, ...]}
```

---

## Build Roadmap

```
Priority 1  →  cmd/datri/main.go         Boot the app
Priority 2  →  internal/config/          Parse datri.yaml
Priority 3  →  internal/server/rest/     REST API (chi router)
Priority 4  →  internal/auth/            JWT middleware
Priority 5  →  internal/server/graphql/  GraphQL support
Priority 6  →  internal/server/websocket/ WebSocket support
Priority 7  →  internal/server/grpc/     gRPC support
```

---

*Last updated: 2026-03-01*

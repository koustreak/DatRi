# DatRi Project Structure

This document explains the purpose of each directory in the DatRi project.

## 📁 Directory Layout

```
DatRi/
├── cmd/datri/                    # Main CLI application entry point
├── internal/                     # Private application code (not importable)
│   ├── server/                   # Server implementations
│   │   ├── rest/                 # REST API server
│   │   ├── graphql/              # GraphQL server
│   │   ├── grpc/                 # gRPC server
│   │   └── websocket/            # WebSocket server
│   ├── database/                 # Database adapters
│   │   ├── sqlite/               # SQLite adapter
│   │   ├── postgres/             # PostgreSQL adapter
│   │   └── mysql/                # MySQL adapter
│   ├── datasource/               # File-based data sources
│   │   ├── json/                 # JSON file handler
│   │   ├── csv/                  # CSV file handler
│   │   └── yaml/                 # YAML file handler
│   ├── auth/                     # Authentication & authorization
│   │   ├── jwt/                  # JWT implementation
│   │   └── apikey/               # API key implementation
│   ├── config/                   # Configuration management
│   ├── schema/                   # Schema introspection & generation
│   └── logger/                   # Structured logging
├── pkg/                          # Public libraries (importable by other projects)
│   ├── protocol/                 # Protocol utilities
│   │   ├── rest/                 # REST utilities
│   │   ├── graphql/              # GraphQL utilities
│   │   └── grpc/                 # gRPC utilities
│   └── errors/                   # Custom error types
├── api/                          # API contracts & definitions
│   ├── proto/v1/                 # Protocol buffer definitions (gRPC)
│   └── openapi/v1/               # OpenAPI/Swagger specifications (REST)
├── docs/                         # Documentation
│   ├── architecture/             # Architecture Decision Records (ADRs)
│   ├── guides/                   # User guides
│   └── api/                      # API documentation
├── scripts/                      # Build & development scripts
├── test/                         # Integration & E2E tests
│   ├── integration/              # Integration tests
│   ├── e2e/                      # End-to-end tests
│   └── fixtures/                 # Test data
├── deployments/                  # Deployment configurations
│   ├── docker/                   # Docker files
│   └── kubernetes/               # Kubernetes manifests
├── .github/                      # GitHub specific files
│   ├── workflows/                # CI/CD pipelines
│   └── ISSUE_TEMPLATE/           # Issue templates
└── examples/                     # Example configurations
    ├── basic/                    # Basic usage examples
    └── advanced/                 # Advanced usage examples
```

## 🎯 Design Principles

### **DRY (Don't Repeat Yourself)**
- Shared logic lives in `pkg/` for reusability
- Common interfaces defined once in `internal/database/adapter.go`
- Protocol-specific utilities abstracted in `pkg/protocol/`

### **Clean Architecture**
- **Dependency Rule**: Dependencies point inward
  - `cmd/` depends on `internal/`
  - `internal/` depends on `pkg/`
  - `pkg/` has no internal dependencies
- **Interface-Driven**: All adapters implement interfaces
- **Testability**: Easy to mock and test in isolation

### **Single Responsibility**
- Each package has one clear purpose
- Server implementations separated by protocol
- Database adapters isolated from business logic

## 📦 Package Responsibilities

### `cmd/datri/`
**Purpose**: Application entry point  
**Responsibilities**:
- Parse CLI flags
- Load configuration
- Initialize dependencies
- Start servers

**Rule**: Keep minimal - no business logic

### `internal/server/`
**Purpose**: Protocol server implementations  
**Responsibilities**:
- Handle HTTP/gRPC/WebSocket connections
- Route requests to handlers
- Middleware integration

**Key Files**:
- `rest/server.go` - REST server setup
- `graphql/server.go` - GraphQL server setup
- `grpc/server.go` - gRPC server setup
- `websocket/server.go` - WebSocket server setup

### `internal/database/`
**Purpose**: Database abstraction layer  
**Responsibilities**:
- Connect to databases
- Execute queries
- Schema introspection
- Connection pooling

**Key Files**:
- `adapter.go` - Database interface definition
- `sqlite/adapter.go` - SQLite implementation
- `postgres/adapter.go` - PostgreSQL implementation

### `internal/datasource/`
**Purpose**: File-based data source handlers  
**Responsibilities**:
- Read/write JSON/CSV/YAML files
- Parse and validate file formats
- Watch for file changes

### `internal/auth/`
**Purpose**: Authentication & authorization  
**Responsibilities**:
- JWT token generation/validation
- API key management
- Auth middleware

### `internal/config/`
**Purpose**: Configuration management  
**Responsibilities**:
- Load config from files/env/flags
- Validate configuration
- Provide config to other packages

### `internal/schema/`
**Purpose**: Schema introspection & generation  
**Responsibilities**:
- Read database schemas
- Generate GraphQL schemas
- Generate gRPC proto definitions
- Generate OpenAPI specs

### `pkg/protocol/`
**Purpose**: Reusable protocol utilities  
**Responsibilities**:
- Common REST helpers
- GraphQL utilities
- gRPC helpers

**Why Public**: Other projects might want to use these utilities

### `pkg/errors/`
**Purpose**: Custom error types  
**Responsibilities**:
- Define domain errors
- Error wrapping/unwrapping
- Error formatting

### `api/`
**Purpose**: API contract definitions  
**Responsibilities**:
- Protocol buffer definitions (`.proto`)
- OpenAPI specifications (`.yaml`)
- Generated code (from protoc, swagger-gen)

### `test/`
**Purpose**: Integration & E2E tests  
**Note**: Unit tests live alongside code (`*_test.go`)

### `deployments/`
**Purpose**: Deployment artifacts  
**Responsibilities**:
- Dockerfiles
- Kubernetes manifests
- Helm charts (future)

### `scripts/`
**Purpose**: Automation scripts  
**Responsibilities**:
- Build scripts
- Test runners
- Linting
- Release automation

### `examples/`
**Purpose**: Usage examples  
**Responsibilities**:
- Sample configurations
- Tutorial code
- Demo databases

## 🔄 Data Flow

```
CLI Input → cmd/datri/main.go
    ↓
Config Loading → internal/config/
    ↓
Database Connection → internal/database/
    ↓
Schema Introspection → internal/schema/
    ↓
Server Initialization → internal/server/
    ↓
Request Handling → Protocol Handlers
    ↓
Response → Client
```

## 🧪 Testing Strategy

- **Unit Tests**: Next to code (`*_test.go`)
- **Integration Tests**: `test/integration/` (test multiple packages together)
- **E2E Tests**: `test/e2e/` (test entire system)
- **Fixtures**: `test/fixtures/` (sample databases, files)

## 📚 Documentation Strategy

- **Code Documentation**: GoDoc comments in code
- **Architecture Decisions**: `docs/architecture/` (ADRs)
- **User Guides**: `docs/guides/`
- **API Docs**: `docs/api/` + auto-generated from OpenAPI/proto

## 🚀 Build & Release

- **Makefile**: Primary build interface
- **Scripts**: Complex build logic in `scripts/`
- **CI/CD**: `.github/workflows/`
- **Docker**: Multi-stage builds in `deployments/docker/`

---

**Last Updated**: 2026-02-08  
**Maintained By**: Koushik (@koustreak)

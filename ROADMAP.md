# DatRi Development Roadmap

## 🎯 Current Version: v1.0 (REST Focus)

### ✅ Implemented
- **REST API Support**
  - HTTP request/response handling
  - Structured JSON logging
  - HTTP middleware for automatic request logging
  - Comprehensive test coverage

- **Logger Package**
  - Enterprise-grade logging with zerolog
  - Multiple log levels (debug, info, warn, error, fatal)
  - JSON and console output formats
  - Context integration
  - HTTP middleware

---

## 🚀 Future Versions

### v2.0 - GraphQL Support
**Planned Features:**
- GraphQL schema definition
- Query, mutation, and subscription support
- GraphQL-specific logging middleware
  - Operation-level logging
  - Field resolver tracking
  - Subscription lifecycle logging
- GraphQL error handling with field paths

**Dependencies to Add:**
- GraphQL library (e.g., `github.com/graphql-go/graphql` or `github.com/99designs/gqlgen`)

---

### v3.0 - WebSocket Support
**Planned Features:**
- WebSocket connection handling
- Bidirectional message streaming
- Connection lifecycle management
- WebSocket-specific logging
  - Upgrade logging
  - Message-level logging
  - Connection statistics (messages sent/received, bytes transferred)

**Dependencies to Add:**
- `github.com/gorilla/websocket`

---

### v4.0 - gRPC Support
**Planned Features:**
- gRPC server implementation
- Unary and streaming RPC support
- Protocol Buffers integration
- gRPC interceptors for logging
  - Unary interceptor
  - Stream interceptor
  - gRPC status code mapping

**Dependencies to Add:**
- `google.golang.org/grpc`
- `google.golang.org/protobuf`

---

### v5.0 - Raw TCP Support
**Planned Features:**
- Raw TCP connection handling
- Custom protocol support
- TCP connection pooling
- TCP-specific logging
  - Connection lifecycle
  - Read/write operation logging
  - Byte transfer statistics

**Dependencies to Add:**
- Standard library `net` package (already available)

---

## 📋 Development Principles

1. **Incremental Releases**
   - Focus on one protocol at a time
   - Ensure stability before adding new features
   - Maintain backward compatibility

2. **Consistent Logging**
   - All protocols use the same logger package
   - Structured JSON output for all protocols
   - Consistent field naming across protocols

3. **Production-Ready**
   - Comprehensive test coverage for each protocol
   - Performance benchmarks
   - Documentation and examples

4. **Developer Experience**
   - Clear API design
   - Extensive documentation
   - Usage examples for each protocol

---

## 🎨 Design Decisions

### Why REST First?
- **Most Common Use Case**: REST APIs are the most widely used
- **Simpler Implementation**: HTTP is well-understood and has excellent Go support
- **Faster Time to Market**: Get a working product to users quickly
- **Foundation for Others**: Many protocols (GraphQL, WebSocket) build on HTTP

### Protocol Priority Order
1. **REST** - Foundation, most common
2. **GraphQL** - Modern API standard, builds on HTTP
3. **WebSocket** - Real-time communication, builds on HTTP
4. **gRPC** - High-performance RPC, different paradigm
5. **TCP** - Maximum flexibility, lowest level

---

## 📝 Notes

- Protocol-specific middleware implementations have been prototyped and can be found in git history
- Each version will include:
  - Core protocol support
  - Logging middleware
  - Comprehensive tests
  - Documentation and examples
  - Performance benchmarks

---

## 🤝 Contributing

When adding support for new protocols:
1. Follow existing logger patterns
2. Implement protocol-specific middleware
3. Add comprehensive tests
4. Update documentation
5. Provide usage examples

---

**Last Updated**: 2026-02-10  
**Current Focus**: REST API (v1.0)

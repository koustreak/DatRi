# DatRi 🚀

**The Swiss Army Knife for Local API Servers**

Transform any SQL database or file into a fully-featured API server with REST, GraphQL, gRPC, and WebSocket support — all from a single binary.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

## 🎯 Why DatRi?

**Stop writing boilerplate API servers.** Whether you're prototyping a frontend, testing microservices, or building a quick backend, DatRi gets you from zero to a production-ready API in seconds.

```bash
# That's it. Your API is live.
datri serve --db mydata.db
```

Now access your data via:
- 🌐 **REST** → `http://localhost:8080/users`
- 📊 **GraphQL** → `http://localhost:8080/graphql`
- ⚡ **gRPC** → `localhost:50051`
- 🔌 **WebSocket** → `ws://localhost:8080/ws`

---

## ✨ Features

- **🔥 Multi-Protocol Support** — REST, GraphQL, gRPC, WebSocket from one source
- **💾 SQL-First** — Works with SQLite, PostgreSQL, MySQL, and more
- **📁 File Support** — Serve JSON, CSV, or YAML files as APIs
- **🔐 Auth Ready** — Built-in support for JWT, API keys, OAuth (coming soon)
- **⚡ Zero Config** — Sensible defaults, works out of the box
- **🐳 Docker Ready** — Single binary, easy to containerize
- **🔄 Hot Reload** — Auto-detects schema changes
- **📖 Auto Documentation** — OpenAPI/Swagger + GraphQL Playground included

---

## 🚀 Quick Start

### Installation

```bash
# Using Go
go install github.com/koustreak/DatRi/cmd/datri@latest

# Or download binary from releases
curl -L https://github.com/koustreak/DatRi/releases/latest/download/datri-linux-amd64 -o datri
chmod +x datri
```

### Basic Usage

**1. Serve a SQLite database:**
```bash
datri serve --db ./myapp.db
```

**2. Serve a JSON file:**
```bash
datri serve --file ./data.json
```

**3. Connect to PostgreSQL:**
```bash
datri serve --db "postgres://user:pass@localhost/mydb"
```

---

## 📚 Examples

### REST API
```bash
# GET all users
curl http://localhost:8080/users

# GET user by ID
curl http://localhost:8080/users/1

# POST new user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

### GraphQL
```graphql
query {
  users(limit: 10) {
    id
    name
    email
  }
}
```

### gRPC
```bash
grpcurl -plaintext localhost:50051 datri.UserService/GetUser
```

### WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/users');
ws.onmessage = (event) => console.log(JSON.parse(event.data));
```

---

## 🛠️ Configuration

Create a `datri.yaml` for advanced setups:

```yaml
server:
  port: 8080
  protocols:
    - rest
    - graphql
    - grpc
    - websocket

database:
  type: postgres
  connection: "postgres://localhost/mydb"
  
auth:
  enabled: true
  type: jwt
  secret: your-secret-key

cors:
  enabled: true
  origins: ["*"]
```

Then run:
```bash
datri serve --config datri.yaml
```

---

## 🎨 Use Cases

| Scenario | How DatRi Helps |
|----------|-----------------|
| **Frontend Development** | Mock backend APIs without writing server code |
| **Microservices Testing** | Quickly spin up test services with realistic data |
| **Prototyping** | Validate ideas with a real API in minutes |
| **Database Exploration** | Instantly expose any SQL database as an API |
| **Mobile App Development** | Test different API patterns (REST vs GraphQL) easily |

---

## 🗺️ Roadmap

- [x] REST API support
- [x] SQL database connectors
- [ ] GraphQL support
- [ ] gRPC support
- [ ] WebSocket support
- [ ] JWT authentication
- [ ] Rate limiting
- [ ] API key management
- [ ] Custom middleware plugins
- [ ] Real-time subscriptions
- [ ] Admin dashboard

---

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## 📄 License

MIT © [Koushik](https://github.com/koustreak)

---

## 🌟 Show Your Support

If DatRi saves you time, give it a ⭐ on GitHub!

**Built with ❤️ using Go**
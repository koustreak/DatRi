# Contributing to DatRi 🚀

First off, **thank you** for considering contributing to DatRi! It's people like you that make DatRi a great tool for the developer community.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Project Structure](#project-structure)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)

---

## 📜 Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow:

- **Be respectful** and inclusive
- **Be collaborative** and constructive
- **Focus on what is best** for the community
- **Show empathy** towards other community members

---

## 🤝 How Can I Contribute?

### Reporting Bugs 🐛

Before creating bug reports, please check existing issues to avoid duplicates.

**When submitting a bug report, include:**
- **Clear title** and description
- **Steps to reproduce** the issue
- **Expected behavior** vs **actual behavior**
- **Environment details** (OS, Go version, DatRi version)
- **Logs or error messages** (if applicable)
- **Sample configuration** (if relevant)

### Suggesting Enhancements 💡

Enhancement suggestions are tracked as GitHub issues.

**When suggesting an enhancement:**
- **Use a clear title** describing the enhancement
- **Provide detailed description** of the proposed functionality
- **Explain why** this enhancement would be useful
- **Provide examples** of how it would work
- **Consider alternatives** you've thought about

### Your First Code Contribution 🎉

Unsure where to begin? Look for issues labeled:
- `good first issue` - Simple issues for newcomers
- `help wanted` - Issues where we need community help
- `documentation` - Documentation improvements

---

## 🛠️ Development Setup

### Prerequisites

- **Go 1.22+** ([Download](https://go.dev/dl/))
- **Git**
- **Make** (optional but recommended)
- **Docker** (for testing Docker builds)

### Setup Steps

1. **Fork the repository** on GitHub

2. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/DatRi.git
   cd DatRi
   ```

3. **Add upstream remote:**
   ```bash
   git remote add upstream https://github.com/koustreak/DatRi.git
   ```

4. **Install dependencies:**
   ```bash
   go mod download
   ```

5. **Install development tools:**
   ```bash
   make install-tools
   ```
   This installs:
   - `golangci-lint` - Linting
   - `gofumpt` - Formatting
   - `mockgen` - Mock generation
   - `protoc-gen-go` - Protocol buffer compiler

6. **Verify setup:**
   ```bash
   make test
   ```

---

## 💻 Coding Standards

We follow **Google's Go Style Guide** and **Effective Go** principles.

### Core Principles

#### 1. **DRY (Don't Repeat Yourself)**
- Extract common logic into reusable functions
- Use interfaces for abstraction
- Avoid code duplication

**Bad:**
```go
// Duplicated error handling
func GetUser(id int) (*User, error) {
    user, err := db.Query(...)
    if err != nil {
        log.Printf("error: %v", err)
        return nil, err
    }
    return user, nil
}

func GetPost(id int) (*Post, error) {
    post, err := db.Query(...)
    if err != nil {
        log.Printf("error: %v", err)
        return nil, err
    }
    return post, nil
}
```

**Good:**
```go
// Reusable error handling
func queryWithLogging[T any](query string, args ...any) (*T, error) {
    var result T
    err := db.QueryRow(query, args...).Scan(&result)
    if err != nil {
        log.Printf("query error: %v", err)
        return nil, err
    }
    return &result, nil
}
```

#### 2. **SOLID Principles**

**Single Responsibility:**
```go
// Bad: Handler does too much
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Validate data
    // Hash password
    // Save to database
    // Send email
    // Return response
}

// Good: Separated concerns
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    req, err := h.parser.ParseCreateUserRequest(r)
    if err != nil {
        h.respondError(w, err)
        return
    }
    
    user, err := h.service.CreateUser(req)
    if err != nil {
        h.respondError(w, err)
        return
    }
    
    h.respondJSON(w, http.StatusCreated, user)
}
```

**Interface Segregation:**
```go
// Bad: Fat interface
type Database interface {
    Query(string) ([]Row, error)
    Insert(string) error
    Update(string) error
    Delete(string) error
    Backup() error
    Restore() error
}

// Good: Focused interfaces
type Querier interface {
    Query(string) ([]Row, error)
}

type Writer interface {
    Insert(string) error
    Update(string) error
    Delete(string) error
}
```

#### 3. **Clean Code**

**Naming:**
- Use **descriptive names**: `getUserByID` not `get`
- **Avoid abbreviations**: `configuration` not `cfg` (except common ones: `ID`, `HTTP`, `URL`)
- **Use camelCase** for unexported, **PascalCase** for exported
- **Interface names**: `Reader`, `Writer`, `Closer` (not `IReader`)

**Functions:**
- **Keep functions small** (<50 lines ideally)
- **Single level of abstraction** per function
- **Limit parameters** (max 3-4, use structs for more)
- **Return early** to reduce nesting

**Comments:**
- **Explain WHY, not WHAT** (code should be self-documenting)
- **GoDoc format** for exported functions
- **TODO comments** should include issue number

```go
// Good: Explains WHY
// We use a buffered channel to prevent blocking when the consumer
// is slower than the producer (issue #123)
messages := make(chan Message, 100)

// Bad: States the obvious
// Create a channel
messages := make(chan Message)
```

### Code Formatting

**Use `gofumpt`** (stricter than `gofmt`):
```bash
make fmt
```

**Run linter before committing:**
```bash
make lint
```

### Error Handling

**Always handle errors explicitly:**
```go
// Bad
result, _ := doSomething()

// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

**Use error wrapping:**
```go
// Wrap errors with context
if err := db.Connect(); err != nil {
    return fmt.Errorf("database connection failed: %w", err)
}
```

**Define custom errors for domain logic:**
```go
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

### Concurrency

**Use contexts for cancellation:**
```go
func (s *Server) ProcessRequest(ctx context.Context, req *Request) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-s.process(req):
        return result
    }
}
```

**Avoid goroutine leaks:**
```go
// Bad: Goroutine may leak
go func() {
    for {
        doWork()
    }
}()

// Good: Controlled shutdown
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            doWork()
        }
    }
}()
```

---

## 📝 Commit Guidelines

We follow **Conventional Commits** specification.

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Build process, tooling, dependencies
- `ci`: CI/CD changes

### Examples

```bash
feat(rest): add pagination support for list endpoints

Implements cursor-based pagination for all list endpoints.
Includes limit and offset query parameters.

Closes #42

---

fix(database): prevent connection pool exhaustion

Added connection pool size limits and proper cleanup.
Fixes issue where connections were not being released.

Fixes #123

---

docs(readme): update installation instructions

Added Docker installation method and troubleshooting section.

---

refactor(auth): extract JWT logic into separate package

Moved JWT generation and validation to pkg/auth for reusability.
No functional changes.
```

### Commit Best Practices

- **One logical change per commit**
- **Write in imperative mood**: "add feature" not "added feature"
- **Keep subject line under 72 characters**
- **Reference issues** in footer (`Closes #123`, `Fixes #456`)
- **Sign commits** (optional but recommended)

---

## 🔄 Pull Request Process

### Before Submitting

1. **Update from upstream:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests:**
   ```bash
   make test
   ```

3. **Run linter:**
   ```bash
   make lint
   ```

4. **Update documentation** if needed

5. **Add tests** for new features

### PR Checklist

- [ ] Code follows the style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass
- [ ] No linting errors
- [ ] Commit messages follow conventions
- [ ] PR description is clear

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How has this been tested?

## Checklist
- [ ] Tests pass
- [ ] Linter passes
- [ ] Documentation updated
```

### Review Process

1. **Automated checks** must pass (CI/CD)
2. **At least one maintainer** must approve
3. **Address feedback** promptly
4. **Squash commits** if requested
5. **Maintainer will merge** when ready

---

## 📂 Project Structure

See [docs/PROJECT_STRUCTURE.md](docs/PROJECT_STRUCTURE.md) for detailed explanation.

**Key points:**
- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public reusable libraries
- `api/` - API definitions (proto, OpenAPI)
- `test/` - Integration and E2E tests

---

## 🧪 Testing Guidelines

### Test Coverage

- **Aim for >80% coverage** for new code
- **100% coverage** for critical paths (auth, data handling)

### Test Types

**Unit Tests** (next to code):
```go
// internal/database/sqlite/adapter_test.go
func TestAdapter_Connect(t *testing.T) {
    adapter := NewAdapter(":memory:")
    err := adapter.Connect()
    assert.NoError(t, err)
}
```

**Integration Tests** (`test/integration/`):
```go
// test/integration/rest_api_test.go
func TestRESTAPI_CreateUser(t *testing.T) {
    // Test with real database
}
```

**E2E Tests** (`test/e2e/`):
```go
// test/e2e/full_flow_test.go
func TestFullUserFlow(t *testing.T) {
    // Test entire user journey
}
```

### Running Tests

```bash
# All tests
make test

# Unit tests only
go test ./...

# With coverage
make test-coverage

# Specific package
go test ./internal/database/...

# Verbose
go test -v ./...

# Race detector
go test -race ./...
```

### Test Best Practices

- **Use table-driven tests** for multiple scenarios
- **Use testify/assert** for assertions
- **Mock external dependencies** (use interfaces)
- **Clean up resources** (use `t.Cleanup()`)
- **Test edge cases** and error paths

**Example:**
```go
func TestParseConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Config
        wantErr bool
    }{
        {
            name:  "valid config",
            input: `{"port": 8080}`,
            want:  &Config{Port: 8080},
        },
        {
            name:    "invalid json",
            input:   `{invalid}`,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseConfig(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

---

## 📖 Documentation

### Code Documentation

**All exported functions must have GoDoc comments:**
```go
// NewAdapter creates a new SQLite database adapter.
// The dsn parameter should be a valid SQLite connection string.
// Returns an error if the connection cannot be established.
func NewAdapter(dsn string) (*Adapter, error) {
    // ...
}
```

### User Documentation

- **README.md** - Quick start and overview
- **docs/guides/** - Detailed tutorials
- **docs/api/** - API reference
- **examples/** - Working examples

### Architecture Documentation

**Use ADRs (Architecture Decision Records)** for significant decisions:

```markdown
# ADR 001: Use Chi Router for REST API

## Status
Accepted

## Context
We need a fast, lightweight HTTP router for the REST API.

## Decision
Use `go-chi/chi` router.

## Consequences
- Faster than Gin
- Middleware-friendly
- Standard library compatible
```

---

## 🎯 Development Workflow

### Feature Development

1. **Create feature branch:**
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make changes** following coding standards

3. **Write tests** for your changes

4. **Commit changes:**
   ```bash
   git add .
   git commit -m "feat(scope): description"
   ```

5. **Push to your fork:**
   ```bash
   git push origin feat/my-feature
   ```

6. **Create Pull Request** on GitHub

### Bug Fixes

1. **Create bug fix branch:**
   ```bash
   git checkout -b fix/issue-123
   ```

2. **Write failing test** that reproduces the bug

3. **Fix the bug**

4. **Verify test passes**

5. **Commit and push**

---

## 🏆 Recognition

Contributors will be:
- Listed in **CONTRIBUTORS.md**
- Mentioned in **release notes**
- Credited in **commit history**

---

## 📞 Getting Help

- **GitHub Discussions** - Ask questions
- **GitHub Issues** - Report bugs, request features
- **Discord** (coming soon) - Real-time chat

---

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to DatRi! 🚀**

*Let's build something amazing together!*

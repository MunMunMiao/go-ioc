# go-ioc

<p>
    <a href="https://deepwiki.com/MunMunMiao/go-ioc"><img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki"></a>
</p>

A lightweight, type-safe dependency injection container for Go, leveraging Go 1.18+ generics.

## Installation

```bash
go get github.com/MunMunMiao/go-ioc
```

## Overview

go-ioc is a reference-based dependency injection library that provides:

- **Type Safety** - Full compile-time type checking using Go generics
- **Zero Reflection** - No runtime reflection for core operations
- **Minimal API** - Just 4 main functions: `Provide`, `Inject`, `RunInInjectionContext`, `ResetGlobalInstances`
- **Concurrency Safe** - Thread-safe singleton management
- **Circular Dependency Detection** - Runtime detection with clear error messages
- **Hierarchical Injection** - Local provider overrides for testing and modular design

## Quick Start

```go
package main

import (
    "fmt"
    ioc "github.com/MunMunMiao/go-ioc"
)

// Define a service
type UserService struct {
    Name string
}

// Create a provider reference
var UserServiceRef = ioc.Provide(func(ctx *ioc.Context) *UserService {
    return &UserService{Name: "default"}
})

func main() {
    ioc.RunInInjectionContext(func(ctx *ioc.Context) any {
        // Inject the service
        service := ioc.Inject(ctx, UserServiceRef)
        fmt.Println(service.Name) // Output: default
        return nil
    })
}
```

## API Reference

### Provide

Creates a dependency provider reference.

```go
func Provide[T any](factory func(ctx *Context) T, opts ...ProvideOptions[T]) *Ref[T]
```

**Parameters:**
- `factory` - Factory function that creates the instance
- `opts` - Optional configuration

**Options:**
| Field | Type | Description |
|-------|------|-------------|
| `Mode` | `Mode` | `ModeGlobal` (default) or `ModeStandalone` |
| `Providers` | `[]any` | Local provider overrides |
| `Overrides` | `any` | Target reference to override |

### Inject

Retrieves a dependency from the context.

```go
func Inject[T any](ctx *Context, ref *Ref[T]) T
```

### RunInInjectionContext

Executes a function within an injection context.

```go
func RunInInjectionContext[T any](fn func(ctx *Context) T) T
```

### ResetGlobalInstances

Clears all cached global instances. Useful for testing.

```go
func ResetGlobalInstances()
```

## Instance Modes

### Global Mode (Default)

Singleton pattern - one instance shared across all contexts.

```go
var ConfigRef = ioc.Provide(func(ctx *ioc.Context) *Config {
    return &Config{Env: "production"}
})
```

### Standalone Mode

New instance per injection context.

```go
var RequestRef = ioc.Provide(func(ctx *ioc.Context) *Request {
    return &Request{ID: generateID()}
}, ioc.ProvideOptions[*Request]{
    Mode: ioc.ModeStandalone,
})
```

## Dependency Injection

Services can inject other services:

```go
type Config struct {
    DatabaseURL string
}

type Database struct {
    URL string
}

var ConfigRef = ioc.Provide(func(ctx *ioc.Context) *Config {
    return &Config{DatabaseURL: "postgres://localhost/db"}
})

var DatabaseRef = ioc.Provide(func(ctx *ioc.Context) *Database {
    config := ioc.Inject(ctx, ConfigRef)  // Inject Config
    return &Database{URL: config.DatabaseURL}
})
```

## Local Providers (Overrides)

Override dependencies for specific scopes, useful for testing:

```go
// Production config
var ConfigRef = ioc.Provide(func(ctx *ioc.Context) *Config {
    return &Config{Env: "production"}
})

// Test config that overrides ConfigRef
var TestConfigRef = ioc.Provide(func(ctx *ioc.Context) *Config {
    return &Config{Env: "test"}
}, ioc.ProvideOptions[*Config]{
    Overrides: ConfigRef,
})

// Service with local provider
var ServiceRef = ioc.Provide(func(ctx *ioc.Context) *Service {
    config := ioc.Inject(ctx, ConfigRef)  // Gets TestConfigRef in this scope
    return &Service{Config: config}
}, ioc.ProvideOptions[*Service]{
    Providers: []any{TestConfigRef},
})
```

## Testing

Use `ResetGlobalInstances()` to ensure test isolation:

```go
func TestService(t *testing.T) {
    ioc.ResetGlobalInstances()  // Clean state
    
    result := ioc.RunInInjectionContext(func(ctx *ioc.Context) string {
        service := ioc.Inject(ctx, ServiceRef)
        return service.GetValue()
    })
    
    assert.Equal(t, "expected", result)
}
```

## Comparison with Other DI Libraries

| Feature | go-ioc | wire | dig | fx |
|---------|--------|------|-----|-----|
| **Approach** | Runtime | Code Generation | Runtime | Runtime |
| **Type Safety** | ✅ Generics | ✅ Generated | ⚠️ Reflection | ⚠️ Reflection |
| **Reflection** | ❌ None | ❌ None | ✅ Heavy | ✅ Heavy |
| **Learning Curve** | Low | Medium | Medium | High |
| **Lines of Code** | ~150 | N/A | ~3000+ | ~5000+ |
| **Circular Detection** | ✅ Runtime | ✅ Compile-time | ✅ Runtime | ✅ Runtime |
| **Local Overrides** | ✅ Built-in | ❌ Manual | ⚠️ Scopes | ⚠️ Modules |
| **Singleton/Transient** | ✅ | ✅ | ✅ | ✅ |

### When to Use go-ioc

✅ **Choose go-ioc when:**
- You want minimal, readable dependency injection
- Type safety without code generation is important
- You need simple testing with local overrides
- Your project is small to medium sized

### When to Consider Alternatives

- **wire** - Large projects needing compile-time validation
- **dig/fx** - Complex applications requiring advanced lifecycle management

## Examples

### MVC Pattern

```go
// Repository
var UserRepoRef = ioc.Provide(func(ctx *ioc.Context) *UserRepository {
    return &UserRepository{}
})

// Service
var UserServiceRef = ioc.Provide(func(ctx *ioc.Context) *UserService {
    return &UserService{
        Repo: ioc.Inject(ctx, UserRepoRef),
    }
})

// Controller
var UserControllerRef = ioc.Provide(func(ctx *ioc.Context) *UserController {
    return &UserController{
        Service: ioc.Inject(ctx, UserServiceRef),
    }
})
```

### DDD Pattern

```go
// Domain Layer - Repository Interface
type OrderRepository interface {
    Save(order *Order) error
    FindByID(id string) (*Order, error)
}

// Infrastructure Layer - Implementation
var OrderRepoRef = ioc.Provide(func(ctx *ioc.Context) OrderRepository {
    return &PostgresOrderRepository{}
})

// Application Layer - Use Case
var CreateOrderRef = ioc.Provide(func(ctx *ioc.Context) *CreateOrderUseCase {
    return &CreateOrderUseCase{
        Repo: ioc.Inject(ctx, OrderRepoRef),
    }
})
```

See the [example](./example) directory for complete MVC and DDD examples.

## License

MIT

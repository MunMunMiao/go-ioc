package ioc

import (
	"fmt"
	"sync"
)

// Mode defines how instances are cached
type Mode int

const (
	// ModeGlobal creates a singleton shared across all contexts
	ModeGlobal Mode = iota
	// ModeStandalone creates a new instance per context
	ModeStandalone
)

// refMarker is an interface to identify Ref types without reflection
type refMarker interface {
	isProvideRef() bool
	getOverride() any
}

// Ref is a reference to a dependency provider
type Ref[T any] struct {
	factory   func(ctx *Context) T
	mode      Mode
	providers []any
	override  any
}

// isProvideRef implements refMarker interface
func (r *Ref[T]) isProvideRef() bool {
	return true
}

// getOverride implements refMarker interface
func (r *Ref[T]) getOverride() any {
	return r.override
}

// ProvideOptions configures a provider
type ProvideOptions[T any] struct {
	Mode      Mode
	Providers []any
	Overrides any
}

// Context holds injection state
type Context struct {
	instances      map[any]any
	localProviders map[any]any
	creating       map[any]bool
	parent         *Context
}

var (
	globalInstances = make(map[any]any)
	globalCreating  = make(map[any]bool)
	globalMu        sync.RWMutex
)

// Provide creates a new dependency provider
func Provide[T any](factory func(ctx *Context) T, opts ...ProvideOptions[T]) *Ref[T] {
	ref := &Ref[T]{
		factory: factory,
		mode:    ModeGlobal,
	}

	if len(opts) > 0 {
		opt := opts[0]
		ref.mode = opt.Mode
		ref.providers = opt.Providers
		if opt.Overrides != nil {
			ref.override = opt.Overrides
		}
	}

	return ref
}

// Inject retrieves a dependency from the context
func Inject[T any](ctx *Context, ref *Ref[T]) T {
	actualRef := findRefInContext(ctx, ref)
	isGlobal := actualRef.mode == ModeGlobal
	useGlobalCache := isGlobal && ctx.parent == nil

	// Check cache
	if useGlobalCache {
		globalMu.RLock()
		if instance, ok := globalInstances[actualRef]; ok {
			globalMu.RUnlock()
			return instance.(T)
		}
		globalMu.RUnlock()
	} else {
		if instance, ok := ctx.instances[actualRef]; ok {
			return instance.(T)
		}
	}

	// Circular dependency detection
	if useGlobalCache {
		globalMu.Lock()
		if globalCreating[actualRef] {
			globalMu.Unlock()
			panic(fmt.Sprintf("Circular dependency detected: Ref(%p)", actualRef))
		}
		globalCreating[actualRef] = true
		globalMu.Unlock()
		defer func() {
			globalMu.Lock()
			delete(globalCreating, actualRef)
			globalMu.Unlock()
		}()
	} else {
		if ctx.creating[actualRef] {
			panic(fmt.Sprintf("Circular dependency detected: Ref(%p)", actualRef))
		}
		ctx.creating[actualRef] = true
		defer delete(ctx.creating, actualRef)
	}

	// Create instance
	var instance T
	if len(actualRef.providers) > 0 {
		childCtx := createContext(ctx)
		for _, provider := range actualRef.providers {
			registerProvider(childCtx, provider)
		}
		instance = actualRef.factory(childCtx)
	} else {
		instance = actualRef.factory(ctx)
	}

	// Cache instance
	if useGlobalCache {
		globalMu.Lock()
		globalInstances[actualRef] = instance
		globalMu.Unlock()
	} else {
		ctx.instances[actualRef] = instance
	}

	return instance
}

// RunInInjectionContext executes a function within an injection context
func RunInInjectionContext[T any](fn func(ctx *Context) T) T {
	ctx := createContext(nil)
	return fn(ctx)
}

// ResetGlobalInstances clears all cached global instances (for testing)
func ResetGlobalInstances() {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalInstances = make(map[any]any)
	globalCreating = make(map[any]bool)
}

// IsProvideRef checks if a value is a Ref (without reflection)
func IsProvideRef(value any) bool {
	if value == nil {
		return false
	}
	_, ok := value.(refMarker)
	return ok
}

func createContext(parent *Context) *Context {
	return &Context{
		instances:      make(map[any]any),
		localProviders: make(map[any]any),
		creating:       make(map[any]bool),
		parent:         parent,
	}
}

func findRefInContext[T any](ctx *Context, ref *Ref[T]) *Ref[T] {
	current := ctx
	for current != nil {
		if localRef, ok := current.localProviders[ref]; ok {
			return localRef.(*Ref[T])
		}
		current = current.parent
	}
	return ref
}

func registerProvider(ctx *Context, provider any) {
	// Extract override target from provider
	override := extractOverride(provider)
	if override != nil {
		ctx.localProviders[override] = provider
	} else {
		ctx.localProviders[provider] = provider
	}
}

func extractOverride(provider any) any {
	if marker, ok := provider.(refMarker); ok {
		return marker.getOverride()
	}
	return nil
}

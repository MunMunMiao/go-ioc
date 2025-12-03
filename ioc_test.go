package ioc

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBasicProvideAndInject(t *testing.T) {
	ResetGlobalInstances()

	ref := Provide(func(ctx *Context) string {
		return "test value"
	})

	result := RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, ref)
	})

	if result != "test value" {
		t.Errorf("expected 'test value', got '%s'", result)
	}
}

func TestFactoryReceivesContext(t *testing.T) {
	ResetGlobalInstances()

	type Config struct {
		APIUrl string
	}

	configRef := Provide(func(ctx *Context) *Config {
		return &Config{APIUrl: "https://api.example.com"}
	})

	type Service struct {
		GetEndpoint func(path string) string
	}

	serviceRef := Provide(func(ctx *Context) *Service {
		config := Inject(ctx, configRef)
		return &Service{
			GetEndpoint: func(path string) string {
				return config.APIUrl + path
			},
		}
	})

	result := RunInInjectionContext(func(ctx *Context) string {
		service := Inject(ctx, serviceRef)
		return service.GetEndpoint("/users")
	})

	if result != "https://api.example.com/users" {
		t.Errorf("expected 'https://api.example.com/users', got '%s'", result)
	}
}

func TestGlobalModeSharesInstances(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	ref := Provide(func(ctx *Context) int32 {
		return atomic.AddInt32(&counter, 1)
	})

	value1 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	value2 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	if value1 != value2 {
		t.Errorf("expected same instance, got %d and %d", value1, value2)
	}
	if counter != 1 {
		t.Errorf("expected counter to be 1, got %d", counter)
	}
}

func TestStandaloneModeCreatesNewInstances(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	ref := Provide(func(ctx *Context) int32 {
		return atomic.AddInt32(&counter, 1)
	}, ProvideOptions[int32]{Mode: ModeStandalone})

	value1 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	value2 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	if value1 == value2 {
		t.Errorf("expected different instances, got same value %d", value1)
	}
	if counter != 2 {
		t.Errorf("expected counter to be 2, got %d", counter)
	}
}

func TestNestedDependencies(t *testing.T) {
	ResetGlobalInstances()

	type DB struct {
		Query func(sql string) string
	}

	dbRef := Provide(func(ctx *Context) *DB {
		return &DB{
			Query: func(sql string) string {
				return "Result: " + sql
			},
		}
	})

	type UserRepo struct {
		FindUser func(id int) string
	}

	userRepoRef := Provide(func(ctx *Context) *UserRepo {
		db := Inject(ctx, dbRef)
		return &UserRepo{
			FindUser: func(id int) string {
				return db.Query("SELECT * FROM users WHERE id = " + string(rune('0'+id)))
			},
		}
	})

	type UserService struct {
		GetUser func(id int) string
	}

	userServiceRef := Provide(func(ctx *Context) *UserService {
		repo := Inject(ctx, userRepoRef)
		return &UserService{
			GetUser: func(id int) string {
				return repo.FindUser(id)
			},
		}
	})

	result := RunInInjectionContext(func(ctx *Context) string {
		service := Inject(ctx, userServiceRef)
		return service.GetUser(1)
	})

	expected := "Result: SELECT * FROM users WHERE id = 1"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestLocalProvidersOverride(t *testing.T) {
	ResetGlobalInstances()

	configRef := Provide(func(ctx *Context) string {
		return "global config"
	})

	type Service struct {
		Config string
	}

	localConfigRef := Provide(func(ctx *Context) string {
		return "local config"
	}, ProvideOptions[string]{Overrides: configRef})

	serviceRef := Provide(func(ctx *Context) *Service {
		config := Inject(ctx, configRef)
		return &Service{Config: config}
	}, ProvideOptions[*Service]{
		Providers: []any{localConfigRef},
	})

	RunInInjectionContext(func(ctx *Context) any {
		globalConfig := Inject(ctx, configRef)
		service := Inject(ctx, serviceRef)

		if globalConfig != "global config" {
			t.Errorf("expected 'global config', got '%s'", globalConfig)
		}
		if service.Config != "local config" {
			t.Errorf("expected 'local config', got '%s'", service.Config)
		}
		return nil
	})
}

func TestInjectSameRefMultipleTimesReturnsSameInstance(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	type Instance struct {
		ID int32
	}

	ref := Provide(func(ctx *Context) *Instance {
		return &Instance{ID: atomic.AddInt32(&counter, 1)}
	})

	RunInInjectionContext(func(ctx *Context) any {
		a := Inject(ctx, ref)
		b := Inject(ctx, ref)
		c := Inject(ctx, ref)

		if a != b || b != c {
			t.Error("expected same instance for all injections")
		}
		if counter != 1 {
			t.Errorf("expected counter to be 1, got %d", counter)
		}
		return nil
	})
}

func TestStandaloneRefInSameContextReturnsSameInstance(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	type Instance struct {
		ID int32
	}

	ref := Provide(func(ctx *Context) *Instance {
		return &Instance{ID: atomic.AddInt32(&counter, 1)}
	}, ProvideOptions[*Instance]{Mode: ModeStandalone})

	RunInInjectionContext(func(ctx *Context) any {
		a := Inject(ctx, ref)
		b := Inject(ctx, ref)

		if a != b {
			t.Error("expected same instance within same context")
		}
		if counter != 1 {
			t.Errorf("expected counter to be 1, got %d", counter)
		}
		return nil
	})
}

func TestFactoryReturningPrimitiveValues(t *testing.T) {
	ResetGlobalInstances()

	numRef := Provide(func(ctx *Context) int { return 42 })
	strRef := Provide(func(ctx *Context) string { return "hello" })
	boolRef := Provide(func(ctx *Context) bool { return true })

	RunInInjectionContext(func(ctx *Context) any {
		if Inject(ctx, numRef) != 42 {
			t.Error("expected 42")
		}
		if Inject(ctx, strRef) != "hello" {
			t.Error("expected 'hello'")
		}
		if Inject(ctx, boolRef) != true {
			t.Error("expected true")
		}
		return nil
	})
}

func TestFactoryReturningFunction(t *testing.T) {
	ResetGlobalInstances()

	fnRef := Provide(func(ctx *Context) func(int) int {
		return func(x int) int { return x * 2 }
	})

	result := RunInInjectionContext(func(ctx *Context) int {
		fn := Inject(ctx, fnRef)
		return fn(5)
	})

	if result != 10 {
		t.Errorf("expected 10, got %d", result)
	}
}

func TestFactoryReturningSlice(t *testing.T) {
	ResetGlobalInstances()

	arrRef := Provide(func(ctx *Context) []int {
		return []int{1, 2, 3}
	})

	result := RunInInjectionContext(func(ctx *Context) []int {
		return Inject(ctx, arrRef)
	})

	if len(result) != 3 || result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Errorf("expected [1, 2, 3], got %v", result)
	}
}

func TestFactoryThrowingError(t *testing.T) {
	ResetGlobalInstances()

	ref := Provide(func(ctx *Context) string {
		panic("Factory error")
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		} else if r != "Factory error" {
			t.Errorf("expected 'Factory error', got '%v'", r)
		}
	}()

	RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, ref)
	})
}

func TestCircularDependencyThrowsError(t *testing.T) {
	ResetGlobalInstances()

	var aRef, bRef *Ref[string]

	aRef = Provide(func(ctx *Context) string {
		return Inject(ctx, bRef)
	})

	bRef = Provide(func(ctx *Context) string {
		return Inject(ctx, aRef)
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for circular dependency")
		}
	}()

	RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, aRef)
	})
}

func TestSelfReferencingCircularDependency(t *testing.T) {
	ResetGlobalInstances()

	var selfRef *Ref[string]
	selfRef = Provide(func(ctx *Context) string {
		return Inject(ctx, selfRef)
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for self-referencing circular dependency")
		}
	}()

	RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, selfRef)
	})
}

func TestThreeWayCircularDependency(t *testing.T) {
	ResetGlobalInstances()

	var aRef, bRef, cRef *Ref[string]

	aRef = Provide(func(ctx *Context) string {
		return Inject(ctx, bRef)
	})
	bRef = Provide(func(ctx *Context) string {
		return Inject(ctx, cRef)
	})
	cRef = Provide(func(ctx *Context) string {
		return Inject(ctx, aRef)
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for three-way circular dependency")
		}
	}()

	RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, aRef)
	})
}

func TestDiamondDependencyPattern(t *testing.T) {
	ResetGlobalInstances()

	type Base struct {
		Value string
	}

	baseRef := Provide(func(ctx *Context) *Base {
		return &Base{Value: "base"}
	})

	type Left struct {
		Base *Base
		Side string
	}

	leftRef := Provide(func(ctx *Context) *Left {
		return &Left{
			Base: Inject(ctx, baseRef),
			Side: "left",
		}
	})

	type Right struct {
		Base *Base
		Side string
	}

	rightRef := Provide(func(ctx *Context) *Right {
		return &Right{
			Base: Inject(ctx, baseRef),
			Side: "right",
		}
	})

	type Top struct {
		Left  *Left
		Right *Right
	}

	topRef := Provide(func(ctx *Context) *Top {
		return &Top{
			Left:  Inject(ctx, leftRef),
			Right: Inject(ctx, rightRef),
		}
	})

	RunInInjectionContext(func(ctx *Context) any {
		top := Inject(ctx, topRef)

		if top.Left.Base != top.Right.Base {
			t.Error("expected same base instance in diamond pattern")
		}
		if top.Left.Side != "left" {
			t.Error("expected left side to be 'left'")
		}
		if top.Right.Side != "right" {
			t.Error("expected right side to be 'right'")
		}
		return nil
	})
}

func TestResetGlobalInstances(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	ref := Provide(func(ctx *Context) int32 {
		return atomic.AddInt32(&counter, 1)
	})

	value1 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	if value1 != 1 {
		t.Errorf("expected 1, got %d", value1)
	}

	ResetGlobalInstances()

	value2 := RunInInjectionContext(func(ctx *Context) int32 {
		return Inject(ctx, ref)
	})

	if value2 != 2 {
		t.Errorf("expected 2, got %d", value2)
	}
}

func TestGlobalDependencyUsedByStandaloneParent(t *testing.T) {
	ResetGlobalInstances()

	var globalCounter int32

	type GlobalInstance struct {
		ID int32
	}

	globalRef := Provide(func(ctx *Context) *GlobalInstance {
		return &GlobalInstance{ID: atomic.AddInt32(&globalCounter, 1)}
	})

	type Wrapper struct {
		Global *GlobalInstance
	}

	standaloneRef := Provide(func(ctx *Context) *Wrapper {
		return &Wrapper{Global: Inject(ctx, globalRef)}
	}, ProvideOptions[*Wrapper]{Mode: ModeStandalone})

	result1 := RunInInjectionContext(func(ctx *Context) *Wrapper {
		return Inject(ctx, standaloneRef)
	})

	result2 := RunInInjectionContext(func(ctx *Context) *Wrapper {
		return Inject(ctx, standaloneRef)
	})

	// Standalone creates new wrapper, but global dependency is shared
	if result1 == result2 {
		t.Error("expected different wrapper instances")
	}
	if result1.Global != result2.Global {
		t.Error("expected same global instance")
	}
	if globalCounter != 1 {
		t.Errorf("expected globalCounter to be 1, got %d", globalCounter)
	}
}

func TestStandaloneDependencyUsedByGlobalParent(t *testing.T) {
	ResetGlobalInstances()

	var standaloneCounter int32

	type StandaloneInstance struct {
		ID int32
	}

	standaloneRef := Provide(func(ctx *Context) *StandaloneInstance {
		return &StandaloneInstance{ID: atomic.AddInt32(&standaloneCounter, 1)}
	}, ProvideOptions[*StandaloneInstance]{Mode: ModeStandalone})

	type Wrapper struct {
		Standalone *StandaloneInstance
	}

	globalRef := Provide(func(ctx *Context) *Wrapper {
		return &Wrapper{Standalone: Inject(ctx, standaloneRef)}
	})

	result1 := RunInInjectionContext(func(ctx *Context) *Wrapper {
		return Inject(ctx, globalRef)
	})

	result2 := RunInInjectionContext(func(ctx *Context) *Wrapper {
		return Inject(ctx, globalRef)
	})

	// Global parent is cached, so standalone is only created once
	if result1 != result2 {
		t.Error("expected same wrapper instance")
	}
	if result1.Standalone != result2.Standalone {
		t.Error("expected same standalone instance")
	}
	if standaloneCounter != 1 {
		t.Errorf("expected standaloneCounter to be 1, got %d", standaloneCounter)
	}
}

func TestIsProvideRef(t *testing.T) {
	ResetGlobalInstances()

	ref := Provide(func(ctx *Context) string { return "value" })

	if !IsProvideRef(ref) {
		t.Error("expected IsProvideRef to return true for Ref")
	}
	if IsProvideRef(nil) {
		t.Error("expected IsProvideRef to return false for nil")
	}
	if IsProvideRef("not a ref") {
		t.Error("expected IsProvideRef to return false for string")
	}
}

func TestConcurrentContexts(t *testing.T) {
	ResetGlobalInstances()

	var counter int32
	ref := Provide(func(ctx *Context) int32 {
		return atomic.AddInt32(&counter, 1)
	}, ProvideOptions[int32]{Mode: ModeStandalone})

	var wg sync.WaitGroup
	results := make([]int32, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			time.Sleep(time.Duration(idx) * 10 * time.Millisecond)
			results[idx] = RunInInjectionContext(func(ctx *Context) int32 {
				return Inject(ctx, ref)
			})
		}(i)
	}

	wg.Wait()

	// Check all results are unique
	seen := make(map[int32]bool)
	for _, v := range results {
		if seen[v] {
			t.Errorf("duplicate value found: %d", v)
		}
		seen[v] = true
	}
}

func TestContextIsolation(t *testing.T) {
	ResetGlobalInstances()

	ref := Provide(func(ctx *Context) float64 {
		return float64(time.Now().UnixNano())
	}, ProvideOptions[float64]{Mode: ModeStandalone})

	var wg sync.WaitGroup
	var mu sync.Mutex
	type Result struct {
		Step string
		ID   float64
	}
	results := make([]Result, 0)

	wg.Add(2)

	go func() {
		defer wg.Done()
		RunInInjectionContext(func(ctx *Context) any {
			id := Inject(ctx, ref)
			mu.Lock()
			results = append(results, Result{"A-start", id})
			mu.Unlock()

			time.Sleep(30 * time.Millisecond)

			mu.Lock()
			results = append(results, Result{"A-middle", id})
			mu.Unlock()

			id2 := Inject(ctx, ref)
			if id != id2 {
				t.Error("expected same ID within context A")
			}

			time.Sleep(30 * time.Millisecond)

			mu.Lock()
			results = append(results, Result{"A-end", id})
			mu.Unlock()
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		RunInInjectionContext(func(ctx *Context) any {
			id := Inject(ctx, ref)
			mu.Lock()
			results = append(results, Result{"B-start", id})
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			results = append(results, Result{"B-middle", id})
			mu.Unlock()

			id2 := Inject(ctx, ref)
			if id != id2 {
				t.Error("expected same ID within context B")
			}

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			results = append(results, Result{"B-end", id})
			mu.Unlock()
			return nil
		})
	}()

	wg.Wait()

	// Verify A results all have same ID
	var aID float64
	for _, r := range results {
		if r.Step[0] == 'A' {
			if aID == 0 {
				aID = r.ID
			} else if aID != r.ID {
				t.Error("context A should have consistent ID")
			}
		}
	}

	// Verify B results all have same ID
	var bID float64
	for _, r := range results {
		if r.Step[0] == 'B' {
			if bID == 0 {
				bID = r.ID
			} else if bID != r.ID {
				t.Error("context B should have consistent ID")
			}
		}
	}

	// Verify A and B have different IDs
	if aID == bID {
		t.Error("contexts A and B should have different IDs")
	}
}

func TestEmptyProvidersArray(t *testing.T) {
	ResetGlobalInstances()

	ref := Provide(func(ctx *Context) string {
		return "value"
	}, ProvideOptions[string]{Providers: []any{}})

	result := RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, ref)
	})

	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

func TestDeepDependencyChainWithOverride(t *testing.T) {
	ResetGlobalInstances()

	type Config struct {
		URL string
	}

	configRef := Provide(func(ctx *Context) *Config {
		return &Config{URL: "prod.com"}
	})

	type HTTP struct {
		BaseURL string
	}

	httpRef := Provide(func(ctx *Context) *HTTP {
		config := Inject(ctx, configRef)
		return &HTTP{BaseURL: config.URL}
	}, ProvideOptions[*HTTP]{Mode: ModeStandalone})

	type API struct {
		Endpoint string
	}

	apiRef := Provide(func(ctx *Context) *API {
		http := Inject(ctx, httpRef)
		return &API{Endpoint: "https://" + http.BaseURL + "/api"}
	}, ProvideOptions[*API]{Mode: ModeStandalone})

	testConfigRef := Provide(func(ctx *Context) *Config {
		return &Config{URL: "test.com"}
	}, ProvideOptions[*Config]{Overrides: configRef})

	appRef := Provide(func(ctx *Context) *API {
		return Inject(ctx, apiRef)
	}, ProvideOptions[*API]{
		Providers: []any{testConfigRef},
	})

	result := RunInInjectionContext(func(ctx *Context) string {
		return Inject(ctx, appRef).Endpoint
	})

	if result != "https://test.com/api" {
		t.Errorf("expected 'https://test.com/api', got '%s'", result)
	}
}

func TestMultipleProvidersInSingleRef(t *testing.T) {
	ResetGlobalInstances()

	aRef := Provide(func(ctx *Context) string { return "A" })
	bRef := Provide(func(ctx *Context) string { return "B" })

	localARef := Provide(func(ctx *Context) string {
		return "X"
	}, ProvideOptions[string]{Overrides: aRef})

	localBRef := Provide(func(ctx *Context) string {
		return "Y"
	}, ProvideOptions[string]{Overrides: bRef})

	combinedRef := Provide(func(ctx *Context) string {
		return Inject(ctx, aRef) + "-" + Inject(ctx, bRef)
	}, ProvideOptions[string]{
		Providers: []any{localARef, localBRef},
	})

	RunInInjectionContext(func(ctx *Context) any {
		if Inject(ctx, aRef) != "A" {
			t.Error("expected global A")
		}
		if Inject(ctx, bRef) != "B" {
			t.Error("expected global B")
		}
		if Inject(ctx, combinedRef) != "X-Y" {
			t.Errorf("expected 'X-Y', got '%s'", Inject(ctx, combinedRef))
		}
		return nil
	})
}

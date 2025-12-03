// DDD Example - Minimal Domain-Driven Design pattern using IoC container
//
// This example demonstrates a simple DDD architecture:
// - Domain Layer: Entities, Value Objects, Domain Services
// - Application Layer: Use Cases / Application Services
// - Infrastructure Layer: Repository implementations
// - Interface Layer: API handlers
package main

import (
	"errors"
	"fmt"
	"time"

	ioc "github.com/MunMunMiao/go-ioc"
)

// ============================================================================
// Domain Layer - Entities and Value Objects
// ============================================================================

// Value Object
type Money struct {
	Amount   float64
	Currency string
}

func (m Money) Add(other Money) Money {
	if m.Currency != other.Currency {
		panic("Currency mismatch")
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}
}

// Entity
type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
}

type OrderItem struct {
	ProductID string
	Name      string
	Price     Money
	Quantity  int
}

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
)

func (o *Order) TotalAmount() Money {
	total := Money{Amount: 0, Currency: "USD"}
	for _, item := range o.Items {
		itemTotal := Money{Amount: item.Price.Amount * float64(item.Quantity), Currency: item.Price.Currency}
		total = total.Add(itemTotal)
	}
	return total
}

func (o *Order) Confirm() error {
	if o.Status != OrderStatusPending {
		return errors.New("order can only be confirmed when pending")
	}
	o.Status = OrderStatusConfirmed
	return nil
}

// ============================================================================
// Domain Layer - Repository Interface (Port)
// ============================================================================

type OrderRepository interface {
	Save(order *Order) error
	FindByID(id string) (*Order, error)
	FindByCustomer(customerID string) ([]*Order, error)
}

// ============================================================================
// Domain Layer - Domain Service
// ============================================================================

type PricingService struct{}

func (s *PricingService) ApplyDiscount(order *Order, discountPercent float64) {
	for i := range order.Items {
		order.Items[i].Price.Amount *= (1 - discountPercent/100)
	}
}

var PricingServiceRef = ioc.Provide(func(ctx *ioc.Context) *PricingService {
	return &PricingService{}
})

// ============================================================================
// Infrastructure Layer - Repository Implementation (Adapter)
// ============================================================================

type InMemoryOrderRepository struct {
	orders map[string]*Order
}

func (r *InMemoryOrderRepository) Save(order *Order) error {
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryOrderRepository) FindByID(id string) (*Order, error) {
	if order, ok := r.orders[id]; ok {
		return order, nil
	}
	return nil, errors.New("order not found")
}

func (r *InMemoryOrderRepository) FindByCustomer(customerID string) ([]*Order, error) {
	var result []*Order
	for _, order := range r.orders {
		if order.CustomerID == customerID {
			result = append(result, order)
		}
	}
	return result, nil
}

var OrderRepositoryRef = ioc.Provide(func(ctx *ioc.Context) OrderRepository {
	return &InMemoryOrderRepository{
		orders: make(map[string]*Order),
	}
})

// ============================================================================
// Application Layer - Use Cases
// ============================================================================

type CreateOrderInput struct {
	CustomerID string
	Items      []OrderItem
}

type CreateOrderUseCase struct {
	orderRepo      OrderRepository
	pricingService *PricingService
}

func (uc *CreateOrderUseCase) Execute(input CreateOrderInput) (*Order, error) {
	order := &Order{
		ID:         fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
		CustomerID: input.CustomerID,
		Items:      input.Items,
		Status:     OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	// Apply domain logic
	if order.TotalAmount().Amount > 100 {
		uc.pricingService.ApplyDiscount(order, 10) // 10% discount for orders > $100
	}

	if err := uc.orderRepo.Save(order); err != nil {
		return nil, err
	}

	return order, nil
}

var CreateOrderUseCaseRef = ioc.Provide(func(ctx *ioc.Context) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo:      ioc.Inject(ctx, OrderRepositoryRef),
		pricingService: ioc.Inject(ctx, PricingServiceRef),
	}
})

type ConfirmOrderUseCase struct {
	orderRepo OrderRepository
}

func (uc *ConfirmOrderUseCase) Execute(orderID string) error {
	order, err := uc.orderRepo.FindByID(orderID)
	if err != nil {
		return err
	}

	if err := order.Confirm(); err != nil {
		return err
	}

	return uc.orderRepo.Save(order)
}

var ConfirmOrderUseCaseRef = ioc.Provide(func(ctx *ioc.Context) *ConfirmOrderUseCase {
	return &ConfirmOrderUseCase{
		orderRepo: ioc.Inject(ctx, OrderRepositoryRef),
	}
})

type GetCustomerOrdersUseCase struct {
	orderRepo OrderRepository
}

func (uc *GetCustomerOrdersUseCase) Execute(customerID string) ([]*Order, error) {
	return uc.orderRepo.FindByCustomer(customerID)
}

var GetCustomerOrdersUseCaseRef = ioc.Provide(func(ctx *ioc.Context) *GetCustomerOrdersUseCase {
	return &GetCustomerOrdersUseCase{
		orderRepo: ioc.Inject(ctx, OrderRepositoryRef),
	}
})

// ============================================================================
// Interface Layer - API Handler
// ============================================================================

type OrderHandler struct {
	createOrder       *CreateOrderUseCase
	confirmOrder      *ConfirmOrderUseCase
	getCustomerOrders *GetCustomerOrdersUseCase
}

func (h *OrderHandler) HandleCreateOrder(customerID string, items []OrderItem) {
	order, err := h.createOrder.Execute(CreateOrderInput{
		CustomerID: customerID,
		Items:      items,
	})
	if err != nil {
		fmt.Printf("Error creating order: %v\n", err)
		return
	}
	fmt.Printf("Order created: %s, Total: $%.2f\n", order.ID, order.TotalAmount().Amount)
}

func (h *OrderHandler) HandleConfirmOrder(orderID string) {
	if err := h.confirmOrder.Execute(orderID); err != nil {
		fmt.Printf("Error confirming order: %v\n", err)
		return
	}
	fmt.Printf("Order %s confirmed\n", orderID)
}

func (h *OrderHandler) HandleGetCustomerOrders(customerID string) {
	orders, err := h.getCustomerOrders.Execute(customerID)
	if err != nil {
		fmt.Printf("Error getting orders: %v\n", err)
		return
	}
	fmt.Printf("Customer %s has %d order(s)\n", customerID, len(orders))
	for _, order := range orders {
		fmt.Printf("  - %s: %s ($%.2f)\n", order.ID, order.Status, order.TotalAmount().Amount)
	}
}

var OrderHandlerRef = ioc.Provide(func(ctx *ioc.Context) *OrderHandler {
	return &OrderHandler{
		createOrder:       ioc.Inject(ctx, CreateOrderUseCaseRef),
		confirmOrder:      ioc.Inject(ctx, ConfirmOrderUseCaseRef),
		getCustomerOrders: ioc.Inject(ctx, GetCustomerOrdersUseCaseRef),
	}
})

// ============================================================================
// Application Entry Point
// ============================================================================

func main() {
	ioc.RunInInjectionContext(func(ctx *ioc.Context) any {
		handler := ioc.Inject(ctx, OrderHandlerRef)

		fmt.Println("=== DDD Order System Demo ===\n")

		// Create an order with items
		items := []OrderItem{
			{ProductID: "P001", Name: "Laptop", Price: Money{Amount: 999.99, Currency: "USD"}, Quantity: 1},
			{ProductID: "P002", Name: "Mouse", Price: Money{Amount: 29.99, Currency: "USD"}, Quantity: 2},
		}

		fmt.Println("1. Creating order for customer C001...")
		handler.HandleCreateOrder("C001", items)

		fmt.Println("\n2. Getting customer orders...")
		handler.HandleGetCustomerOrders("C001")

		// Note: In real scenario, we'd use the actual order ID from step 1
		// This is just for demonstration
		fmt.Println("\n3. Creating another small order (no discount)...")
		smallItems := []OrderItem{
			{ProductID: "P003", Name: "USB Cable", Price: Money{Amount: 9.99, Currency: "USD"}, Quantity: 1},
		}
		handler.HandleCreateOrder("C001", smallItems)

		fmt.Println("\n4. Final customer orders...")
		handler.HandleGetCustomerOrders("C001")

		return nil
	})
}

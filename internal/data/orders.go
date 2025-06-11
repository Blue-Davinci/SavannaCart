package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/database"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
)

var (
	ErrOrderNotFound         = errors.New("order not found")
	ErrOrderItemNotFound     = errors.New("order item not found")
	ErrInsufficientStock     = errors.New("insufficient stock for the requested quantity")
	ErrInvalidOrderStatus    = errors.New("invalid order status")
	ErrOrderCannotBeModified = errors.New("order cannot be modified in current status")
	ErrEmptyOrder            = errors.New("order must contain at least one item")
)

// Order status constants
const (
	OrderStatusPlaced     = "PLACED"
	OrderStatusProcessing = "PROCESSING"
	OrderStatusShipped    = "SHIPPED"
	OrderStatusDelivered  = "DELIVERED"
	OrderStatusCancelled  = "CANCELLED"
)

// Define the OrderModel type
type OrderModel struct {
	DB *database.Queries
}

// Order represents an order in the system
type Order struct {
	ID        int32           `json:"id"`
	UserID    int32           `json:"user_id"`
	TotalKES  decimal.Decimal `json:"total_kes"`
	Status    string          `json:"status"`
	Version   int32           `json:"version"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Items     []*OrderItem    `json:"items,omitempty"`
	User      *UserInfo       `json:"user,omitempty"`
}

// OrderItem represents an item within an order
type OrderItem struct {
	ID           int32           `json:"id"`
	OrderID      int32           `json:"order_id"`
	ProductID    int32           `json:"product_id"`
	ProductName  string          `json:"product_name,omitempty"`
	Quantity     int32           `json:"quantity"`
	UnitPriceKES decimal.Decimal `json:"unit_price_kes"`
	CreatedAt    time.Time       `json:"created_at"`
}

// UserInfo represents user information for orders
type UserInfo struct {
	ID        int32  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// OrderStatistics represents order statistics for admin dashboard
type OrderStatistics struct {
	TotalOrders       int64           `json:"total_orders"`
	PlacedOrders      int64           `json:"placed_orders"`
	ProcessingOrders  int64           `json:"processing_orders"`
	ShippedOrders     int64           `json:"shipped_orders"`
	DeliveredOrders   int64           `json:"delivered_orders"`
	CancelledOrders   int64           `json:"cancelled_orders"`
	TotalRevenue      decimal.Decimal `json:"total_revenue"`
	AverageOrderValue decimal.Decimal `json:"average_order_value"`
}

// CreateOrderRequest represents the data needed to create a new order
type CreateOrderRequest struct {
	UserID int32                     `json:"user_id"`
	Items  []*CreateOrderItemRequest `json:"items"`
}

// CreateOrderItemRequest represents an item to be added to an order
type CreateOrderItemRequest struct {
	ProductID int32 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}

// ProductAvailability represents product availability check result
type ProductAvailability struct {
	ID            int32           `json:"id"`
	Name          string          `json:"name"`
	StockQuantity int32           `json:"stock_quantity"`
	IsAvailable   bool            `json:"is_available"`
	CurrentPrice  decimal.Decimal `json:"current_price"`
}

// Timeout constants for our module
const (
	DefaultOrderDBContextTimeout = 10 * time.Second
)

// populateOrder converts a database Order row into an Order struct.
func populateOrder(orderRow any) *Order {
	switch order := orderRow.(type) {
	case database.Order:
		orderStruct := &Order{
			ID:        order.ID,
			UserID:    order.UserID,
			Status:    order.Status,
			Version:   order.Version,
			CreatedAt: order.CreatedAt,
			UpdatedAt: order.UpdatedAt,
		}
		orderStruct.TotalKES, _ = decimal.NewFromString(order.TotalKes)
		return orderStruct
	default:
		return nil // Return nil if the type does not match
	}
}

// populateOrderItem converts a database OrderItem row into an OrderItem struct.
func populateOrderItem(orderItemRow any) *OrderItem {
	switch item := orderItemRow.(type) {
	case database.OrderItem:
		orderItem := &OrderItem{
			ID:        item.ID,
			OrderID:   item.OrderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
		}
		orderItem.UnitPriceKES, _ = decimal.NewFromString(item.UnitPriceKes)
		return orderItem
	default:
		return nil // Return nil if the type does not match
	}
}

// populateOrderStatistics converts a database statistics row into an OrderStatistics struct.
func populateOrderStatistics(statsRow any) *OrderStatistics {
	switch stats := statsRow.(type) {
	case database.GetOrderStatisticsRow:
		orderStats := &OrderStatistics{
			TotalOrders:      stats.TotalOrders,
			PlacedOrders:     stats.PlacedOrders,
			ProcessingOrders: stats.ProcessingOrders,
			ShippedOrders:    stats.ShippedOrders,
			DeliveredOrders:  stats.DeliveredOrders,
			CancelledOrders:  stats.CancelledOrders,
		}
		orderStats.TotalRevenue, _ = decimal.NewFromString(stats.TotalRevenue)
		orderStats.AverageOrderValue, _ = decimal.NewFromString(stats.AverageOrderValue)
		return orderStats
	default:
		return nil // Return nil if the type does not match
	}
}

// Validation functions
func ValidateCreateOrderRequest(v *validator.Validator, req *CreateOrderRequest) {
	v.Check(req.UserID > 0, "user_id", "must be a valid user ID")
	v.Check(len(req.Items) > 0, "items", "must contain at least one item")

	for i, item := range req.Items {
		v.Check(item.ProductID > 0, fmt.Sprintf("items[%d].product_id", i), "must be a valid product ID")
		v.Check(item.Quantity > 0, fmt.Sprintf("items[%d].quantity", i), "must be greater than 0")
	}
}

func ValidateOrderStatus(v *validator.Validator, status string) {
	validStatuses := []string{
		OrderStatusPlaced,
		OrderStatusProcessing,
		OrderStatusShipped,
		OrderStatusDelivered,
		OrderStatusCancelled,
	}

	v.Check(status != "", "status", "must be provided")
	v.Check(validator.PermittedValue(status, validStatuses...), "status", "must be a valid status")
}

// isValidStatusTransition checks if the status transition is valid
func isValidStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		OrderStatusPlaced:     {OrderStatusProcessing, OrderStatusCancelled},
		OrderStatusProcessing: {OrderStatusShipped, OrderStatusCancelled},
		OrderStatusShipped:    {OrderStatusDelivered},
		OrderStatusDelivered:  {}, // No further transitions allowed
		OrderStatusCancelled:  {}, // No further transitions allowed
	}

	allowedTransitions, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// CheckProductAvailability checks if a product has sufficient stock
func (m OrderModel) CheckProductAvailability(productID int32, requiredQuantity int32) (*ProductAvailability, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	result, err := m.DB.CheckProductAvailability(ctx, database.CheckProductAvailabilityParams{
		ID:            productID,
		StockQuantity: requiredQuantity,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrGeneralRecordNotFound
		default:
			return nil, err
		}
	} // Get current price from products table
	product, err := m.DB.GetProductByIdOnly(ctx, productID)
	if err != nil {
		return nil, err
	}

	currentPrice, _ := decimal.NewFromString(product.PriceKes)

	availability := &ProductAvailability{
		ID:            result.ID,
		Name:          result.Name,
		StockQuantity: result.StockQuantity,
		IsAvailable:   result.IsAvailable,
		CurrentPrice:  currentPrice,
	}

	return availability, nil
}

// CreateOrder creates a new order with the provided items
func (m OrderModel) CreateOrder(req *CreateOrderRequest) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	// Check product availability and calculate total in one pass
	var totalAmount decimal.Decimal
	productAvailabilityMap := make(map[int32]*ProductAvailability)

	for _, item := range req.Items {
		availability, err := m.CheckProductAvailability(item.ProductID, item.Quantity)
		if err != nil {
			return nil, err
		}

		if !availability.IsAvailable {
			return nil, fmt.Errorf("product %s: %w", availability.Name, ErrInsufficientStock)
		}

		// Store availability data for later use
		productAvailabilityMap[item.ProductID] = availability

		// Calculate item total
		itemTotal := availability.CurrentPrice.Mul(decimal.NewFromInt32(item.Quantity))
		totalAmount = totalAmount.Add(itemTotal)
	}

	// Create the order
	dbOrder, err := m.DB.CreateOrder(ctx, database.CreateOrderParams{
		UserID:   req.UserID,
		TotalKes: totalAmount.String(),
		Status:   OrderStatusPlaced,
	})
	if err != nil {
		return nil, err
	}

	// Create order items using cached availability data
	var items []*OrderItem
	for _, item := range req.Items {
		// Get cached availability data
		availability := productAvailabilityMap[item.ProductID]

		dbOrderItem, err := m.DB.CreateOrderItem(ctx, database.CreateOrderItemParams{
			OrderID:      dbOrder.ID,
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			UnitPriceKes: availability.CurrentPrice.String(),
		})
		if err != nil {
			return nil, err
		}

		orderItem := populateOrderItem(dbOrderItem)
		if orderItem != nil {
			orderItem.ProductName = availability.Name
			items = append(items, orderItem)
		}

		// Update product stock using cached data
		err = m.DB.UpdateProductStockQuantity(ctx, database.UpdateProductStockQuantityParams{
			ID:            item.ProductID,
			StockQuantity: availability.StockQuantity - item.Quantity,
		})
		if err != nil {
			return nil, err
		}
	}

	// Populate and return the order
	order := populateOrder(dbOrder)
	if order != nil {
		order.Items = items
	}

	return order, nil
}

// GetOrderByID retrieves an order by its ID
func (m OrderModel) GetOrderByID(orderID int32) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	dbOrder, err := m.DB.GetOrderById(ctx, orderID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrOrderNotFound
		default:
			return nil, err
		}
	}
	order := &Order{}
	order = populateOrder(dbOrder)

	return order, nil
}

// GetOrderWithItems retrieves an order with its items by order ID
func (m OrderModel) GetOrderWithItems(orderID int32) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	rows, err := m.DB.GetOrderByIdWithItems(ctx, orderID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrOrderNotFound
		default:
			return nil, err
		}
	}

	if len(rows) == 0 {
		return nil, ErrOrderNotFound
	}

	// Build the order from the first row
	firstRow := rows[0]
	order := &Order{
		ID:        firstRow.OrderID,
		UserID:    firstRow.UserID,
		Status:    firstRow.Status,
		Version:   firstRow.Version,
		CreatedAt: firstRow.OrderCreatedAt,
		UpdatedAt: firstRow.OrderUpdatedAt,
	}
	order.TotalKES, _ = decimal.NewFromString(firstRow.TotalKes)

	// Build order items
	var items []*OrderItem
	for _, row := range rows {
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			items = append(items, item)
		}
	}

	order.Items = items
	return order, nil
}

// GetAllOrdersWithItems retrieves all orders with their items (admin view)
func (m OrderModel) GetAllOrdersWithItems(name string, filters Filters) ([]*Order, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	rows, err := m.DB.GetAllOrdersWithItems(ctx, database.GetAllOrdersWithItemsParams{
		Column1: name,
		Limit:   int32(filters.limit()),
		Offset:  int32(filters.offset()),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrGeneralRecordNotFound
		default:
			return nil, Metadata{}, err
		}
	}

	if len(rows) == 0 {
		return []*Order{}, Metadata{}, nil
	}

	// Group rows by order ID
	orderMap := make(map[int32]*Order)
	var totalRecords int64

	for _, row := range rows {
		totalRecords = row.TotalCount

		// Check if order already exists in map
		order, exists := orderMap[row.OrderID]
		if !exists {
			// Create new order
			order = &Order{
				ID:        row.OrderID,
				UserID:    row.UserID,
				Status:    row.Status,
				Version:   row.Version,
				CreatedAt: row.OrderCreatedAt,
				UpdatedAt: row.OrderUpdatedAt,
				User: &UserInfo{
					ID:        row.UserID,
					FirstName: row.UserFirstName,
					LastName:  row.UserLastName,
					Email:     row.UserEmail,
				},
			}
			order.TotalKES, _ = decimal.NewFromString(row.TotalKes)
			order.Items = []*OrderItem{}
			orderMap[row.OrderID] = order
		}

		// Add order item if it exists
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			order.Items = append(order.Items, item)
		}
	}

	// Convert map to slice
	orders := make([]*Order, 0, len(orderMap))
	for _, order := range orderMap {
		orders = append(orders, order)
	}

	// Calculate metadata
	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)

	return orders, metadata, nil
}

// GetUserOrdersWithItems retrieves orders for a specific user
func (m OrderModel) GetUserOrdersWithItems(userID int32, filters Filters) ([]*Order, Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	rows, err := m.DB.GetUserOrdersWithItems(ctx, database.GetUserOrdersWithItemsParams{
		UserID: userID,
		Limit:  int32(filters.limit()),
		Offset: int32(filters.offset()),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrGeneralRecordNotFound
		default:
			return nil, Metadata{}, err
		}
	}

	if len(rows) == 0 {
		return []*Order{}, Metadata{}, nil
	}

	// Group rows by order ID
	orderMap := make(map[int32]*Order)
	var totalRecords int64

	for _, row := range rows {
		totalRecords = row.TotalCount

		// Check if order already exists in map
		order, exists := orderMap[row.OrderID]
		if !exists {
			// Create new order
			order = &Order{
				ID:        row.OrderID,
				UserID:    row.UserID,
				Status:    row.Status,
				Version:   row.Version,
				CreatedAt: row.OrderCreatedAt,
				UpdatedAt: row.OrderUpdatedAt,
			}
			order.TotalKES, _ = decimal.NewFromString(row.TotalKes)
			order.Items = []*OrderItem{}
			orderMap[row.OrderID] = order
		}

		// Add order item if it exists
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			order.Items = append(order.Items, item)
		}
	}

	// Convert map to slice
	orders := make([]*Order, 0, len(orderMap))
	for _, order := range orderMap {
		orders = append(orders, order)
	}

	// Calculate metadata
	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)

	return orders, metadata, nil
}

// populateOrders converts a database row into an Order struct with items.
func populateOrders(orderRow any) *Order {
	switch row := orderRow.(type) {
	case database.GetAllOrdersWithItemsRow:
		order := &Order{
			ID:        row.OrderID,
			UserID:    row.UserID,
			Status:    row.Status,
			Version:   row.Version,
			CreatedAt: row.OrderCreatedAt,
			UpdatedAt: row.OrderUpdatedAt,
			User: &UserInfo{
				ID:        row.UserID,
				FirstName: row.UserFirstName,
				LastName:  row.UserLastName,
				Email:     row.UserEmail,
			},
		}
		order.TotalKES, _ = decimal.NewFromString(row.TotalKes)

		// Add order item if it exists
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			order.Items = []*OrderItem{item}
		} else {
			order.Items = []*OrderItem{}
		}

		return order

	case database.GetUserOrdersWithItemsRow:
		order := &Order{
			ID:        row.OrderID,
			UserID:    row.UserID,
			Status:    row.Status,
			Version:   row.Version,
			CreatedAt: row.OrderCreatedAt,
			UpdatedAt: row.OrderUpdatedAt,
		}
		order.TotalKES, _ = decimal.NewFromString(row.TotalKes)

		// Add order item if it exists
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			order.Items = []*OrderItem{item}
		} else {
			order.Items = []*OrderItem{}
		}

		return order

	case database.GetOrderByIdWithItemsRow:
		order := &Order{
			ID:        row.OrderID,
			UserID:    row.UserID,
			Status:    row.Status,
			Version:   row.Version,
			CreatedAt: row.OrderCreatedAt,
			UpdatedAt: row.OrderUpdatedAt,
		}
		order.TotalKES, _ = decimal.NewFromString(row.TotalKes)

		// Add order item if it exists
		if row.OrderItemID.Valid {
			item := &OrderItem{
				ID:          row.OrderItemID.Int32,
				OrderID:     row.OrderID,
				ProductID:   row.ProductID.Int32,
				ProductName: row.ProductName.String,
				Quantity:    row.Quantity.Int32,
				CreatedAt:   row.ItemCreatedAt.Time,
			}
			item.UnitPriceKES, _ = decimal.NewFromString(row.UnitPriceKes.String)
			order.Items = []*OrderItem{item}
		} else {
			order.Items = []*OrderItem{}
		}

		return order

	default:
		return nil // Return nil if the type does not match
	}
}

// GetOrderStatistics retrieves order statistics for a date range
func (m OrderModel) GetOrderStatistics(startDate, endDate time.Time) (*OrderStatistics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	dbStats, err := m.DB.GetOrderStatistics(ctx, database.GetOrderStatisticsParams{
		CreatedAt:   startDate,
		CreatedAt_2: endDate,
	})
	if err != nil {
		return nil, err
	}
	stats := populateOrderStatistics(dbStats)

	return stats, nil
}

// UpdateOrderStatus updates the status of an order
func (m OrderModel) UpdateOrderStatus(orderID int32, newStatus string, expectedVersion int32) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultOrderDBContextTimeout)
	defer cancel()

	// Get current order to validate status transition
	currentOrder, err := m.DB.GetOrderById(ctx, orderID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrOrderNotFound
		default:
			return nil, err
		}
	}

	// Check version for optimistic locking
	if currentOrder.Version != expectedVersion {
		return nil, ErrEditConflict
	}

	// Validate status transition
	if !isValidStatusTransition(currentOrder.Status, newStatus) {
		return nil, ErrInvalidOrderStatus
	}

	// Update the order status
	updatedOrder, err := m.DB.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID:      orderID,
		Status:  newStatus,
		Version: expectedVersion,
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}

	// Convert to service order
	order := populateOrder(updatedOrder)
	return order, nil
}

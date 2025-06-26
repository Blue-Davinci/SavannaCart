package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Blue-Davinci/SavannaCart/internal/data"
	"github.com/Blue-Davinci/SavannaCart/internal/validator"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// getAllOrdersHandler() handles requests to get all orders (admin only)
// It supports pagination, sorting, and filtering by customer name.
// It retrieves all orders with their items and returns them as a JSON response.
func (app *application) getAllOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Name = app.readString(qs, "name", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "")
	input.Filters.SortSafelist = []string{"", "created_at", "-created_at", "total_kes", "-total_kes"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	orders, metadata, err := app.models.Orders.GetAllOrdersWithItems(input.Name, input.Filters)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"orders": orders, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getUserOrdersHandler() handles requests to get orders for the authenticated user
// It supports pagination and sorting, and retrieves all orders with their items.
// It returns the orders as a JSON response.
func (app *application) getUserOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"", "created_at", "-created_at", "total_kes", "-total_kes"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get user from context
	user := app.contextGetUser(r)
	if user == nil {
		app.authenticationRequiredResponse(w, r)
		return
	}

	orders, metadata, err := app.models.Orders.GetUserOrdersWithItems(int32(app.contextGetUser(r).ID), input.Filters)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"orders": orders, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getOrderStatisticsHandler() handles requests to get order statistics (admin only)
// It retrieves statistics for orders within a specified date range.
// The date range can be specified using query parameters "start_date" and "end_date".
func (app *application) getOrderStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	// Get date range parameters
	startDateStr := app.readString(qs, "start_date", "")
	endDateStr := app.readString(qs, "end_date", "")

	v := validator.New()

	var startDate, endDate time.Time
	var err error

	// Parse start date
	if startDateStr == "" {
		// Default to 30 days ago
		startDate = time.Now().AddDate(0, 0, -30)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			v.AddError("start_date", "must be a valid date in YYYY-MM-DD format")
		}
	}

	// Parse end date
	if endDateStr == "" {
		// Default to now
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			v.AddError("end_date", "must be a valid date in YYYY-MM-DD format")
		}
	}

	// Validate date range
	if startDate.After(endDate) {
		v.AddError("start_date", "must be before end date")
	}

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get statistics
	stats, err := app.models.Orders.GetOrderStatistics(startDate, endDate)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Add date range to response
	response := envelope{
		"statistics": stats,
		"date_range": envelope{
			"start_date": startDate.Format("2006-01-02"),
			"end_date":   endDate.Format("2006-01-02"),
		},
	}

	err = app.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// createOrderHandler() handles requests to create a new order
// It expects a JSON body with an array of items, each containing a product ID and quantity.
// It validates the request, creates the order, and sends a confirmation email to the user.
func (app *application) createOrderHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Items []struct {
			ProductID int32 `json:"product_id"`
			Quantity  int32 `json:"quantity"`
		} `json:"items"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get user ID from context (set by authentication middleware)
	user := app.contextGetUser(r)
	if user == nil {
		app.authenticationRequiredResponse(w, r)
		return
	}

	// Prepare create request
	var orderItems []*data.CreateOrderItemRequest
	for _, item := range input.Items {
		orderItems = append(orderItems, &data.CreateOrderItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	createReq := &data.CreateOrderRequest{
		UserID: int32(app.contextGetUser(r).ID), // Use the user ID from the context
		Items:  orderItems,
	}

	// Validate the request
	v := validator.New()
	data.ValidateCreateOrderRequest(v, createReq)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Create the order
	order, err := app.models.Orders.CreateOrder(createReq)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInsufficientStock):
			v.AddError("items", err.Error())
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrGeneralRecordNotFound):
			v.AddError("items", "one or more products not found")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrEmptyOrder):
			v.AddError("items", "order must contain at least one item")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Send order confirmation email in background
	app.background(func() {
		app.sendOrderConfirmationEmail(order.ID)
	})
	// Send admin order notification email in background
	app.background(func() {
		app.sendAdminOrderNotification(order.ID)
	})

	// Send SMS notification to user in background
	app.background(func() {
		app.sendOrderConfirmationSMS(order.ID)
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"order": order}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// updateOrderStatusHandler handles requests to update order status (admin only)
func (app *application) updateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Get order ID from URL
	id, err := app.readIDParam(r, "orderID")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	orderID := int32(id)

	var input struct {
		Status  string `json:"status"`
		Version int32  `json:"version"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate status
	v := validator.New()
	data.ValidateOrderStatus(v, input.Status)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Update order status
	order, err := app.models.Orders.UpdateOrderStatus(orderID, input.Status, input.Version)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrOrderNotFound):
			app.notFoundResponse(w, r)
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		case errors.Is(err, data.ErrInvalidOrderStatus):
			v.AddError("status", err.Error())
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Send order status update email in background
	app.background(func() {
		app.sendOrderStatusUpdateEmail(orderID, input.Status)
	})

	err = app.writeJSON(w, http.StatusOK, envelope{"order": order}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// sendOrderStatusUpdateEmail sends an email notification to the user when order status changes
func (app *application) sendOrderStatusUpdateEmail(orderID int32, newStatus string) {
	// Get full order details with items
	fullOrder, err := app.models.Orders.GetOrderWithItems(orderID)
	if err != nil {
		app.logger.Error("Failed to get order details for email notification",
			zap.Int32("order_id", orderID),
			zap.Error(err))
		return
	}

	// Get user details using the UserID from the order
	user, err := app.models.Users.GetUserByID(int64(fullOrder.UserID))
	if err != nil {
		app.logger.Error("Failed to get user details for email notification",
			zap.Int32("order_id", orderID),
			zap.Int32("user_id", fullOrder.UserID),
			zap.Error(err))
		return
	}

	// Prepare order items for email template
	var emailItems []map[string]any
	for _, item := range fullOrder.Items {
		emailItems = append(emailItems, map[string]any{
			"productName": item.ProductName,
			"quantity":    item.Quantity,
			"unitPrice":   item.UnitPriceKES.StringFixed(2),
		})
	}
	// Create email data map
	data := map[string]any{
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"orderID":     fullOrder.ID,
		"status":      newStatus,
		"statusLower": strings.ToLower(newStatus), // Add lowercase version for CSS classes
		"totalAmount": fullOrder.TotalKES.StringFixed(2),
		"orderDate":   fullOrder.CreatedAt.Format("January 2, 2006"),
		"items":       emailItems,
	}

	// Add tracking URL if order is shipped (you can modify this based on your tracking system)
	if newStatus == "SHIPPED" {
		data["trackingURL"] = fmt.Sprintf("https://track.savannacart.com/order/%d", fullOrder.ID)
	}

	// Send the order status update email
	err = app.mailer.Send(user.Email, "order_status_update.tmpl", data)
	if err != nil {
		app.logger.Error("Error sending order status update email",
			zap.String("email", user.Email),
			zap.Int32("order_id", fullOrder.ID),
			zap.String("status", newStatus),
			zap.Error(err))
		return
	}

	app.logger.Info("Order status update email sent successfully",
		zap.String("email", user.Email),
		zap.Int32("order_id", fullOrder.ID),
		zap.String("status", newStatus))
}

// sendOrderConfirmationEmail sends a confirmation email to the user when a new order is created
func (app *application) sendOrderConfirmationEmail(orderID int32) {
	// Get full order details with items
	fullOrder, err := app.models.Orders.GetOrderWithItems(orderID)
	if err != nil {
		app.logger.Error("Failed to get order details for confirmation email",
			zap.Int32("order_id", orderID),
			zap.Error(err))
		return
	}

	// Get user details using the UserID from the order
	user, err := app.models.Users.GetUserByID(int64(fullOrder.UserID))
	if err != nil {
		app.logger.Error("Failed to get user details for confirmation email",
			zap.Int32("order_id", orderID),
			zap.Int32("user_id", fullOrder.UserID),
			zap.Error(err))
		return
	}

	// Prepare order items for email template
	var emailItems []map[string]any
	for _, item := range fullOrder.Items {
		emailItems = append(emailItems, map[string]any{
			"productName": item.ProductName,
			"quantity":    item.Quantity,
			"unitPrice":   item.UnitPriceKES.StringFixed(2),
		})
	}
	// Create email data map for order confirmation
	data := map[string]any{
		"firstName":   user.FirstName,
		"lastName":    user.LastName,
		"orderID":     fullOrder.ID,
		"status":      "PENDING", // New orders start as pending
		"statusLower": "pending", // Add lowercase version for CSS classes
		"totalAmount": fullOrder.TotalKES.StringFixed(2),
		"orderDate":   fullOrder.CreatedAt.Format("January 2, 2006"),
		"items":       emailItems,
	}

	// Send the order confirmation email (reusing the order_status_update template)
	err = app.mailer.Send(user.Email, "order_status_update.tmpl", data)
	if err != nil {
		app.logger.Error("Error sending order confirmation email",
			zap.String("email", user.Email),
			zap.Int32("order_id", fullOrder.ID),
			zap.Error(err))
		return
	}
	app.logger.Info("Order confirmation email sent successfully",
		zap.String("email", user.Email),
		zap.Int32("order_id", fullOrder.ID))
}

// sendAdminOrderNotification sends email notifications to all super users about new orders
func (app *application) sendAdminOrderNotification(orderID int32) {
	// Get full order details with items
	fullOrder, err := app.models.Orders.GetOrderWithItems(orderID)
	if err != nil {
		app.logger.Error("Failed to get order details for admin notification",
			zap.Int32("order_id", orderID),
			zap.Error(err))
		return
	}

	// Get customer details using the UserID from the order
	customer, err := app.models.Users.GetUserByID(int64(fullOrder.UserID))
	if err != nil {
		app.logger.Error("Failed to get customer details for admin notification",
			zap.Int32("order_id", orderID),
			zap.Int32("user_id", fullOrder.UserID),
			zap.Error(err))
		return
	}

	// Get all super users with permissions
	superUsers, err := app.models.Permissions.GetAllSuperUsersWithPermissions()
	if err != nil {
		app.logger.Error("Failed to get super users for admin notification",
			zap.Int32("order_id", orderID),
			zap.Error(err))
		return
	}

	// If no super users found, log and return
	if len(superUsers) == 0 {
		app.logger.Warn("No super users found for admin notification",
			zap.Int32("order_id", orderID))
		return
	}

	// Prepare order items for email template
	var emailItems []map[string]any
	for _, item := range fullOrder.Items {
		// Calculate total price for this item (quantity * unit price)
		quantityDecimal := decimal.NewFromInt32(item.Quantity)
		totalPrice := item.UnitPriceKES.Mul(quantityDecimal)

		emailItems = append(emailItems, map[string]any{
			"productName": item.ProductName,
			"quantity":    item.Quantity,
			"unitPrice":   item.UnitPriceKES.StringFixed(2),
			"totalPrice":  totalPrice.StringFixed(2),
		})
	}

	// Create email data map for admin notification
	data := map[string]any{
		"orderID":           fullOrder.ID,
		"totalAmount":       fullOrder.TotalKES.StringFixed(2),
		"orderDate":         fullOrder.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
		"customerFirstName": customer.FirstName,
		"customerLastName":  customer.LastName,
		"customerEmail":     customer.Email,
		"items":             emailItems,
		"dashboardURL":      "https://admin.savannacart.com", // You can make this configurable
		"currentYear":       fullOrder.CreatedAt.Year(),
	}

	// Handle phone number (it's a string, not sql.NullString)
	if customer.PhoneNumber != "" {
		data["customerPhone"] = customer.PhoneNumber
	} else {
		data["customerPhone"] = nil
	}

	// Create a map to track unique emails (to avoid sending duplicate emails to same admin)
	uniqueEmails := make(map[string]bool)
	var adminEmails []string

	// Collect unique admin emails
	for _, superUser := range superUsers {
		if !uniqueEmails[superUser.UserEmail] {
			uniqueEmails[superUser.UserEmail] = true
			adminEmails = append(adminEmails, superUser.UserEmail)
		}
	}

	// Send email to each unique admin
	successCount := 0
	failureCount := 0

	for _, adminEmail := range adminEmails {
		err = app.mailer.Send(adminEmail, "admin_order_notification.tmpl", data)
		if err != nil {
			app.logger.Error("Error sending admin order notification email",
				zap.String("admin_email", adminEmail),
				zap.Int32("order_id", fullOrder.ID),
				zap.Error(err))
			failureCount++
		} else {
			app.logger.Info("Admin order notification email sent successfully",
				zap.String("admin_email", adminEmail),
				zap.Int32("order_id", fullOrder.ID))
			successCount++
		}
	}
	// Log summary
	app.logger.Info("Admin order notification summary",
		zap.Int32("order_id", fullOrder.ID),
		zap.Int("total_admins", len(adminEmails)),
		zap.Int("success_count", successCount),
		zap.Int("failure_count", failureCount))
}

// sendOrderConfirmationSMS sends a simple confirmation SMS to the user when a new order is created
func (app *application) sendOrderConfirmationSMS(orderID int32) {
	// Get full order details with items
	fullOrder, err := app.models.Orders.GetOrderWithItems(orderID)
	if err != nil {
		app.logger.Error("Failed to get order details for SMS confirmation",
			zap.Int32("order_id", orderID),
			zap.Error(err))
		return
	}

	// Get user details using the UserID from the order
	user, err := app.models.Users.GetUserByID(int64(fullOrder.UserID))
	if err != nil {
		app.logger.Error("Failed to get user details for SMS confirmation",
			zap.Int32("order_id", orderID),
			zap.Int32("user_id", fullOrder.UserID),
			zap.Error(err))
		return
	}

	// Check if user has a phone number
	if user.PhoneNumber == "" {
		app.logger.Info("User has no phone number, skipping SMS confirmation",
			zap.Int32("order_id", orderID),
			zap.Int32("user_id", fullOrder.UserID))
		return
	}

	// Check if SMS service is enabled
	if !app.sms.IsEnabled() {
		app.logger.Info("SMS service is disabled, skipping SMS confirmation",
			zap.Int32("order_id", orderID))
		return
	}

	// Send SMS confirmation
	err = app.sms.SendOrderConfirmation(user.PhoneNumber, fullOrder.ID, fullOrder.TotalKES.StringFixed(2))
	if err != nil {
		// Check if it's a trial account limitation and log appropriately
		if strings.Contains(err.Error(), "Trial accounts") || strings.Contains(err.Error(), "restricted") {
			app.logger.Info("SMS sending restricted due to trial account limitations",
				zap.String("phone_number", user.PhoneNumber),
				zap.Int32("order_id", fullOrder.ID),
				zap.String("solution", "Verify phone number in Twilio console or upgrade account"))
		} else {
			app.logger.Error("Error sending order confirmation SMS",
				zap.String("phone_number", user.PhoneNumber),
				zap.Int32("order_id", fullOrder.ID),
				zap.Error(err))
		}
		return
	}

	app.logger.Info("Order confirmation SMS sent successfully",
		zap.String("phone_number", user.PhoneNumber),
		zap.Int32("order_id", fullOrder.ID))
}

package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"
	"log"

	"resto-backend/internal/models"
	"resto-backend/internal/fcm"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// POST /orders
type orderItemInput struct {
	MenuItemID int64 `json:"menu_item_id"`
	Quantity   int   `json:"quantity"`
}

type createOrderRequest struct {
	Items []orderItemInput `json:"items"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	userID := c.GetInt64("userID")

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Order must contain at least one item"})
		return
	}

	for _, item := range req.Items {
		if item.MenuItemID <= 0 || item.Quantity <= 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "menu_item_id and quantity must be positive integers"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Implement transaction to create order and order items
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	var orderID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, status, total, created_at, updated_at)
		 VALUES ($1, 'pending', 0, NOW(), NOW())
		 RETURNING id`, userID).Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	total := 0
	for _, item := range req.Items {
		var price int
		err := tx.QueryRow(ctx,
			`SELECT price FROM menu_items WHERE id = $1 AND is_available = TRUE`,
			item.MenuItemID).Scan(&price)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Menu item not found or unavailable", "menu_item_id": item.MenuItemID})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu item price"})
			}
			return
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (order_id, menu_item_id, quantity, price_at_order)
			 VALUES ($1, $2, $3, $4)`,
			orderID, item.MenuItemID, item.Quantity, price)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order item"})
			return
		}

		total += price * item.Quantity
	}

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET total = $1, updated_at = NOW() WHERE id = $2`,
		total, orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order total"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": orderID, "status": "pending", "total": total})
}

// GET /orders/:id
func (h *Handler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var order models.Order
	err = h.DB.QueryRow(ctx,
		`SELECT id, user_id, status, total, created_at, updated_at 
		 FROM orders 
		 WHERE id = $1`, id).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		}
		return
	}

	// check if the user is the owner of the order
	requestingUserID := c.GetInt64("userID")
	if order.UserID != requestingUserID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	rows, err := h.DB.Query(ctx,
		`SELECT oi.id, oi.menu_item_id, m.name, oi.quantity, oi.price_at_order 
		 FROM order_items oi
		 JOIN menu_items m ON oi.menu_item_id = m.id
		 WHERE oi.order_id = $1
		 ORDER BY oi.id`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order items"})
		return
	}
	defer rows.Close()

	order.Items = []models.OrderItem{}
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ID, &item.MenuItemID, &item.Name, &item.Quantity, &item.PriceAtOrder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan order item"})
			return
		}
		order.Items = append(order.Items, item)
	}
	
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while iterating over order items"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// PATCH /orders/:id/status
type updateStatusRequest struct {
	Status string `json:"status"`
}

var validStatuses = map[string]bool{
	"pending":   true,
	"preparing": true,
	"ready":     true,
	"completed": true,
	"cancelled": true,
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var ownerID int64
	err = h.DB.QueryRow(ctx,
		`UPDATE orders SET status = $1, updated_at = NOW()
		 WHERE id = $2
		 RETURNING user_id`, req.Status, id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order owner"})
		return
	}

	// TODO: implement FCM notification to the user when req.Status = "ready"
	if req.Status == "ready" && h.FCM != nil {
		var deviceToken *string
		err = h.DB.QueryRow(ctx,
			`SELECT device_token FROM users WHERE id = $1`, ownerID).Scan(&deviceToken)
		if err == nil && deviceToken != nil && *deviceToken != "" {
			if err := fcm.SendOrderReady(ctx, h.FCM, *deviceToken, id); err != nil {
				log.Printf("Failed to send FCM notification for order %d: %v", id, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status})
}
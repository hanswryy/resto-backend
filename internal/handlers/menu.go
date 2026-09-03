package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"resto-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// GET /menu
func (h *Handler) ListMenu(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.DB.Query(ctx,
		`SELECT id, name, description, price, is_available 
		 FROM menu_items
		 ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu items"})
		return
	}
	defer rows.Close()

	items := []models.MenuItem{}
	for rows.Next() {
		var item models.MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.IsAvailable); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan menu item"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while iterating over menu items"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GET /menu/:id
func (h *Handler) GetMenuItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var m models.MenuItem
	err = h.DB.QueryRow(ctx,
		`SELECT id, name, description, price, is_available 
		 FROM menu_items 
		 WHERE id = $1`, id).Scan(&m.ID, &m.Name, &m.Description, &m.Price, &m.IsAvailable)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu item"})
		}
		return
	}

	c.JSON(http.StatusOK, m)
}
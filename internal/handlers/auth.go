package handlers

import (
	"context"
	"net/http"
	"time"

	"resto-backend/internal/auth"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email		string `json:"email" binding:"required,email"`
	Password	string `json:"password" binding:"required"`
	DeviceToken	string `json:"device_token"` 
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var userID int64
	var passwordHash string
	err := h.DB.QueryRow(ctx,
		`SELECT id, password_hash FROM users WHERE email = $1`,
		req.Email,
	).Scan(&userID, &passwordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := auth.GenerateToken(userID, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	if req.DeviceToken != "" {
		_, _ = h.DB.Exec(ctx,
			`UPDATE users SET device_token = $1 WHERE id = $2`,
			req.DeviceToken, userID,
		)
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
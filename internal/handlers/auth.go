// backend/internal/handlers/auth.go
package handlers

import (
	"net/http"
	// "time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/hotel-bed-management/backend/internal/auth"
	"github.com/yourusername/hotel-bed-management/backend/internal/db"
	"github.com/yourusername/hotel-bed-management/backend/internal/models"
)

type AuthHandler struct {
	db     *db.Database
	secret string
}

func NewAuthHandler(db *db.Database, secret string) *AuthHandler {
	return &AuthHandler{db: db, secret: secret}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	err := h.db.Pool.QueryRow(c, 
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1", 
		req.Username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.Role, h.secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}
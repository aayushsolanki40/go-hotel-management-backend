// backend/cmd/server/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	// "github.com/yourusername/hotel-bed-management/backend/internal/auth"
	"github.com/yourusername/hotel-bed-management/backend/internal/db"
	"github.com/yourusername/hotel-bed-management/backend/internal/handlers"
	"github.com/yourusername/hotel-bed-management/backend/internal/middleware"
)


func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	database, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize default admin user if not exists
	initAdminUser(database)

	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-key"
	}

	authHandler := handlers.NewAuthHandler(database, secret)
	hotelHandler := handlers.NewHotelHandler(database)

	// Public routes
	router.POST("/api/auth/login", authHandler.Login)

	// Protected routes
	authMiddleware := middleware.AuthMiddleware(secret)
	api := router.Group("/api")
	api.Use(authMiddleware)
	{
		api.GET("/hotels", hotelHandler.GetHotels)
		api.GET("/hotels/:hotelId/beds", hotelHandler.GetBeds)
		api.POST("/customers", hotelHandler.CreateCustomer)
		api.POST("/customers/checkout", hotelHandler.CheckoutCustomer)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(router.Run(":" + port))
}

func initAdminUser(db *db.Database) {
	var count int
	err := db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		log.Printf("Failed to check users: %v", err)
		return
	}

	if count == 0 {
		// Create default admin user
		username := "admin"
		password := "admin123" // In production, this should be set via environment variable
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			return
		}

		_, err = db.Pool.Exec(context.Background(), 
			"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
			username, string(hashedPassword), "admin",
		)
		if err != nil {
			log.Printf("Failed to create admin user: %v", err)
			return
		}

		log.Println("Default admin user created")
	}
}
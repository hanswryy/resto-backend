package main

import (
  "log"
  "os"

	"resto-backend/internal/db"
  "resto-backend/internal/handlers"

	"github.com/gin-gonic/gin"
  "github.com/joho/godotenv"
)

func main() {
  if err := godotenv.Load(); err != nil {
    log.Println("No .env file found")
  }

  databaseURL := os.Getenv("DATABASE_URL")
  if databaseURL == "" {
    log.Fatal("DATABASE_URL environment variable is not set")
  }

  pool, err := db.Connect(databaseURL)
  if err != nil {
    log.Fatalf("Failed to connect to the database: %v", err)
  }
  defer pool.Close()
  log.Println("Successfully connected to the database")

  jwtSecret := os.Getenv("JWT_SECRET")
  if jwtSecret == "" {
    log.Fatal("JWT_SECRET environment variable is not set")
  }

  // Initialize the handler with the database connection and JWT secret
  h := handlers.New(pool, jwtSecret)

  // Set up the Gin router and define routes
  router := gin.Default()

  router.GET("/health", func(c *gin.Context) {
    if err := pool.Ping(c.Request.Context()); err != nil {
      c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
      return
    }
    c.JSON(200, gin.H{"status": "healthy"})
  })

  // Authentication routes
  router.POST("/auth/login", h.Login)

  if err := router.Run(":8080"); err != nil {
    log.Fatalf("Failed to run server: %v", err)
  }
}
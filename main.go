package main

import (
  "context"
  "log"
  "os"

	"resto-backend/internal/db"
  "resto-backend/internal/handlers"
  "resto-backend/internal/middleware"
  "resto-backend/internal/fcm"

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

  if credPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON"); credPath != "" {
    fmcClient, err := fcm.NewClient(context.Background(), credPath)
    if err != nil {
      log.Fatalf("Failed to initialize Firebase Cloud Messaging client: %v", err)
    }
    h.FCM = fmcClient
    log.Println("Firebase Cloud Messaging client initialized successfully")
  } else {
    log.Println("FIREBASE_SERVICE_ACCOUNT_JSON environment variable is not set. FCM will not be initialized.")
  }

  router := setupRouter(h, jwtSecret)

  if err := router.Run(":8080"); err != nil {
    log.Fatalf("Failed to run server: %v", err)
  }
}

func setupRouter(h *handlers.Handler, jwtSecret string) *gin.Engine {
  router := gin.Default()

  // Health check route
  router.GET("/health", func(c *gin.Context) {
    if err := h.DB.Ping(c.Request.Context()); err != nil {
      c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
      return
    }
    c.JSON(200, gin.H{"status": "healthy"})
  })

  // Authentication routes
  router.POST("/auth/login", h.Login)

  // Menu routes
  router.GET("/menu", h.ListMenu)
  router.GET("/menu/:id", h.GetMenuItem)
  
  // Order routes (protected)
  router.POST("/orders", middleware.RequireAuth(jwtSecret), h.CreateOrder)
  router.GET("/orders/:id", middleware.RequireAuth(jwtSecret), h.GetOrder)
  router.PATCH("/orders/:id/status", middleware.RequireAuth(jwtSecret), h.UpdateOrderStatus)

  return router
}
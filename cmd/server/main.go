package main

import (
	"log"
	"os"

	"url-shortener/cache"
	"url-shortener/database"
	"url-shortener/docs"
	"url-shortener/handlers"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title URL Shortener API
// @version 1.0
// @description A simple URL shortener service built with Go and Gin
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

func main() {
	// Initialize Swagger docs
	docs.SwaggerInfo.Title = "URL Shortener API"
	docs.SwaggerInfo.Description = "A simple URL shortener service built with Go and Gin"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Initialize database
	database.InitDB()

	// Initialize Redis cache
	cache.InitRedis()

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Swagger documentation route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// API Routes
	api := r.Group("/")
	{
		api.POST("/shorten", handlers.ShortenURL)
		api.GET("/:shortCode", handlers.RedirectURL)
		api.GET("/stats/:shortCode", handlers.GetURLStats)
		api.GET("/health", handlers.HealthCheck)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Swagger docs available at http://localhost:%s/swagger/index.html", port)
	log.Fatal(r.Run(":" + port))
}

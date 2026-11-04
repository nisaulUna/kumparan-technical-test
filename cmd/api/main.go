package main

import (
	"log"
	"net/http"
	"os"

	"backend-article-service/internal/database"
	"backend-article-service/internal/handler"
	"backend-article-service/internal/repository"
	"backend-article-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Failed to load .env file: ", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not configured")
	}

	// Connect to PostgreSQL
	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Successfully connected to PostgreSQL")

	// Initialize application layers
	authorRepo := repository.NewAuthorRepository(db)
	authorService := service.NewAuthorService(authorRepo)
	authorHandler := handler.NewAuthorHandler(authorService)

	// Initialize article layers
	articleRepo := repository.NewArticleRepository(db)
	articleService := service.NewArticleService(articleRepo)
	articleHandler := handler.NewArticleHandler(articleService)

	// Initialize router

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Article Service API is running",
		})
	})

	// Author routes
	router.POST("/authors", authorHandler.Create)
	router.GET("/authors", authorHandler.GetAll)

	// Article routes
	router.POST("/articles", articleHandler.Create)
	router.GET("/articles", articleHandler.GetAll)
	router.PATCH("/articles/:id", articleHandler.Update)

	// Start server
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

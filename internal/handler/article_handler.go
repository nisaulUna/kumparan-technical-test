package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-article-service/internal/model"
	"backend-article-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ArticleHandler struct {
	Service *service.ArticleService
}

func NewArticleHandler(
	s *service.ArticleService,
) *ArticleHandler {
	return &ArticleHandler{Service: s}
}

// POST /articles
func (h *ArticleHandler) Create(c *gin.Context) {
	var req model.CreateArticleRequest

	// Validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article request",
		})
		return
	}

	// Create article through service
	article, err := h.Service.Create(
		c.Request.Context(),
		req,
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidArticle):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrAuthorNotFound):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Author does not exist",
			})

		default:
			log.Printf("Failed to create article: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create article",
			})
		}
		return
	}

	// Return created article
	c.JSON(http.StatusCreated, article)
}

// GET /articles
func (h *ArticleHandler) GetAll(c *gin.Context) {
	// Default pagination limit
	limit := 20

	// Validate limit parameter
	if value := c.Query("limit"); value != "" {
		n, err := strconv.Atoi(value)

		if err != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "limit must be between 1 and 100",
			})
			return
		}

		limit = n
	}

	// Read query parameters
	params := model.ArticleListParams{
		Query:  strings.TrimSpace(c.Query("query")),
		Author: strings.TrimSpace(c.Query("author")),
		Limit:  limit,
		Cursor: c.Query("cursor"),
	}

	// Set query timeout
	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		5*time.Second,
	)
	defer cancel()

	result, err := h.Service.GetAll(
		ctx,
		params,
	)

	if errors.Is(err, context.DeadlineExceeded) {
		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error": "Article query timed out",
		})
		return
	}

	if err != nil {
		if errors.Is(err, service.ErrInvalidCursor) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid pagination cursor",
			})
			return
		}

		log.Printf("Failed to retrieve articles: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve articles",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UPDATE
func (h *ArticleHandler) Update(c *gin.Context) {
	id := c.Param("id")

	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article ID",
		})
		return
	}

	var fields map[string]json.RawMessage

	if err := c.ShouldBindJSON(&fields); err != nil || len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid update request",
		})
		return
	}

	var req model.UpdateArticleRequest

	for key, value := range fields {
		if string(value) == "null" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Null values are not allowed",
			})
			return
		}

		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Fields must be strings",
			})
			return
		}

		switch key {
		case "title":
			req.Title = &text
		case "body":
			req.Body = &text
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unknown field: " + key,
			})
			return
		}
	}

	article, err := h.Service.Update(
		c.Request.Context(),
		id,
		req,
	)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidArticleUpdate):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Article not found",
			})

		default:
			log.Printf("Failed to update article: %v", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update article",
			})
		}
		return
	}

	c.JSON(http.StatusOK, article)
}

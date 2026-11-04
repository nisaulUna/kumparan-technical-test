package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend-article-service/internal/model"
	"backend-article-service/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthorHandler struct {
	Service *service.AuthorService
}

func NewAuthorHandler(s *service.AuthorService) *AuthorHandler {
	return &AuthorHandler{Service: s}
}

// Create handles POST /authors and returns the created author.
func (h *AuthorHandler) Create(c *gin.Context) {
	var req model.CreateAuthorRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid author request",
		})
		return
	}

	author, err := h.Service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidAuthorName) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create author",
		})
		return
	}

	c.JSON(http.StatusCreated, author)
}

// GetAll handles GET /authors and returns authors with limit-offset pagination.
func (h *AuthorHandler) GetAll(c *gin.Context) {
	limit := 20
	offset := 0

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

	if value := c.Query("offset"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "offset must be non-negative",
			})
			return
		}
		offset = n
	}

	authors, err := h.Service.GetAll(
		c.Request.Context(),
		limit,
		offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve authors",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   authors,
		"limit":  limit,
		"offset": offset,
	})
}

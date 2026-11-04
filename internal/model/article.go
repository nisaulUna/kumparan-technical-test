package model

import "time"

type Article struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateArticleRequest struct {
	AuthorID string `json:"author_id" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body" binding:"required"`
}

type ArticleListParams struct {
	Query  string
	Author string
	Limit  int
	Cursor string
}

type ArticleListResponse struct {
	Data       []Article `json:"data"`
	NextCursor *string   `json:"next_cursor"`
	HasMore    bool      `json:"has_more"`
}

type UpdateArticleRequest struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

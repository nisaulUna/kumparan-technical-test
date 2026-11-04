package service

import (
	"context"
	"errors"
	"strings"

	"encoding/base64"
	"encoding/json"
	"time"

	"backend-article-service/internal/model"
	"backend-article-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidArticle = errors.New("article fields cannot be empty")
	ErrAuthorNotFound = errors.New("author does not exist")
)

type ArticleService struct {
	Repo *repository.ArticleRepository
}

func NewArticleService(
	repo *repository.ArticleRepository,
) *ArticleService {
	return &ArticleService{Repo: repo}
}

// Create new article
func (s *ArticleService) Create(
	ctx context.Context,
	req model.CreateArticleRequest,
) (model.Article, error) {

	// Normalize input
	authorID := strings.TrimSpace(req.AuthorID)
	title := strings.TrimSpace(req.Title)
	body := strings.TrimSpace(req.Body)

	// Validate required fields
	if authorID == "" || title == "" || body == "" {
		return model.Article{}, ErrInvalidArticle
	}

	// Generate unique article ID
	article := model.Article{
		ID:       uuid.NewString(),
		AuthorID: authorID,
		Title:    title,
		Body:     body,
	}

	// Save article to database
	if err := s.Repo.Create(ctx, &article); err != nil {
		var pgErr *pgconn.PgError

		// PostgreSQL foreign key violation
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return model.Article{}, ErrAuthorNotFound
		}

		return model.Article{}, err
	}

	return article, nil
}

var ErrInvalidCursor = errors.New("invalid pagination cursor")

type articleCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Author    string    `json:"author"`
}

func (s *ArticleService) GetAll(
	ctx context.Context,
	params model.ArticleListParams,
) (model.ArticleListResponse, error) {

	params.Query = strings.TrimSpace(params.Query)
	params.Author = strings.TrimSpace(params.Author)

	var cursorTime *time.Time
	var cursorID string

	// Decode and validate pagination cursor
	if params.Cursor != "" {
		if len(params.Cursor) > 2048 {
			return model.ArticleListResponse{}, ErrInvalidCursor
		}

		decoded, err := base64.RawURLEncoding.DecodeString(
			params.Cursor,
		)
		if err != nil {
			return model.ArticleListResponse{}, ErrInvalidCursor
		}

		var cursor articleCursor
		if err := json.Unmarshal(decoded, &cursor); err != nil {
			return model.ArticleListResponse{}, ErrInvalidCursor
		}

		if cursor.CreatedAt.IsZero() ||
			cursor.ID == "" ||
			cursor.Query != params.Query ||
			cursor.Author != params.Author {
			return model.ArticleListResponse{}, ErrInvalidCursor
		}

		cursorTime = &cursor.CreatedAt
		cursorID = cursor.ID
	}

	// Retrieve matching articles
	articles, err := s.Repo.GetAll(
		ctx,
		params.Query,
		params.Author,
		params.Limit,
		cursorTime,
		cursorID,
	)
	if err != nil {
		return model.ArticleListResponse{}, err
	}

	// Determine whether another page exists
	hasMore := len(articles) > params.Limit

	if hasMore {
		articles = articles[:params.Limit]
	}

	var nextCursor *string

	if hasMore {
		last := articles[len(articles)-1]

		cursor := articleCursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
			Query:     params.Query,
			Author:    params.Author,
		}

		encoded, err := json.Marshal(cursor)
		if err != nil {
			return model.ArticleListResponse{}, err
		}

		value := base64.RawURLEncoding.EncodeToString(encoded)
		nextCursor = &value
	}

	return model.ArticleListResponse{
		Data:       articles,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

var (
	ErrInvalidArticleUpdate = errors.New("invalid article update")
	ErrArticleNotFound      = errors.New("article not found")
)

// Update an existing article
func (s *ArticleService) Update(
	ctx context.Context,
	id string,
	req model.UpdateArticleRequest,
) (model.Article, error) {

	// At least one field must be provided
	if req.Title == nil && req.Body == nil {
		return model.Article{}, ErrInvalidArticleUpdate
	}

	// Validate and normalize title
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)

		if title == "" {
			return model.Article{}, ErrInvalidArticleUpdate
		}

		req.Title = &title
	}

	// Validate and normalize body
	if req.Body != nil {
		body := strings.TrimSpace(*req.Body)

		if body == "" {
			return model.Article{}, ErrInvalidArticleUpdate
		}

		req.Body = &body
	}

	// Update article through repository
	article, err := s.Repo.Update(ctx, id, req)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Article{}, ErrArticleNotFound
		}

		return model.Article{}, err
	}

	return article, nil
}

package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend-article-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArticleRepository struct {
	DB *pgxpool.Pool
}

func NewArticleRepository(db *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{DB: db}
}

// Create inserts a new article and reads its database-generated timestamps.
func (r *ArticleRepository) Create(
	ctx context.Context,
	article *model.Article,
) error {
	query := `
		INSERT INTO articles (id, author_id, title, body)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	return r.DB.QueryRow(
		ctx,
		query,
		article.ID,
		article.AuthorID,
		article.Title,
		article.Body,
	).Scan(
		&article.CreatedAt,
		&article.UpdatedAt,
	)
}

// GetAll retrieves filtered articles in newest-first order with one extra row for pagination.
func (r *ArticleRepository) GetAll(
	ctx context.Context,
	query string,
	author string,
	limit int,
	cursorTime *time.Time,
	cursorID string,
) ([]model.Article, error) {

	var sql strings.Builder
	args := make([]any, 0)

	// Base query
	sql.WriteString(`
		SELECT
			a.id,
			a.author_id,
			a.title,
			a.body,
			a.created_at,
			a.updated_at
		FROM articles a
		WHERE 1 = 1
	`)

	// Search keywords in title and body
	if query != "" {
		args = append(args, query)

		sql.WriteString(fmt.Sprintf(`
			AND a.search_vector @@
				plainto_tsquery('simple', $%d)
		`, len(args)))
	}

	// Filter articles by author name
	if author != "" {
		args = append(args, author)

		sql.WriteString(fmt.Sprintf(`
			AND EXISTS (
				SELECT 1
				FROM authors au
				WHERE au.id = a.author_id
				  AND au.name = $%d
			)
		`, len(args)))
	}

	// Keyset pagination using created_at and id
	if cursorTime != nil {
		args = append(args, *cursorTime, cursorID)
		n := len(args)

		sql.WriteString(fmt.Sprintf(`
			AND (a.created_at, a.id) < ($%d, $%d)
		`, n-1, n))
	}

	// Sort articles by newest first
	sql.WriteString(`
		ORDER BY a.created_at DESC, a.id DESC
	`)

	// Fetch one extra record to detect the next page
	args = append(args, limit+1)

	sql.WriteString(fmt.Sprintf(
		" LIMIT $%d", len(args),
	))

	// Execute SQL query
	rows, err := r.DB.Query(
		ctx,
		sql.String(),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map database rows to Article struct
	articles := make([]model.Article, 0)

	for rows.Next() {
		var article model.Article

		err := rows.Scan(
			&article.ID,
			&article.AuthorID,
			&article.Title,
			&article.Body,
			&article.CreatedAt,
			&article.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}

// Update saves the supplied article fields and refreshes the update timestamp.
func (r *ArticleRepository) Update(
	ctx context.Context,
	id string,
	req model.UpdateArticleRequest,
) (model.Article, error) {
	query := `
		UPDATE articles
		SET
			title = COALESCE($2::TEXT, title),
			body = COALESCE($3::TEXT, body),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING
			id,
			author_id,
			title,
			body,
			created_at,
			updated_at
	`

	var article model.Article

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
		req.Title,
		req.Body,
	).Scan(
		&article.ID,
		&article.AuthorID,
		&article.Title,
		&article.Body,
		&article.CreatedAt,
		&article.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Article{}, pgx.ErrNoRows
		}
		return model.Article{}, err
	}

	return article, nil
}

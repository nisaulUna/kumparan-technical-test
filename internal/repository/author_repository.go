package repository

import (
	"context"

	"backend-article-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthorRepository struct {
	DB *pgxpool.Pool
}

func NewAuthorRepository(db *pgxpool.Pool) *AuthorRepository {
	return &AuthorRepository{DB: db}
}

// Insert new author
func (r *AuthorRepository) Create(
	ctx context.Context,
	author model.Author,
) error {
	query := `
		INSERT INTO authors (id, name)
		VALUES ($1, $2)
	`

	_, err := r.DB.Exec(ctx, query, author.ID, author.Name)
	return err
}

// Retrieve authors with pagination
func (r *AuthorRepository) GetAll(
	ctx context.Context,
	limit int,
	offset int,
) ([]model.Author, error) {
	query := `
		SELECT id, name
		FROM authors
		ORDER BY id ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.DB.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := make([]model.Author, 0)

	for rows.Next() {
		var author model.Author

		if err := rows.Scan(&author.ID, &author.Name); err != nil {
			return nil, err
		}

		authors = append(authors, author)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}

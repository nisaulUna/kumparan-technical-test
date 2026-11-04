package service

import (
	"context"
	"errors"
	"strings"

	"backend-article-service/internal/model"
	"backend-article-service/internal/repository"

	"github.com/google/uuid"
)

var ErrInvalidAuthorName = errors.New("author name cannot be empty")

type AuthorService struct {
	Repo *repository.AuthorRepository
}

func NewAuthorService(repo *repository.AuthorRepository) *AuthorService {
	return &AuthorService{Repo: repo}
}

func (s *AuthorService) Create(
	ctx context.Context,
	req model.CreateAuthorRequest,
) (model.Author, error) {
	name := strings.TrimSpace(req.Name)

	if name == "" {
		return model.Author{}, ErrInvalidAuthorName
	}

	author := model.Author{
		ID:   uuid.NewString(),
		Name: name,
	}

	if err := s.Repo.Create(ctx, author); err != nil {
		return model.Author{}, err
	}

	return author, nil
}

func (s *AuthorService) GetAll(
	ctx context.Context,
	limit int,
	offset int,
) ([]model.Author, error) {
	return s.Repo.GetAll(ctx, limit, offset)
}

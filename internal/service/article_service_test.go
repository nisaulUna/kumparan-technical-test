package service

import (
	"context"
	"errors"
	"testing"

	"backend-article-service/internal/model"
)

// Test invalid PATCH requests
func TestUpdateArticleValidation(t *testing.T) {
	tests := []struct {
		name string
		req  model.UpdateArticleRequest
	}{
		{
			name: "No fields provided",
			req:  model.UpdateArticleRequest{},
		},
		{
			name: "Empty title",
			req:  model.UpdateArticleRequest{Title: strPtr("")},
		},
		{
			name: "Whitespace title",
			req:  model.UpdateArticleRequest{Title: strPtr("   ")},
		},
		{
			name: "Empty body",
			req:  model.UpdateArticleRequest{Body: strPtr("")},
		},
		{
			name: "Whitespace body",
			req:  model.UpdateArticleRequest{Body: strPtr("   ")},
		},
	}

	s := &ArticleService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Update(
				context.Background(),
				"test-id",
				tt.req,
			)

			if !errors.Is(err, ErrInvalidArticleUpdate) {
				t.Errorf(
					"Expected ErrInvalidArticleUpdate, got %v",
					err,
				)
			}
		})
	}
}

// Test invalid pagination cursor
func TestInvalidPaginationCursor(t *testing.T) {
	s := &ArticleService{}

	params := model.ArticleListParams{
		Limit:  20,
		Cursor: "invalid-cursor",
	}

	_, err := s.GetAll(context.Background(), params)

	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf(
			"Expected ErrInvalidCursor, got %v",
			err,
		)
	}
}

func strPtr(value string) *string {
	return &value
}

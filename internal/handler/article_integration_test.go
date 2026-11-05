package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"backend-article-service/internal/handler"
	"backend-article-service/internal/model"
	"backend-article-service/internal/repository"
	"backend-article-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestArticleIntegration verifies author and article endpoints against PostgreSQL.
func TestArticleIntegration(t *testing.T) {
	// 1. Load the explicitly configured database
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Verify database identity
	var dbName string
	err = db.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&dbName)

	if err != nil {
		db.Close()
		t.Fatalf("Failed to query test database identity: %v", err)
	}

	if dbName != "article_service" {
		db.Close()
		t.Fatalf("Test must connect to article_service; connected to %q", dbName)
	}

	// 3. Track only records created by this test
	var authorIDs []string
	var articleIDs []string

	t.Cleanup(func() {
		defer db.Close()

		for _, id := range articleIDs {
			if _, err := db.Exec(
				context.Background(),
				"DELETE FROM articles WHERE id = $1",
				id,
			); err != nil {
				t.Errorf("Article cleanup failed: %v", err)
			}
		}

		for _, id := range authorIDs {
			if _, err := db.Exec(
				context.Background(),
				"DELETE FROM authors WHERE id = $1",
				id,
			); err != nil {
				t.Errorf("Author cleanup failed: %v", err)
			}
		}
	})

	// 4. Initialize application layers
	authorRepo := repository.NewAuthorRepository(db)
	authorService := service.NewAuthorService(authorRepo)
	authorHandler := handler.NewAuthorHandler(authorService)

	articleRepo := repository.NewArticleRepository(db)
	articleService := service.NewArticleService(articleRepo)
	articleHandler := handler.NewArticleHandler(articleService)

	// 5. Initialize test router
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST("/authors", authorHandler.Create)
	router.GET("/authors", authorHandler.GetAll)

	router.POST("/articles", articleHandler.Create)
	router.GET("/articles", articleHandler.GetAll)
	router.PATCH("/articles/:id", articleHandler.Update)

	// 6. Helper to send HTTP requests
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()

		var payload []byte

		if body != nil {
			var err error
			payload, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}

		req := httptest.NewRequest(
			method,
			path,
			bytes.NewReader(payload),
		)

		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		return recorder
	}

	// 7. Helpers to create test records
	createAuthor := func(name string) model.Author {
		t.Helper()

		w := send("POST", "/authors", map[string]string{
			"name": name,
		})

		if w.Code != http.StatusCreated {
			t.Fatalf("POST author failed: %s", w.Body.String())
		}

		var author model.Author
		if err := json.Unmarshal(w.Body.Bytes(), &author); err != nil {
			t.Fatal(err)
		}

		authorIDs = append(authorIDs, author.ID)
		return author
	}

	createArticle := func(authorID, title, body string) model.Article {
		t.Helper()

		w := send("POST", "/articles", model.CreateArticleRequest{
			AuthorID: authorID,
			Title:    title,
			Body:     body,
		})

		if w.Code != http.StatusCreated {
			t.Fatalf("POST article failed: %s", w.Body.String())
		}

		var article model.Article
		if err := json.Unmarshal(w.Body.Bytes(), &article); err != nil {
			t.Fatal(err)
		}

		articleIDs = append(articleIDs, article.ID)
		return article
	}

	readArticles := func(path string) model.ArticleListResponse {
		t.Helper()

		w := send("GET", path, nil)

		if w.Code != http.StatusOK {
			t.Fatalf("GET failed: %d %s", w.Code, w.Body.String())
		}

		var result model.ArticleListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}

		return result
	}

	// 8. Generate unique test data
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")

	authorName := "TestAndi" + suffix
	keyword := "testgolang" + suffix
	newKeyword := "testrust" + suffix

	andi1 := createAuthor(authorName)
	andi2 := createAuthor(authorName)
	budi := createAuthor("TestBudi" + suffix)

	first := createArticle(
		andi1.ID,
		"Belajar "+keyword,
		"Tutorial backend untuk pemula",
	)

	second := createArticle(
		andi2.ID,
		"Learning backend",
		"Panduan "+keyword+" for developers",
	)

	third := createArticle(
		budi.ID,
		"Introduction to "+keyword,
		"English programming tutorial",
	)

	// TEST 1: Keyword Search
	t.Run("Keyword Search", func(t *testing.T) {
		result := readArticles(
			"/articles?query=" + keyword,
		)

		if len(result.Data) != 3 {
			t.Fatalf(
				"Expected 3 articles, got %d",
				len(result.Data),
			)
		}

		seen := map[string]bool{}
		for _, article := range result.Data {
			seen[article.ID] = true
		}

		if !seen[first.ID] || !seen[second.ID] || !seen[third.ID] {
			t.Fatal("Keyword search did not return all matching articles")
		}
	})

	// TEST 2: Duplicate Author Names
	t.Run("Duplicate Author Names", func(t *testing.T) {
		path := "/articles?author=" + url.QueryEscape(authorName)

		result := readArticles(path)

		if len(result.Data) != 2 {
			t.Fatalf(
				"Expected 2 articles, got %d",
				len(result.Data),
			)
		}

		seen := map[string]bool{}

		for _, article := range result.Data {
			seen[article.AuthorID] = true
		}

		if !seen[andi1.ID] || !seen[andi2.ID] {
			t.Fatal("Articles from both author IDs were not found")
		}
	})

	// TEST 3: Combined Search and Author Filter
	t.Run("Combined Filter", func(t *testing.T) {
		path := "/articles?query=" + keyword +
			"&author=" + url.QueryEscape(authorName)

		result := readArticles(path)

		if len(result.Data) != 2 {
			t.Fatalf(
				"Expected 2 articles, got %d",
				len(result.Data),
			)
		}

		for _, article := range result.Data {
			if article.AuthorID == budi.ID {
				t.Fatal("Article from wrong author was returned")
			}
		}
	})

	// TEST 4: Cursor Pagination
	t.Run("Cursor Pagination", func(t *testing.T) {
		path := "/articles?query=" + keyword +
			"&author=" + url.QueryEscape(authorName) +
			"&limit=1"

		page1 := readArticles(path)

		if len(page1.Data) != 1 ||
			!page1.HasMore ||
			page1.NextCursor == nil {
			t.Fatal("Invalid first pagination page")
		}

		page2 := readArticles(
			path + "&cursor=" + url.QueryEscape(*page1.NextCursor),
		)

		if len(page2.Data) != 1 {
			t.Fatal("Expected one article on second page")
		}

		if page1.Data[0].ID == page2.Data[0].ID {
			t.Fatal("Duplicate article across pages")
		}

		if page2.HasMore {
			t.Fatal("Unexpected extra page")
		}
	})

	// TEST 5: PATCH Article and Timestamp
	t.Run("Update Article", func(t *testing.T) {
		w := send(
			"PATCH",
			"/articles/"+first.ID,
			map[string]string{
				"title": "Belajar " + newKeyword,
				"body":  "Updated article about Rust",
			},
		)

		if w.Code != http.StatusOK {
			t.Fatalf("PATCH failed: %s", w.Body.String())
		}

		var updated model.Article
		if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
			t.Fatal(err)
		}

		if !updated.CreatedAt.Equal(first.CreatedAt) {
			t.Fatal("created_at changed after PATCH")
		}

		if updated.UpdatedAt.Before(first.UpdatedAt) {
			t.Fatal("updated_at moved backwards")
		}

		// Verify the new keyword is searchable
		newResults := readArticles(
			"/articles?query=" + newKeyword,
		)

		found := false
		for _, article := range newResults.Data {
			if article.ID == first.ID {
				found = true
			}
		}

		if !found {
			t.Fatal("Updated article was not found by new keyword")
		}

		// Verify the old keyword no longer matches
		oldResults := readArticles(
			"/articles?query=" + keyword,
		)

		for _, article := range oldResults.Data {
			if article.ID == first.ID {
				t.Fatal("Old keyword still matches updated article")
			}
		}
	})

	// TEST 6: Invalid Pagination
	t.Run("Invalid Cursor", func(t *testing.T) {
		w := send(
			"GET",
			"/articles?cursor=invalid-cursor",
			nil,
		)

		if w.Code != http.StatusBadRequest {
			t.Fatalf(
				"Expected 400, got %d",
				w.Code,
			)
		}
	})
}

# Backend Article Service

A lightweight REST API built with Go (Golang), Gin, and PostgreSQL for managing authors and articles. The service supports article creation, keyword search across titles and bodies, filtering by author name, newest-first ordering, cursor-based pagination, and article updates.

## 1. Tech Stack

- **Language:** Go (Golang)
- **Web Framework:** Gin
- **Database:** PostgreSQL
- **Database Driver:** pgx / pgxpool
- **ID Generation:** UUID v4
- **Search:** PostgreSQL Full-Text Search (TSVECTOR, GIN index)
- **Testing:** Go testing package, httptest
- **API Testing:** Postman
- **Architecture:** Handler → Service → Repository
- **ORM:** None; parameterized SQL queries are used directly

## 2. Features

- Create and list authors
- Create and list articles
- Search article titles and bodies using keywords
- Filter articles by author name
- Support duplicate author names with distinct author IDs
- Sort articles by newest creation time
- Paginate results using keyset/cursor pagination
- Update article titles and bodies without modifying their creation timestamps
- Automatically refresh the search vector after article content changes
- Validate requests and return appropriate HTTP responses

## 3. Project Structure

```text
cmd/api/              Application entry point
internal/database/    PostgreSQL connection pool
internal/model/       Data and request models
internal/repository/  SQL queries and database access
internal/service/     Business logic and validation
internal/handler/     HTTP handlers and responses
scripts/              Load testing utility
schema.sql            Database schema and indexes
.env.example          Example environment configuration
```

## 4. Database Design

### Authors

| Column | Type | Description |
|---|---|---|
| id | TEXT | Primary key, UUID v4 |
| name | TEXT | Author name, not necessarily unique |

### Articles

| Column | Type | Description |
|---|---|---|
| id | TEXT | Primary key, UUID v4 |
| author_id | TEXT | Foreign key to authors |
| title | TEXT | Article title |
| body | TEXT | Article content |
| created_at | TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | Last update timestamp |
| search_vector | TSVECTOR | Generated keyword search vector |

### Indexing

- `idx_articles_search`: GIN index for full-text search
- `idx_articles_created`: B-tree index for newest-first ordering and cursor pagination
- `idx_authors_name`: B-tree index for author-name lookup
- `idx_articles_author_created`: B-tree index for author-specific listing by creation time

## 5. Installation and Setup

### Requirements

- Go 1.27.3 or later
- PostgreSQL
- Git

### Clone the Repository

```bash
git clone https://github.com/nisaulUna/kumparan-technical-test.git
cd backend-article-service
```

### Install Dependencies

```bash
go mod download
```

### Create Database

Create a PostgreSQL database named:

```text
article_service
```

Run the SQL statements from `schema.sql` PostgreSQL command-line tools.

For example:

```bash
psql -U postgres -d article_service -f schema.sql
```

### Configure Environment

Copy `.env.example` to `.env` and update the PostgreSQL connection string.

```env
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@localhost:5432/article_service?sslmode=disable
```

### Run the Application

```bash
go run ./cmd/api
```

The API is available at:

```text
http://localhost:8080
```

Health check:

```http
GET /health
```

## 6. API Documentation

### POST /authors

Create an author.

Request:

```json
{
  "name": "Nisa"
}
```

### GET /authors

Retrieve a list of authors.

Example:

```http
GET /authors?limit=20&offset=0
```

Parameters:

- `limit`: Maximum number of authors to return
- `offset`: Number of records to skip

### POST /articles

Create an article associated with an existing author.

Request:

```json
{
  "author_id": "EXISTING_AUTHOR_UUID",
  "title": "Pemerintah Rampungkan 1.027 DIM RUU Adminduk, Target Masuk DPR Awal November",
  "body": "Pemerintah menyelesaikan pembahasan 1.027 Daftar Inventarisasi Masalah (DIM) Rancangan Undang-Undang Administrasi Kependudukan pada 27 Oktober 2026. Dokumen tersebut disusun melalui koordinasi sejumlah kementerian dan lembaga sebelum disampaikan kepada DPR RI. Wakil Menteri Dalam Negeri Bima Arya Sugiarto menargetkan penyampaian dokumen paling lambat 8 November 2026 untuk melanjutkan proses pembahasan legislasi."
}
```

### GET /articles

Retrieve articles ordered by `created_at DESC, id DESC`.

Supported query parameters:

| Parameter | Description |
|---|---|
| query | Full-text search over title and body |
| author | Exact author-name filter |
| limit | Maximum page size, 1–100 |
| cursor | Cursor returned by a previous page |

Examples:

```http
GET /articles
GET /articles?query=golang
GET /articles?author=Andi
GET /articles?query=golang&author=Andi
GET /articles?limit=2
GET /articles?limit=2&cursor=NEXT_CURSOR
```

Response structure:

```json
{
  "data": [],
  "next_cursor": null,
  "has_more": false
}
```

The `data` array contains article objects. `next_cursor` is populated when another page is available.

### PATCH /articles/:id

Update an article title, body, or both.

Example:

```http
PATCH /articles/ARTICLE_UUID
```

Request:

```json
{
  "title": "Pemerintah Rampungkan 1.027 DIM RUU Adminduk, Target Masuk DPR Awal November",
  "body": "Pemerintah menyelesaikan pembahasan 1.027 Daftar Inventarisasi Masalah (DIM) Rancangan Undang-Undang Administrasi Kependudukan pada 27 Oktober 2026. Dokumen tersebut disusun melalui koordinasi sejumlah kementerian dan lembaga sebelum disampaikan kepada DPR RI. Wakil Menteri Dalam Negeri Bima Arya Sugiarto menargetkan penyampaian dokumen paling lambat 8 November 2026 untuk melanjutkan proses pembahasan legislasi."
}
```

The `created_at` field remains unchanged. The `updated_at` field is refreshed.

## 7. Search Implementation

Article search uses PostgreSQL Full-Text Search with the `simple` text search configuration.

A generated `TSVECTOR` combines the title and body:

```sql
to_tsvector('simple', title || ' ' || body)
```

Search queries use:

```sql
search_vector @@ plainto_tsquery('simple', $1)
```

A GIN index supports full-text lookup.

The `simple` configuration is suitable for basic token-based search across mixed Indonesian and English content without language-specific stemming.

**Limitation:** This is lexical search, not semantic search. It does not automatically translate words or match equivalent terms across languages. Language-aware ranking, typo tolerance, and multilingual stemming would require additional design.

## 8. Pagination Strategy

The API uses keyset pagination instead of conventional OFFSET pagination for article listings.

Ordering:

```sql
ORDER BY created_at DESC, id DESC
```

Cursor condition:

```sql
(created_at, id) < ($1, $2)
```

Using `id` as a secondary ordering key provides deterministic results when multiple articles share the same creation timestamp.

The cursor also retains the search and author filters, preventing a cursor from being reused with incompatible query conditions.

This design avoids scanning and discarding increasingly large OFFSET ranges during deep pagination.

## 9. Scalability Considerations

The assessment scenario includes approximately 50 million articles, daily growth of 200,000 articles, a read-heavy workload, and uneven distribution across authors.

The current design addresses these requirements through:

**Database indexing**

Search, sorting, and author filtering use dedicated indexes. Actual planner choices depend on data distribution, query selectivity, and table size.

**Read-heavy access**

Article retrieval uses indexed queries and bounded pagination. Production deployments may additionally benefit from caching, replicas, or connection pooling at the infrastructure layer.

**Skewed author distribution**

A high-volume author may own a substantial share of all articles. This can make filtering by author less selective and should be considered when evaluating query plans.

**Connection pooling**

The application uses `pgxpool` to reuse database connections and prevent unbounded connection creation under concurrent requests.

**Query timeout**

Article listing uses a bounded execution context to prevent excessively long database operations.

**Concurrency**

Go and Gin support concurrent HTTP request handling. PostgreSQL connection pooling limits the number of simultaneous database connections per application instance.

**Future scaling options**

Potential improvements include read replicas, caching, partitioning where justified by real query patterns, advanced search strategies, and production monitoring.

These are design considerations, not features already implemented.

## 10. Testing

The project includes unit tests and integration tests.

### Unit Tests

Validate article updates and invalid pagination cursors.

```bash
go test ./internal/service -v
```

### Integration Tests

Exercise the HTTP handlers against PostgreSQL, including:

- Keyword search in article titles and bodies
- Duplicate author names
- Combined keyword and author filtering
- Cursor pagination
- Article updates
- Creation timestamp preservation
- Search results after updates
- Invalid cursor handling

The integration tests require a configured `TEST_DATABASE_URL`.

On PowerShell:

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@localhost:5432/article_service?sslmode=disable"
```

Run:

```bash
go test ./internal/handler -v
go test ./... -v
go test -race ./...
```

The tests create temporary records with unique identifiers and attempt to remove their own test data afterward.

A dedicated isolated test database is recommended for CI and production-grade testing.

## 11. Preliminary Load Test

A local concurrent-read test was conducted using `GET /articles?limit=20`.

| Metric | Result |
|---|---|
| Total requests | 100 |
| Maximum concurrency | 20 |
| Successful | 100 |
| Failed | 0 |
| Total duration | 184.94 ms |
| Throughput | 540.71 requests/sec |

The test used a small local database. These results confirm successful handling of the tested workload only and do not prove performance at 50 million records.

The benchmark does not yet capture realistic write ratios, large-scale author skew, or latency percentiles.

To repeat the basic test:

```bash
go run ./scripts/loadtest.go
```

## 12. AI Assistance Disclosure

AI assistance was used during development to support:

- Initial project structuring
- SQL schema and query drafting
- Test scenario preparation
- Debugging explanation

The generated suggestions were reviewed, corrected, and tested during local development.

### Example: AI-Assisted Integration Test Configuration Issue

An initial AI-assisted testing setup expected a `TEST_DATABASE_URL` environment variable but did not automatically load it from the application's `.env` file.

Running the integration tests initially failed with:

```text
TEST_DATABASE_URL is not configured
```

The issue occurred because `go test` executes test packages directly and does not run the application's `main.go`, where environment loading was configured.

**Correction:** The test database connection string was explicitly configured in the PowerShell environment before running the tests. After this configuration, the integration tests passed.

## 13. Known Limitations

- Full-text search is token-based and does not provide semantic multilingual matching.
- Load tests were performed using a small local dataset.
- Cursor pagination does not provide a fixed snapshot across concurrent insertions or updates.
- The cursor is encoded but not cryptographically signed.
- UUID identifiers are stored as TEXT, which is less storage-efficient than PostgreSQL's native UUID type.
# IsItAlive?

**A Go REST API for checking website availability and tracking historical check results.**

IsItAlive? allows users to submit a URL, check whether the website responds, measure its response time, and store the result in a PostgreSQL database.

## Features

- **Website availability checks:** Send HTTP GET requests to submitted URLs.
- **Response time measurement:** Measure how long a website takes to respond.
- **HTTP status tracking:** Capture the target website's HTTP status code.
- **Persistent storage:** Store check results in PostgreSQL.
- **Check history:** Retrieve previously performed checks.
- **Pagination:** Control the number of returned records using `limit` and `offset`.
- **Input validation:** Validate URLs and JSON request bodies.
- **Request timeouts:** Prevent requests from hanging indefinitely.
- **Error handling:** Return appropriate HTTP error responses.
- **Automated tests:** Test API behavior and error scenarios.

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Backend programming language |
| `net/http` | HTTP server and routing |
| PostgreSQL | Relational database |
| `database/sql` | Database connectivity |
| sqlc | Type-safe SQL code generation |
| Goose | Database migrations |
| `github.com/lib/pq` | PostgreSQL driver |
| `github.com/google/uuid` | UUID identifiers |
| `godotenv` | Environment variable loading |

## Project Structure

```text
isitalive/
├── internal/
│   └── database/       # sqlc generated database code
├── sql/
│   ├── queries/        # SQL queries
│   └── schema/         # Database migrations/schema
├── main.go             # Application entry point and HTTP handlers
├── database.go         # Database connection setup
├── sqlc.yaml           # sqlc configuration
├── .env                # Environment variables (not committed)
├── .gitignore
└── README.md
```

## Getting Started

### Prerequisites

Make sure you have installed:

- [Go](https://go.dev/dl/)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev/)

### 1. Clone the repository

```bash
git clone https://github.com/Eyob49/isitalive.git
cd isitalive
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure environment variables

Create a `.env` file in the project root:

```env
DB_URL=postgres://USERNAME:PASSWORD@localhost:5432/isitalive?sslmode=disable
```

Replace `USERNAME` and `PASSWORD` with your PostgreSQL credentials.

Make sure `.env` is included in `.gitignore`.

### 4. Create the database

Create a PostgreSQL database named:

```sql
CREATE DATABASE isitalive;
```

Configure the database connection in your `.env` file.

### 5. Apply database migrations

Apply the Goose migrations from the project's schema directory using your configured migration setup.

### 6. Generate database code

```bash
sqlc generate
```

### 7. Run the application

```bash
go run .
```

The server starts at:

```text
http://localhost:8080
```

## API Documentation

### 1. Health Check

Check whether the API server is running.

**Endpoint**

```http
GET /health
```

**Response**

```http
HTTP/1.1 200 OK
```

```text
OK
```

### 2. Check a Website

Submit a URL to check its availability.

**Endpoint**

```http
POST /check
```

**Request Body**

```json
{
  "url": "https://example.com"
}
```

**Successful Response**

```json
{
  "id": "28aba1a8-454e-4089-b534-a69bb6f5eb74",
  "url": "https://example.com",
  "status_code": 200,
  "response_time_ms": 1966,
  "checked_at": "2026-11-04T20:51:28Z"
}
```

The API returns the target website's HTTP status code in the JSON response. A successful API check does not necessarily mean the target website returned HTTP 200.

### 3. Retrieve Check History

Retrieve previously saved website checks.

**Endpoint**

```http
GET /checks
```

By default, the API returns the latest 10 records.

**Response**

```json
[
  {
    "id": "28aba1a8-454e-4089-b534-a69bb6f5eb74",
    "url": "https://example.com",
    "status_code": 200,
    "response_time_ms": 1966,
    "checked_at": "2026-11-04T20:51:28Z"
  }
]
```

### 4. Pagination

The history endpoint supports two query parameters:

| Parameter | Default | Description |
|---|---:|---|
| `limit` | 10 | Maximum number of records to return (1–100) |
| `offset` | 0 | Number of records to skip (non-negative) |

**Examples**

Retrieve the first 10 records:

```http
GET /checks
```

Retrieve the first 5 records:

```http
GET /checks?limit=5
```

Retrieve the next 5 records:

```http
GET /checks?limit=5&offset=5
```

Retrieve records after skipping 20:

```http
GET /checks?limit=10&offset=20
```

## Error Handling

The API uses appropriate HTTP status codes to communicate errors.

| Status Code | Meaning |
|---|---|
| 400 | Invalid request, URL, JSON, or query parameter |
| 405 | HTTP method not allowed |
| 413 | Request payload too large |
| 500 | Internal server or database error |
| 502 | Target website request failed |
| 504 | Target website request timed out |

## Running Tests

Run all tests:

```bash
go test ./...
```

Run static analysis:

```bash
go vet ./...
```

## Current Scope

IsItAlive? currently supports manual website availability checks and persistent check history.

The project is being developed as a learning project to explore Go backend engineering, REST API development, PostgreSQL integration, SQL query generation, HTTP communication, and automated testing.

## Future Improvements

Potential future features include:

- Scheduled website monitoring.
- Historical uptime statistics.
- Availability monitoring over time.
- Better pagination strategies.
- Authentication and user-specific monitoring.
- Notification system for website downtime.
- Docker support and deployment automation.

## Author

**Eyob**

GitHub: [@Eyob49](https://github.com/Eyob49)

---

*Built with Go, PostgreSQL, and curiosity about backend engineering.*
# Car Service

A production-style REST API for managing cars, built with Go.

The project focuses on backend engineering fundamentals rather than business logic, including clean architecture, SQL-first development, Docker, database migrations, testing, and cloud deployment.

---

## Features

- RESTful CRUD API
- PostgreSQL with pgx
- SQL-first development using sqlc
- Database migrations with golang-migrate
- Docker & Docker Compose
- Structured JSON logging
- Graceful shutdown
- Environment-based configuration
- Unit tests
- GitHub Actions CI
- Health check endpoint

---

## Tech Stack

| Technology | Purpose |
|------------|---------|
| Go | Backend |
| Chi | HTTP Router |
| PostgreSQL | Database |
| pgx | PostgreSQL Driver |
| sqlc | Type-safe SQL generation |
| golang-migrate | Database migrations |
| Docker | Containerization |
| GitHub Actions | Continuous Integration |

---

## Project Structure

```text
.
├── cmd/
│   └── car_service/
├── database/
│   ├── migrations/
│   └── queries/
├── internal/
│   ├── config/
│   ├── db/
│   ├── handler/
│   ├── middleware/
│   ├── service/
│   └── router.go
├── tests/
├── Dockerfile
├── docker-compose.yml
├── sqlc.yaml
└── README.md
```

---

## API

| Method | Endpoint | Description |
|---------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/cars` | List and filter cars |
| GET | `/cars/{id}` | Get car by ID |
| POST | `/cars` | Create car |
| PUT | `/cars/{id}` | Update car |
| DELETE | `/cars/{id}` | Delete car |

### Filtering cars

`GET /cars` accepts these optional query parameters:

| Parameter | Meaning |
|-----------|---------|
| `name` | Case-insensitive partial match against brand, model, or both |
| `year` | Exact production year (`production_year` is also accepted) |
| `created_from`, `created_to` | Inclusive creation-time range in RFC3339 format |
| `created_at` | Exact creation time in RFC3339 format |
| `min_price`, `max_price` | Inclusive price range |
| `price` | Exact price |
| `page`, `limit` | Pagination; limit is between 1 and 100 |

Example:

```bash
curl "http://localhost:8080/cars?name=BMW&year=2023&created_from=2026-01-01T00:00:00Z&min_price=30000&max_price=60000"
```

---

## Swagger

After starting the service, open [Swagger UI](http://localhost:8080/swagger/index.html). It documents the health, ping, and car CRUD endpoints.

To document a new endpoint, add Swag annotations above its handler and regenerate the spec with the project-pinned generator version:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.8.1 init -g cmd/car_service/main.go -o docs
```

---

## Quick Start

Clone the repository.

```bash
git clone https://github.com/InatoInato/car_service.git
cd car_service
```

Start the application.

```bash
docker compose up --build
```

The API will be available at

```
http://localhost:8080
```

Health check

```bash
curl http://localhost:8080/health
```

Create a car

```bash
curl -X POST http://localhost:8080/cars \
-H "Content-Type: application/json" \
-d '{
  "brand":"BMW",
  "model":"X5",
  "production_year":2023,
  "color":"Blue",
  "price":42000
}'
```

---

## Configuration

Configuration is provided through environment variables.

Create a local configuration file before running the application.

```bash
cp .env.example .env
```

---

## Running Tests

Run all unit tests.

```bash
go test ./...
```

Run the HTTP and PostgreSQL integration tests against the Docker Compose stack.

```bash
docker compose up -d --build
go test -tags=integration ./test -run Integration -count=1
```

Set `CAR_SERVICE_BASE_URL` to test another running instance.

Run formatting checks.

```bash
gofmt -w .
go vet ./...
```

---

## CI

Every push and pull request automatically runs:

- Go formatting
- go vet
- Unit tests
- Application build
- Docker image build

---

## Architecture

```
             HTTP Request
                  │
                  ▼
            Chi Router
                  │
                  ▼
             Handlers
                  │
                  ▼
             Services
                  │
                  ▼
           sqlc Queries
                  │
                  ▼
             PostgreSQL
```

---

## Roadmap

## Roadmap

### Application Foundation
- [x] REST API
- [x] PostgreSQL
- [x] sqlc
- [x] Database migrations
- [x] Structured logging
- [x] Graceful shutdown
- [x] Configuration validation
- [x] Request ID middleware

### Containerization
- [x] Docker
- [x] Docker Compose
- [x] Docker health checks

### Testing & Quality
- [x] Unit tests
- [x] Integration tests
- [x] GitHub Actions CI

### Application Features
- [x] Redis cache
- [x] Pagination
- [x] OpenAPI / Swagger

### Deployment
- [ ] Deployment to AWS EC2
- [ ] GitHub Actions CD

### Infrastructure
- [x] Terraform infrastructure

### Observability
- [ ] Metrics (/metrics with Prometheus)

### Orchestration
- [ ] Kubernetes

---

## Learning Goals

This project is built to practice production-oriented backend engineering:

- Clean Architecture
- SQL-first development
- Containerization
- Database migrations
- Automated testing
- CI/CD
- Cloud deployment
- Infrastructure as Code

---

## License

This project is intended for educational and portfolio purposes.

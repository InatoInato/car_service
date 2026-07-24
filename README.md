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
| GET | `/cars` | List all cars |
| GET | `/cars/{id}` | Get car by ID |
| POST | `/cars` | Create car |
| PUT | `/cars/{id}` | Update car |
| DELETE | `/cars/{id}` | Delete car |

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

- [x] REST API
- [x] PostgreSQL
- [x] sqlc
- [x] Docker
- [x] Docker Compose
- [x] Database migrations
- [x] Structured logging
- [x] Graceful shutdown
- [x] Unit tests
- [x] GitHub Actions CI
- [ ] Redis cache
- [ ] Integration tests
- [x] Docker health checks
- [ ] Deployment to AWS EC2
- [ ] Terraform infrastructure
- [ ] Kubernetes deployment

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

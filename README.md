# Car Service

A cloud-native REST API for managing cars, written in Go.

This project is built as a production-style backend service without using an ORM. The goal is to focus on clean architecture, SQL, Docker, and cloud deployment rather than business complexity.

## Tech Stack

* Go
* Chi Router
* PostgreSQL
* pgx
* Redis
* Docker & Docker Compose
* golang-migrate

## Features

* REST API
* CRUD operations for cars
* PostgreSQL storage
* Redis caching
* Graceful shutdown
* Environment-based configuration
* Structured JSON logging
* Docker support

## Project Structure

```text
cmd/            Application entrypoint
internal/       Application source code
migrations/     Database migrations
docker/         Dockerfiles
docs/           Documentation
scripts/        Helper scripts
tests/          Tests
```

## API

| Method | Endpoint     | Description     |
| ------ | ------------ | --------------- |
| GET    | `/health`    | Health check    |
| GET    | `/cars`      | List cars       |
| POST   | `/cars`      | Create a car    |
| GET    | `/cars/{id}` | Get a car by ID |
| PUT    | `/cars/{id}` | Update a car    |
| DELETE | `/cars/{id}` | Delete a car    |

## Running Locally

Clone the repository:

```bash
git clone https://github.com/<your-github>/car-service.git
cd car-service
```

Start the application:

```bash
docker compose up --build
```

Or run the API directly:

```bash
go run ./cmd/api
```

## Configuration

Application settings are provided through environment variables.

Create a `.env` file from `.env.example` before starting the service.

## Development Roadmap

* [x] Project boilerplate
* [ ] Configuration
* [ ] HTTP server
* [ ] PostgreSQL integration
* [ ] Redis integration
* [ ] CRUD API
* [ ] Validation
* [ ] Logging middleware
* [ ] Unit tests
* [ ] Integration tests
* [ ] CI/CD with GitHub Actions
* [ ] AWS deployment
* [ ] Kubernetes

## Architecture

```
Client
   │
HTTP API
   │
Handlers
   │
Services
   │
Repositories
   │
PostgreSQL

Redis (Cache)
```

## Goals

This project is intended to demonstrate:

* Clean Architecture principles
* Production-ready Go practices
* SQL without an ORM
* Docker-based development
* Cloud-native application design
* AWS deployment workflow

## License

This project is available for educational and learning purposes.

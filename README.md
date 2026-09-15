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

## AWS setup

This setup uses a small EC2 server. SSH and port 8080 are open only to your chosen
IP address. The API still has no login system, so keep this rule narrow.

### 1. Choose fixed settings

Use Terraform 1.10 or newer. Create `deployment/terraform/terraform.tfvars`:

```hcl
aws_region       = "us-east-1"
instance_type    = "t2.micro"
ssh_public_key   = "YOUR SSH PUBLIC KEY"
ami_id           = "ami-REPLACE_ME"
subnet_id        = "subnet-REPLACE_ME"
allowed_ssh_cidr = "YOUR.PUBLIC.IP.ADDRESS/32"
```

Replace all placeholders. Never put a private SSH key here.

For an existing EC2 server, copy its current AMI ID and subnet ID from the EC2
console. This avoids an unwanted replacement. For a new server, choose a Canonical
Ubuntu 22.04 amd64 image and a public subnet in your region. The subnet must have
a route to an Internet Gateway. The security group uses that subnet's VPC.

Use your real public IPv4 address with `/32`. When your IP changes, update this
value yourself. Terraform no longer reads the IP of a laptop or CI runner.

### 2. Initialize Terraform (no S3 needed)

Run commands from the repository root. Terraform stores its resource record locally
in `deployment/terraform/terraform.tfstate`. Keep a private backup and run Terraform
from the same checkout. Do not commit state or run simultaneous applies from other computers.

```bash
terraform -chdir=deployment/terraform init
```

If you previously used a working remote backend, back up its state and use
`init -migrate-state` to copy it locally. Do not discard existing state or use
`-reconfigure` to bypass migration. A previously failed S3 initialization does
not require a bucket now.

Use AWS credentials through your normal AWS CLI profile or role. Check them with:

```bash
aws sts get-caller-identity
```

The required AMI, subnet, SSH key and public IP are account/user-specific; Terraform
cannot safely invent them. Merge the fields in `deployment/terraform/terraform.tfvars.example`
into your existing `terraform.tfvars`, preserving any existing server's AMI and subnet.

### 3. Check the plan before applying

```bash
terraform -chdir=deployment/terraform validate
terraform -chdir=deployment/terraform test
terraform -chdir=deployment/terraform plan -out=deploy.tfplan
# Review the plan before creating AWS resources:
terraform -chdir=deployment/terraform apply deploy.tfplan
```

The tests use a fake AWS provider. They do not create resources or test a live server.

Check the AWS account, region, IP rule, and any changes to the server or disk.
If the plan says the server must be replaced, stop. Check the AMI and subnet IDs.
Do not remove `prevent_destroy` just to make the plan pass.

The code keeps the existing resource names. It also adds a snapshot policy and
an IAM role used by AWS to create backups. Your Terraform operator needs permission
to create that role, attach its policy, pass it to DLM, and create a DLM policy.

### 4. Understand disk protection and backups

Terraform blocks server replacement. AWS termination protection also blocks an
accidental termination request. The root disk is kept if you later choose to remove
these protections and terminate the server. Retaining a disk does not attach it to
a new server automatically, and it does not protect against manual disk deletion.

AWS DLM schedules a snapshot each day at 03:00 UTC and keeps the last seven
snapshots. The `Backup = "car-service-daily"` tag selects the disk. Use this tag only
for this stack. Snapshots and retained disks have storage costs. A snapshot is not
available immediately after the first apply; check for a completed snapshot in EC2.

These are disk snapshots, not PostgreSQL logical backups. To test recovery, create
a separate disk from a snapshot, attach it to a test server in the same Availability
Zone, and start a compatible PostgreSQL version using the recovered data. Never
format the recovered disk. Check the car records and API before trusting the backup.
Do not change or overwrite the live database during this test. One snapshot per day
can lose up to about one day of data; backup failures can make that gap longer.

### 5. Update existing servers on purpose

`user_data.sh` runs on first boot. Terraform ignores later changes to this field,
so editing the script does not stop a running server. To update an existing host,
review the script and run it during a planned maintenance period. It may restart
Docker. Deploy the app, its environment, and its migrations separately.

### 6. Files to keep safe

Do not edit state files or `.terraform.lock.hcl` by hand. Commit the lock file.
Keep state backups, saved plans, credentials, and `terraform.tfvars` out of Git.
Keep the Ubuntu image, CPU architecture, and bootstrap script compatible.

References: [Terraform local state](https://developer.hashicorp.com/terraform/language/backend/local)
and [AWS snapshot policies](https://docs.aws.amazon.com/ebs/latest/userguide/snapshot-lifecycle.html).

### 7. Start the application on EC2

Terraform installs Docker on the server; application startup is the next step.
SSH to the output IP with your private key, then wait for setup:

```bash
sudo cloud-init status --wait
docker info
```

Clone this repository on the server (or copy your checkout there). From its root,
create the environment file once; preserve it on subsequent deployments:

```bash
cp -n deployment/docker/.env.example deployment/docker/.env
openssl rand -hex 24
```

Edit `deployment/docker/.env`: replace `POSTGRES_PASSWORD` with the generated hex
value. Keep `POSTGRES_USER=postgres` and `POSTGRES_DB=cars` for this setup. Use a
URL-safe password because the migration connection string is a URL. Changing this
file later does not change the password in an already initialized database.

Build and start from this checkout (works without a registry image):

```bash
APP_IMAGE=car-service:local docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --build --wait
curl --fail http://localhost:8080/health
```

PostgreSQL starts first, migrations create/update the tables, and then the app
starts. Compose sets `POSTGRES_HOST=postgres` and `REDIS_ADDR=redis:6379` automatically.
The database and Redis have no published host ports. On your laptop, access
`http://EC2_PUBLIC_IP:8080/health` from the IP allowed by `allowed_ssh_cidr`.

For updates, pull the intended code revision and rerun the same Compose command.
Keep the same directory/project name so Compose reuses the database volume.
`docker compose ... down` keeps data; adding `-v` deletes it.

### 8. CI images and troubleshooting

Deployment fixes: local Terraform state removes the S3 requirement; production
Compose runs migrations before the app, overrides container connection addresses,
and supports building locally. Both Docker healthchecks use GET (the old HEAD
request returned 405 and incorrectly marked the app unhealthy).

The workflow now lives in `.github/workflows/ci.yml`. Pull requests run Go checks;
pushes to `main` also publish an amd64 image to `ghcr.io/<owner>/<repository>`, tagged
with `latest` and the commit SHA. This only runs after the changes are pushed to GitHub.

To avoid building on the small EC2 instance, set `APP_IMAGE` in the production
`.env` to the published image with its commit SHA. If the package is private,
authenticate on the server using `docker login ghcr.io` with a token that can read
the package (or make the package public). Then, from the matching code checkout:

```bash
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml pull
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --no-build --wait
```

When startup fails, inspect the app and migration output:

```bash
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml ps -a
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml logs --tail=100 app migrate postgres
```

If `init` asks for an S3 bucket, you are using the old configuration. If `plan`
asks for variables, finish `terraform.tfvars`. If Docker is unavailable on EC2,
check `sudo tail -100 /var/log/user-data.log`.

Local checks from the repository root:

```bash
go test ./...
go vet ./...
terraform -chdir=deployment/terraform validate
terraform -chdir=deployment/terraform test
# With the application running on port 8080:
go test -tags=integration ./test -count=1
```

Verified locally: Terraform initialization and validation, all six mocked Terraform
tests, Go tests/vet, fresh PostgreSQL migrations, healthy production containers,
and HTTP CRUD/filter integration tests. A real EC2 apply and GitHub image publication
have not been run.

To reproduce the isolated production check on port 18080 (after creating `.env` above):

```bash
APP_IMAGE=car-service:check APP_HOST_PORT=18080 APP_BIND_ADDRESS=127.0.0.1 \
  docker compose -p car-service-check --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --build --wait
CAR_SERVICE_BASE_URL=http://127.0.0.1:18080 go test -tags=integration ./test -count=1
curl --fail http://127.0.0.1:18080/health
# Stop this check while keeping its database:
docker compose -p car-service-check --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml down
```

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

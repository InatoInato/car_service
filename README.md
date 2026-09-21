# Car Service

A REST API for managing cars, written in Go.

The business logic is deliberately boring — the point of this project is everything around it: clean architecture, SQL-first development, migrations, Docker, testing, CI, and deploying the thing to real infrastructure.

## Tech Stack

| Technology | Purpose |
|------------|---------|
| Go | Backend |
| Chi | HTTP router |
| PostgreSQL | Database |
| pgx | Postgres driver |
| sqlc | Type-safe SQL generation |
| golang-migrate | Migrations |
| Redis | Caching |
| Docker | Containerization |
| Terraform | Infrastructure as code |
| GitHub Actions | CI |

## Quick Start

```bash
git clone https://github.com/InatoInato/car_service.git
cd car_service
cp .env.example .env
docker compose up --build
```

The API comes up on `http://localhost:8080`.

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/cars \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "BMW",
    "model": "X5",
    "production_year": 2023,
    "color": "Blue",
    "price": 42000
  }'
```

Swagger UI is at [`/swagger/index.html`](http://localhost:8080/swagger/index.html).

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/cars` | List and filter cars |
| GET | `/cars/{id}` | Get car by ID |
| POST | `/cars` | Create car |
| GET | `/cars/generations` | Optional, advisory generation suggestions |
| PUT | `/cars/{id}` | Update car |
| DELETE | `/cars/{id}` | Delete car |

### Optional listing details

Cars now include nullable `image`, `model_generation`, and `description` fields.
Old create requests still work. On PUT, omitted optional fields stay unchanged;
explicit `null`, empty, or whitespace-only values clear them. Responses include
all three keys as strings or JSON `null`. Core PUT fields still use the existing
replacement behavior.

`image` is an absolute HTTP(S) URL (not an upload, binary data or filesystem path).
Limits: image 2,048 characters, generation 100, description 10,000; request body
128 KiB. Descriptions are plain text: frontends must escape them, not render HTML.
The backend does not fetch image URLs or verify that they point to a real image.

```sh
curl --get http://localhost:8080/cars/generations \
  --data-urlencode 'brand=Mercedes-Benz' \
  --data-urlencode 'model=E280' \
  --data-urlencode 'year=1995'
```

This returns both W124 facelift and early W210 suggestions, source URLs, body style,
and a coverage notice. It never selects a generation or blocks car creation.
The initial local catalogue covers **only E280 saloons, 1993–1999**; other makes,
models and years can legitimately return `{"candidates":[],"notice":"..."}`.
No internet request or credential is needed.

See [design, API examples, data sources and engineering review](docs/listing-enrichment.md).

### Predictable development commands

```sh
make help
make test                # all Docker-free tests
make test-http           # HTTP status/header/body-limit contracts, no Docker
make race                # same tests with race detection
make lint                # gofmt check + go vet; no extra linter
make build               # bin/car_service
make test-load           # isolated DB + Redis, bounded load scenarios
make test-integration    # isolated DB + Redis; all integration/load tests with -race
make test-down           # stop only the isolated test stack
```

`make generate` pins sqlc v1.31.1; `make docs` pins swag v1.8.1. They use `go run`
and require network access on the first run. Generated files are committed.
`make run` runs the API against the database configured in `.env`; it does not
silently migrate that database.

Upgrade an existing local Compose stack (back up important data first):

```sh
docker compose stop car_service
make migrate-up
docker compose up -d --no-deps --build car_service
```

The schema must be upgraded before the new binary runs. Migration 000002 only adds
nullable columns; it does not rewrite 000001 or remove existing cars. The new binary
uses `cars:v2:` Redis keys, leaving old keys to expire without flushing Redis.
Do not run old and new app versions together: their cache namespaces differ.

`make migrate-down CONFIRM=drop-listing-details` rolls back **one** migration in the
local Compose database. This deletes optional-field data. Stop the new app first;
restore/run a compatible old binary after rollback. Do not roll back production
without a backup and an explicit recovery plan.

### Filtering

`GET /cars` takes these optional query parameters:

| Parameter | Meaning |
|-----------|---------|
| `name` | Case-insensitive partial match on brand, model, or both |
| `year` | Exact production year (`production_year` also works) |
| `created_from`, `created_to` | Inclusive RFC3339 creation range |
| `created_at` | Exact creation time, RFC3339 |
| `min_price`, `max_price` | Inclusive price range |
| `price` | Exact price |
| `page`, `limit` | Pagination; `limit` is 1–100 |

```bash
curl "http://localhost:8080/cars?name=BMW&year=2023&min_price=30000&max_price=60000"
```

## Project Structure

```text
.
├── cmd/car_service/       # entrypoint
├── database/
│   ├── migrations/
│   └── queries/           # sqlc source
├── deployment/
│   ├── docker/
│   └── terraform/
├── internal/
│   ├── config/
│   ├── db/
│   ├── handler/
│   ├── middleware/
│   ├── service/
│   └── router.go
├── test/
├── Dockerfile
├── docker-compose.yml
└── sqlc.yaml
```

Request flow:

```
HTTP → Chi Router → Handler → Service → sqlc Queries → PostgreSQL
```

## Testing

Unit and handler tests. No Docker needed:

```bash
go test -race -count=1 -timeout=2m ./...
```

The handler tests run the real router and service against a stub database, covering
success paths, field mapping, malformed JSON, bad IDs and filters, 404s, DB failures,
error response bodies, and context propagation. They don't add validation rules of
their own.

Integration tests default to the separate test stack, not your development API:

```bash
make test-integration
make test-down
```

The suite starts the real application in-process, so the race detector covers
handlers/services as well as tests. It checks CRUD, optional fields, migrations,
Redis, concurrency and bounded load. Migration tests create a transaction-local
schema and roll it back; they never roll back your application database.
The old external-API smoke test can still be selected explicitly with
`CAR_SERVICE_BASE_URL=http://... go test -tags=integration ./test -run '^TestIntegrationCarsCRUDAndFilters$'`.
Never point write tests at production.

### Concurrency and load

These use a separate stack on `127.0.0.1:15432` (Postgres) and `127.0.0.1:16379`
(Redis) with its own database — your dev data is safe. Keep those ports free.

```bash
docker compose -p car-service-tests -f deployment/docker/docker-compose.test.yml \
  up -d --wait --wait-timeout 120 postgres redis
docker compose -p car-service-tests -f deployment/docker/docker-compose.test.yml \
  run --rm migrate

# Concurrent handlers/service/Postgres/Redis under the race detector
go test -race -tags=integration,load ./test -run '^TestConcurrent' -count=1 -timeout=2m -v

# Load, without race-detector overhead
make test-load

# Bigger local sample
LOAD_WORKERS=16 LOAD_ITERATIONS=100 \
  make test-load

# Tear down (keeps volumes)
docker compose -p car-service-tests -f deployment/docker/docker-compose.test.yml down
```

The load harness starts the real router in-process on an ephemeral port, so the race
detector instruments server code. `CAR_SERVICE_BASE_URL` is ignored here. Each worker
owns its own car and cleans up only its own fixtures.

The default run has two scenarios: 800 mixed update/read/list requests and 200
concurrent uncached list requests. The second scenario pressures the PostgreSQL
connection pool instead of measuring Redis alone. Setup and teardown happen outside
the measurement. Each result line includes the scenario, request count, errors,
throughput, and p50/p95/p99. Any HTTP or correctness error fails the test.

Two caveats worth knowing:

- These tests don't promise cache linearizability. There's a known stale-cache
  interleaving that still needs a real fix plus a regression test.
- No latency threshold is enforced, because shared CI runners are noisy. Set a baseline
  on fixed hardware first. This is a smoke test, not a capacity or soak test.

Lint:

```bash
gofmt -l .
go vet ./...
```

## CI

Every push and PR runs gofmt, `go vet`, race-enabled unit/handler tests, the concurrent
API tests, the default load smoke test, then builds the binary and the Docker image.

Pushes to `main` additionally publish an amd64 image to `ghcr.io/<owner>/<repository>`,
tagged `latest` and with the commit SHA. The workflow lives in `.github/workflows/ci.yml`.

## Deployment

The AWS setup is one small EC2 instance. SSH and port 8080 are open only to a single IP
you choose. **The API has no authentication**, so keep that rule narrow.

### 1. Variables

Terraform 1.10+. Create `deployment/terraform/terraform.tfvars`:

```hcl
aws_region       = "us-east-1"
instance_type    = "t2.micro"
ssh_public_key   = "YOUR SSH PUBLIC KEY"
ami_id           = "ami-REPLACE_ME"
subnet_id        = "subnet-REPLACE_ME"
allowed_ssh_cidr = "YOUR.PUBLIC.IP/32"
```

Never put a private key in here.

For an **existing** instance, copy its current AMI and subnet ID from the EC2 console —
guessing here triggers a replacement. For a **new** one, pick a Canonical Ubuntu 22.04
amd64 image and a public subnet with an Internet Gateway route. The security group is
created in that subnet's VPC.

`allowed_ssh_cidr` must be your real public IPv4 with `/32`. Terraform no longer detects
it automatically, so update it yourself when your IP changes.

These values are account-specific, which is why there are no defaults. Start from
`deployment/terraform/terraform.tfvars.example`.

### 2. Init

State is local, in `deployment/terraform/terraform.tfstate` — no S3 bucket needed. Keep a
private backup, always run from the same checkout, and don't run concurrent applies from
another machine.

```bash
terraform -chdir=deployment/terraform init
aws sts get-caller-identity   # confirm the right account
```

Migrating from a working remote backend? Back it up and use `init -migrate-state`. Don't
use `-reconfigure` to skip the migration.

### 3. Plan, then apply

```bash
terraform -chdir=deployment/terraform validate
terraform -chdir=deployment/terraform test
terraform -chdir=deployment/terraform plan -out=deploy.tfplan
terraform -chdir=deployment/terraform apply deploy.tfplan
```

The tests run against a fake AWS provider — nothing is created and no live server is
touched.

Read the plan before applying. Check the account, region, IP rule, and anything touching
the instance or its volume. **If the plan wants to replace the server, stop** and recheck
your AMI and subnet IDs. Don't delete `prevent_destroy` to make it go through.

Besides the existing resources, this creates a DLM snapshot policy and the IAM role AWS
uses to run it. Your operator needs permission to create that role, attach its policy,
pass it to DLM, and create the DLM policy itself.

### 4. Backups

Protection is layered: Terraform blocks replacement, AWS termination protection blocks
accidental terminate calls, and the root volume is retained if you ever remove both and
terminate anyway. Retaining a volume doesn't attach it to anything automatically, and
none of this stops a manual volume deletion.

DLM snapshots the volume tagged `Backup = "car-service-daily"` every day at 03:00 UTC and
keeps the last seven. Use that tag only for this stack. Snapshots and retained volumes
cost money. The first snapshot won't exist immediately after apply — check EC2 for a
completed one.

These are **disk** snapshots, not logical Postgres backups. Daily snapshots mean up to
~24h of data loss, more if a backup silently fails. To actually test recovery: create a
new volume from a snapshot, attach it to a test instance in the same AZ, start a
compatible Postgres version on it, and check the car records through the API. Never
format the recovered volume, and never point the test at the live database.

### 5. Updating an existing instance

`user_data.sh` only runs on first boot, and Terraform ignores later changes to it. Editing
the script does nothing to a running host. To apply changes, review the script and run it
during a maintenance window — it may restart Docker. App code, environment, and migrations
are deployed separately.

### 6. Running the app on EC2

Terraform installs Docker; starting the app is a separate step. SSH in with your private
key:

```bash
sudo cloud-init status --wait
docker info
```

Clone the repo (or copy your checkout) and create the env file once:

```bash
cp -n deployment/docker/.env.example deployment/docker/.env
openssl rand -hex 24
```

Put that hex value in `POSTGRES_PASSWORD`. Keep `POSTGRES_USER=postgres` and
`POSTGRES_DB=cars`. Use a URL-safe password — the migration connection string is a URL.
Changing this file later won't change the password of an already-initialized database.

Build and start:

```bash
APP_IMAGE=car-service:local docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --build --wait
curl --fail http://localhost:8080/health
```

Postgres starts first, migrations run, then the app. Compose sets `POSTGRES_HOST=postgres`
and `REDIS_ADDR=redis:6379` for you. Neither Postgres nor Redis publishes a host port. From
your laptop, hit `http://EC2_PUBLIC_IP:8080/health` from the IP in `allowed_ssh_cidr`.

To deploy an update, pull the revision you want and rerun the same command. Keep the same
project/directory name so Compose reuses the database volume. `down` keeps your data;
`down -v` deletes it.

To avoid building on a `t2.micro`, set `APP_IMAGE` in the production `.env` to the
published image at a specific commit SHA, then:

```bash
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml pull
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --no-build --wait
```

If the GHCR package is private, `docker login ghcr.io` on the server with a read-capable
token, or make the package public.

### Troubleshooting

| Symptom | Cause |
|---------|-------|
| `init` asks for an S3 bucket | You're on the old configuration |
| `plan` prompts for variables | `terraform.tfvars` is incomplete |
| Docker missing on EC2 | Check `sudo tail -100 /var/log/user-data.log` |
| App marked unhealthy | Both healthchecks use GET now; HEAD used to return 405 |

Container logs:

```bash
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml ps -a
docker compose --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml logs --tail=100 app migrate postgres
```

### What's actually been verified

Locally: Terraform init/validate, all six mocked Terraform tests, Go tests and vet, a fresh
migration run, healthy production containers, and the HTTP CRUD/filter integration tests.

Not yet done: a real EC2 apply, and publishing an image from GitHub Actions.

Isolated production check on port 18080:

```bash
APP_IMAGE=car-service:check APP_HOST_PORT=18080 APP_BIND_ADDRESS=127.0.0.1 \
  docker compose -p car-service-check --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml up -d --build --wait
CAR_SERVICE_BASE_URL=http://127.0.0.1:18080 go test -tags=integration ./test -count=1
curl --fail http://127.0.0.1:18080/health

docker compose -p car-service-check --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml down
```

## Regenerating code

sqlc after touching `database/queries/`:

```bash
sqlc generate
```

Swagger after adding annotations to a handler:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.8.1 init -g cmd/car_service/main.go -o docs
```

## Roadmap

**Done**

- REST API, PostgreSQL, sqlc, migrations
- Structured logging, graceful shutdown, config validation, request ID middleware
- Docker, Compose, container healthchecks
- Unit, integration and load tests, GitHub Actions CI
- Redis cache, pagination, OpenAPI/Swagger
- Terraform infrastructure

**Next**

- [ ] Deploy to AWS EC2
- [ ] GitHub Actions CD
- [ ] Prometheus metrics at `/metrics`
- [ ] Kubernetes

## License

Educational and portfolio use.

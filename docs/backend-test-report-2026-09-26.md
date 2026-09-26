# Backend test report — 2026-09-26

Tested commit `bc60615` plus the working-tree pagination fix and its new
regression test. This report records observations and proposed fixes; no
application changes were made during this audit.

All database writes used the isolated `car-service-tests` PostgreSQL instance.
Live HTTP probes used the newly built Docker image on `127.0.0.1:18082`.
No AWS deployment or remote CI run was tested.

## Verification log

```text
make lint                                      PASS
go build ./...                                 PASS
git diff --check                               PASS
make test-integration                          PASS
  all packages, -race, integration and load tags
  CRUD, filters, exact prices, optional fields, validation
  generation candidates, provider failures, concurrent operations
  migration up/down/up and equal-timestamp pagination
docker compose config --quiet                  PASS
docker compose build car_service               PASS
Docker image health check                      healthy
bash -n deployment/terraform/user_data.sh       PASS
terraform -chdir=deployment/terraform fmt -check PASS
```

Fourteen additional checks against the running Docker app passed: health;
zero/one/two generation candidates; missing generation parameters; creating
and retrieving all listing fields; combined name/year/price/exact-creation
filtering; preserving omitted optional values; clearing null/empty values;
negative-price rejection; unsafe-image-scheme rejection; deletion; and 404
after deletion. Five live pages over the 20,000-row catalog returned 100 unique
cars with the correct total. The existing regression also tested tied timestamps.

Database outage and recovery:

```text
PostgreSQL stopped:  /health 503, /cars 500, /cars/generations 200
PostgreSQL restored: /health 200, /cars 200
```

## 1. High priority: an update can commit after its response expires

Location: `cmd/car_service/main.go:86`, `internal/handler/car.go:346`.

The server has a ten-second HTTP write timeout. CRUD database calls inherit the
request context but have no explicit operation deadline. To reproduce, create
one disposable car and hold its row lock in a separate database session:

```sql
BEGIN;
SELECT id FROM cars WHERE id = '<test-car-id>' FOR UPDATE;
SELECT pg_sleep(13);
ROLLBACK;
```

While the lock was held, a PUT changed that car's price from 100 to 200:

```text
UPDATE client error=RemoteDisconnected
Remote end closed connection without response
elapsed=13.026 seconds
STORED PRICE after client result=200.0
server log: method=PUT status=200 duration_ms=13023
```

The update waited for the lock, committed, and failed to deliver a response.
The server log's attempted 200 response did not mean the client received it.
The price was confirmed by a subsequent GET. The fixture was then deleted.

Impact: users see failures after successful changes; retries become ambiguous.
Long lock waits also occupy database connections and can make other requests wait.

Proposed fix: give CRUD database work an explicit context deadline shorter than
the HTTP write timeout. Configure PostgreSQL lock/statement limits as appropriate
for the chosen request budget, and return a controlled timeout response while
there is still time to write it. Add a regression that holds a row lock beyond
that budget and verifies prompt failure and an unchanged price. Track timeout
errors explicitly; response status logs alone cannot prove delivery.

## 2. High priority during credential rotation: valid passwords break startup

Location: `cmd/car_service/main.go:56`.

A disposable role with a synthetic password containing `/`, `?`, and `#`
successfully authenticated through `psql`. The application failed with the same
credentials before it could connect:

```text
psql authentication: PASS (current_user=audit_password_probe)
application startup: exit 1
failed to parse as URL (invalid port ... after host)
```

Cause: raw credentials are inserted into a PostgreSQL URL using `fmt.Sprintf`.
Reserved URL characters change the meaning of the connection string.

Proposed fix: build the URL with `net/url` and `url.UserPassword`, or populate
typed pgx configuration fields. Test passwords containing URL-reserved characters.
The production migration command also interpolates raw credentials into a URL
(`deployment/docker/docker-compose.prod.yml:37`); address that path in the same
change. Its equivalent failure was identified by inspection, not a migration run
with the special password. The disposable role was removed.

## 3. Medium priority: invalid search input reaches PostgreSQL or loses its filter

Location: `internal/handler/car.go:166` and `internal/handler/car.go:191`.

```text
GET /cars?name=%00              500 (NUL in PostgreSQL text)
GET /cars?name=%FF              500 (invalid UTF-8, SQLSTATE 22021)
GET /cars?name=%ZZ              200 (malformed filter silently omitted)
GET /cars?year=2024&year=bad     200 (first duplicate value used)
GET /cars?min_price=2&max_price=1 400 (correctly rejected)
```

The first two failures are client input errors that become server errors. The
third request returns an unfiltered catalog rather than explaining the bad input.
Duplicate handling is also inconsistent with the generation endpoint, which
rejects repeated required parameters.

Proposed fix: use `url.ParseQuery(r.URL.RawQuery)` and handle its error. Validate
search text for UTF-8, NUL, and a documented length bound before persistence.
Reject conflicting or repeated single-value parameters consistently. Return 400
with a useful message, and test that invalid input never calls the database.

## 4. Medium priority: search becomes the bottleneck as the catalog grows

The load tests use a real HTTP server inside the Go test process and Docker
PostgreSQL. They discard application logs and run without race instrumentation
for these latency measurements. They are short, closed-loop local measurements,
not production capacity guarantees or a sustained-load test.

The larger fixture had 20,000 rows spread across 20 brands, 200 model labels,
35 years, varied prices/timestamps, and short descriptions. `ANALYZE cars` ran
before measurement. Test workers search for their own small result sets among
these rows.

| Catalog | Scenario | Workers | Requests | Errors | p95 | p99 | Requests/sec |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Small | Mixed update/read/list | 8 | 1,600 | 0 | 2.44 ms | 6.04 ms | 6,576 |
| Small | Filtered list | 8 | 400 | 0 | 1.51 ms | 8.93 ms | 6,307 |
| 20,000 | Mixed update/read/list | 8 | 1,600 | 0 | 44.16 ms | 52.22 ms | 660 |
| 20,000 | Filtered list | 8 | 400 | 0 | 96.16 ms | 146.77 ms | 106.5 |
| 20,000 | Filtered list | 32 | 800 | 0 | 350.42 ms | 379.01 ms | 106.7 |

Increasing concurrency produced longer waits with almost no throughput gain.
`FilterCars` and `CountFilteredCars` both scan rows and evaluate multiple
lowercase substring expressions. The pagination tie breaker fixes correctness;
it does not remove those scans.

Commands used after `make test-up`:

```sh
CAR_SERVICE_BASE_URL= LOAD_WORKERS=8 LOAD_ITERATIONS=50 \
  go test -tags=integration,load ./test -run '^TestLoad' -count=1 -timeout=2m -v
CAR_SERVICE_BASE_URL= LOAD_WORKERS=32 LOAD_ITERATIONS=25 \
  go test -tags=integration,load ./test -run '^TestLoadListBurst$' -count=1 -timeout=2m -v
```

A reversible SQL experiment evaluated a possible fix. The existing name predicate
was compared with this indexed predicate, using `model173` (100 matching rows):

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX audit_name_trgm ON cars
USING gin (lower(brand || ' ' || model) gin_trgm_ops);
-- Candidate predicate:
-- lower(brand || ' ' || model) LIKE '%model173%'
```

`EXPLAIN (ANALYZE, BUFFERS)` results:

```text
Current page query:     sequential scan, 20.782 ms
Current count query:    sequential scan, 19.364 ms
Candidate page query:   bitmap index scan, 0.524 ms
Candidate count query:  bitmap index scan, 0.488 ms
ROLLBACK
temporary_indexes_left=0
```

This is a single SQL-plan comparison, not an optimized HTTP benchmark. Both
queries found 100 matches. The candidate needs a migration and matching query
changes, with literal `%`, `_`, and backslash escaped to preserve today's
substring-search semantics. One- and two-character searches or very common terms
can still require substantial scanning. The index also costs space and write
work. Measure those cases before adopting it. Extension and index creation were
rolled back; neither was added to the application schema.

## Recommended order

1. Bound database operation/lock waits and test the failed-response/committed-write case.
2. Correct credential encoding in the application and migration startup paths.
3. Reject malformed query parameters before database access.
4. Implement and benchmark the indexed search candidate with a forward migration.

The full existing test suite passed, but the three new failure probes above
expose missing regression coverage. None is evidence that race detection failed:
database waits, URL parsing, and malformed input are different failure classes.

Cleanup: all 20,000 catalog fixtures and individual HTTP/lock fixtures were
removed. The disposable role was dropped. The experimental index and extension
were rolled back. The temporary API container and dedicated test stack were
removed. OrbStack was already running at the start and was left running.

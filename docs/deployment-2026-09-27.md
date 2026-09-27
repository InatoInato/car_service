# Deployment — 2026-09-27

The service is running on EC2 at http://54.85.73.54:8080. SSH and the API allow
only 95.46.137.23/32. A changed home/public IP requires updating
`allowed_ssh_cidr` and applying the reviewed Terraform plan.

## Scope

- Encoded application database credentials; migration credentials use PostgreSQL
  environment variables, avoiding raw password interpolation into a URL.
- Malformed query encoding, invalid text, duplicate parameters and conflicting
  aliases return 400 before SQL. Search names are limited to 201 characters.
- Set an explicit 20 GiB gp3 root disk and added a Docker build check to PR CI.
- Kept the current one-service architecture and IP-restricted access.

Search indexing is deferred for this small deployment. The earlier large-catalog
benchmark remains relevant if the catalog grows. No new application dependencies.

## Verification log

```text
Go race suite, vet, formatting, build                 PASS
PostgreSQL integration/concurrency/load suite          PASS
Terraform validate and mocked tests                   PASS (6 tests)
amd64 Docker image build                              PASS
Production Compose migrations with special password   PASS
Production image CRUD and optional-field tests         PASS
Terraform apply                                       7 added, 0 changed, 0 destroyed
EC2 cloud-init                                        done
EC2 system / instance status                          ok / ok
App / PostgreSQL containers                           healthy / healthy
Live /health                                         200
Live create/read/filter/update/delete                  PASS
Live generation lookup for E280 1995                   2 candidates
Live malformed filters                                400
Temporary live test car                               deleted
Root disk after deploy                                20 GiB, about 16 GiB free
Available memory after deploy                         about 507 MiB
Daily snapshot policy                                 ENABLED
```

The special test password included `/`, `?`, `#`, `@`, `:` and `$`. The isolated
test stacks and synthetic database were removed. Existing local development
containers were left running.

## Release and resources

- Region: `us-east-1`; instance: `i-03f9d9d110b3d6cde`, `t2.micro`.
- Elastic IP: `54.85.73.54`; root volume: `vol-0ab4ec369b3b20d86`.
- Snapshot policy: `policy-04971c24d7a482ba3` (daily, seven retained).
- Image: `car-service:release-20260927`, built locally for `linux/amd64` and
  transferred to EC2. Source: commit `9d07dca` plus this working-tree fix set.
- Running image identity: `sha256:c1bd204c2042f3f4311c82d6ee93a5d2e8de48681149c5a0b74b3d98aeec9825`.
- Application files: `/home/ubuntu/car_service`; Compose project: `docker`.
- Database volume: `docker_postgres_data`. Keep the project name stable.
- Server credentials: `/home/ubuntu/car_service/deployment/docker/.env`, mode 600.
  The database password was generated on the server and is not in Git.
- Terraform state and chosen deployment variables remain local and ignored by Git.
  Keep a private backup of the state.

## Operating commands

```sh
curl --fail http://54.85.73.54:8080/health
ssh -i ~/.ssh/aws-lab ubuntu@54.85.73.54

# On EC2:
cd /home/ubuntu/car_service
docker compose -p docker --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml ps
docker compose -p docker --env-file deployment/docker/.env \
  -f deployment/docker/docker-compose.prod.yml logs --tail=100 app postgres
df -h /
free -m
```

For later releases, use an immutable image tag and run migrations before starting
the new app. The CI image check is implemented but has not yet run on GitHub.
The changes in this release have not been committed or pushed by the assistant.

This is one server with daily snapshots. No restore drill or sustained AWS load
test has been performed; the first scheduled snapshot is not immediate. EC2, the
Elastic IP, disk and snapshots are billable while retained.

export PATH := $(PATH):$(HOME)/go/bin

-include .env
export

DB_URL=$(DATABASE_URL)

# --- Docker ---

up:
	docker compose up --build

down:
	docker compose down

reset:
	docker compose down -v
	docker compose up --build

# --- Observability (Prometheus + Grafana + Loki + Tempo + exporters) ---
# Start the monitoring profile alongside the running app. Grafana at :3001
# (admin/admin), Prometheus at :9090. Tracing: set OTEL_ENABLED=true in .env
# and restart the api. See docs/observability.md.

monitoring-up:
	docker compose --profile monitoring up -d

monitoring-down:
	docker compose --profile monitoring down

# Backfill ~60 days of users/orders/reviews so the business dashboards have
# history. Idempotent; run once after the app is up.
seed-history:
	go run ./cmd/seedhistory

# Live technical + security traffic (failed logins, rate-limit bursts, forged
# webhooks). Run against the default stack, not the loadtest override.
hostile:
	docker run --rm --network host -v "$(PWD)/loadtest:/s" grafana/k6:0.57.0 run /s/hostile.js

# Baseline browse/search/checkout load (for the technical + cAdvisor panels).
loadtest:
	docker run --rm --network host -v "$(PWD)/loadtest:/s" grafana/k6:0.57.0 run /s/load.js

# --- Local development ---

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -count=1

# Generate thumbnail/card/full variants for seeded product images that only
# have a legacy url. Run after the catalogue is seeded.
image-variants:
	go run ./cmd/imagevariants

# --- Observability (see docs/observability.md) ---

# Start/stop the app together with Prometheus, Grafana, Loki, Promtail,
# cAdvisor, and node-exporter. Grafana: http://localhost:3001 (admin

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

migrate-force:
	migrate -path migrations -database "$(DB_URL)" force $(version)

db:
	docker exec -it my-postgres psql -U admin -d mystore

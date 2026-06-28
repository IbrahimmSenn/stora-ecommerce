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

# --- Migrations (local) ---

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

migrate-force:
	migrate -path migrations -database "$(DB_URL)" force $(version)

db:
	docker exec -it my-postgres psql -U admin -d mystore

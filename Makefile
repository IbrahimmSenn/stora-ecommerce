export PATH := $(PATH):$(HOME)/go/bin

include .env
export

DB_URL=$(DATABASE_URL)

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

migrate-force:
	migrate -path migrations -database "$(DB_URL)" force $(version)
	
db:
	docker exec -it my-postgres psql -U admin -d mystore
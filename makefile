local-db := postgres://postgres:secret@localhost:5432/postgres?sslmode=disable
template-db := ${DB_PROVIDER}://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}

migrate-local-up:
	migrate -path ./migrations -database $(local-db) -verbose up

migrate-local-down:
	migrate -path ./migrations -database $(local-db) -verbose down -all

migrate-up:
	migrate -path ./migrations -database $(template-db) -verbose up

migrate-down:
	migrate -path ./migrations -database $(template-db) -verbose down -all

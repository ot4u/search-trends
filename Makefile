.PHONY: run test bench lint docker-up docker-down

run:
	go run ./cmd/app

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./internal/...

lint:
	golangci-lint run

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

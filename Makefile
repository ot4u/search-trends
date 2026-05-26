.PHONY: run test bench lint load-test docker-up docker-down

run:
	go run ./cmd/app

test:
	go test ./...

bench:
	go test -bench=. -benchmem -benchtime=100000x ./internal/...
	
lint:
	golangci-lint run

load-test:
	bash tests/load/run-all.sh

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

format:
	go fmt ./...

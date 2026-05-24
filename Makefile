up:
	docker compose up --build -d

install:
	go mod tidy

stop:
	docker compose stop

down:
	docker compose down

test-unit:
	go test -v -cover ./internal/usecase/...

format:
	go fmt ./...

lint:
	golangci-lint run --fix
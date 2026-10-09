.PHONY: all build run test clean docker-build docker-up docker-down

BINARY_NAME=bot
BUILD_DIR=bin

all: build

build:
	@echo "Building StdQuizBot binary..."
	go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bot

run:
	@echo "Running StdQuizBot locally..."
	go run ./cmd/bot

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)

docker-build:
	docker build -t stdquizbot .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

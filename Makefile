.PHONY: run build tidy

run:
	go run ./cmd/gonews

build:
	go build -o bin/gonews ./cmd/gonews

tidy:
	go mod tidy

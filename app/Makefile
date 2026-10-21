APP_NAME := mssh
GO := go

.PHONY: build run

build:
	mkdir -p bin
	$(GO) build -o bin/$(APP_NAME) ./cmd/$(APP_NAME)

run:
	$(GO) run ./cmd/$(APP_NAME)

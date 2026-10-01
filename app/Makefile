BINARY := bin/mssh

.DEFAULT_GOAL := check

.PHONY: fmt test vet build check clean

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p $(dir $(BINARY))
	go build -o $(BINARY) ./cmd/mssh

check: fmt test vet build

clean:
	rm -f $(BINARY)

BINARY := bin/mssh

.DEFAULT_GOAL := check

.PHONY: fmt test vet build build-gui compile-gui check clean

fmt:
	go fmt ./...

test:
	go test -tags ci ./...

vet:
	go vet -tags ci ./...

build:
	mkdir -p $(dir $(BINARY))
	go build -o $(BINARY) ./cmd/mssh

build-gui:
	mkdir -p $(dir $(BINARY))
	go build -o bin/mssh-gui ./cmd/mssh-gui

compile-gui:
	go build -tags ci -o /tmp/mssh-gui-ci ./cmd/mssh-gui

check: fmt test vet build compile-gui

clean:
	rm -f $(BINARY) bin/mssh-gui

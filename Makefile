BINARY ?= stol
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

build:
	go build -o bin/$(BINARY) .

lint:
	golangci-lint run ./...

test: lint
	go test -race ./...

install: build
	install -Dm755 bin/$(BINARY) $(BINDIR)/$(BINARY)

clean:
	rm -rf bin

.PHONY: build lint test install clean

MODULE := github.com/SZabrodskii/gophkeeper-stas
VERSION ?= dev
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X $(MODULE)/pkg/buildinfo.Version=$(VERSION) \
           -X $(MODULE)/pkg/buildinfo.Date=$(DATE) \
           -X $(MODULE)/pkg/buildinfo.Commit=$(COMMIT)

.PHONY: build build-server build-client test lint swag clean

build: build-server build-client

build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/gophkeeper ./cmd/client

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

swag:
	swag init -g cmd/server/main.go -o docs

clean:
	rm -rf bin/ coverage.out

tls-cert:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
		-out certs/server.crt -days 365 -nodes -subj '/CN=localhost'

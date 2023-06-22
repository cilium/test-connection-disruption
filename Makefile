GO ?= go

all: client server

client: cmd/client
	CGO_ENABLED=0 $(GO) build -a -ldflags '-extldflags "-static"' ./cmd/client

server: cmd/server
	CGO_ENABLED=0 $(GO) build -a -ldflags '-extldflags "-static"' ./cmd/server

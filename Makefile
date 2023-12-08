GO ?= go
TAG ?= v0.0.10
IMAGE ?= quay.io/cilium/test-connection-disruption

all: client server

client: cmd/client
	CGO_ENABLED=0 $(GO) build -a -ldflags '-extldflags "-static"' ./cmd/client

server: cmd/server
	CGO_ENABLED=0 $(GO) build -a -ldflags '-extldflags "-static"' ./cmd/server

.PHONY: image

image: client server
	docker build --tag $(IMAGE):$(TAG) .

.PHONY: publish

publish:
	@docker buildx create --use --name=crossplatform --node=crossplatform && \
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--output "type=image,push=true" \
		--tag $(IMAGE):$(TAG) .

.PHONY: build test fmt smoke-live

build:
	go build ./cmd/mo

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

smoke-live:
	./scripts/smoke-live.sh

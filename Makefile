.PHONY: build test fmt smoke-live web-serve

build:
	go build ./cmd/mo

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

smoke-live:
	./scripts/smoke-live.sh

web-serve:
	cd web && python3 -m http.server 4173

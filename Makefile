.PHONY: build test fmt smoke-live docs-install docs-serve docs-build

build:
	go build ./cmd/mo

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

smoke-live:
	./scripts/smoke-live.sh

docs-install:
	pip install -r requirements-docs.txt

docs-serve:
	mkdocs serve

docs-build:
	mkdocs build --strict

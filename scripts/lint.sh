#!/bin/bash -e
# Run from directory above via ./scripts/lint.sh

$(exit $(go fmt ./... | wc -l))
go mod tidy
go vet ./...
misspell -error -locale US ./...

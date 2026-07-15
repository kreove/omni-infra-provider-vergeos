#!/usr/bin/env sh
set -eu

gofmt -w $(find . -name '*.go' -not -path './vendor/*')
go mod tidy
go test ./...
go build ./cmd/omni-infra-provider-vergeos

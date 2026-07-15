.PHONY: build test fmt validate docker

build:
	go build -o _out/omni-infra-provider-vergeos ./cmd/omni-infra-provider-vergeos

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

validate:
	./scripts/validate.sh

docker:
	docker build -t omni-infra-provider-vergeos:alpha .

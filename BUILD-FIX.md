# Alpha 2 build fix

The original alpha archive did not contain `go.sum`. The Docker build therefore
stopped with `missing go.sum entry` errors.

The Dockerfile now runs:

```sh
go mod tidy
go mod verify
```

before compiling the provider.

Build with:

```sh
docker build --pull --no-cache -t omni-infra-provider-vergeos:alpha2 .
```

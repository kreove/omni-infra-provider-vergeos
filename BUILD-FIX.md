# Container build

The source archive does not ship a generated `go.sum`. The Dockerfile resolves and verifies the module graph inside the build stage before testing and compiling:

```sh
go mod tidy
go mod verify
go test ./...
go build ...
```

Build with:

```sh
docker build --pull --no-cache \
  -t omni-infra-provider-vergeos:autoimage \
  .
```

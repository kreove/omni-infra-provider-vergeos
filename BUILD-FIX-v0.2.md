# Build fix v0.2

The project now uses the local Go module path `omni-infra-provider-vergeos`.
All internal imports use that same prefix, preventing `go mod tidy` from trying
to resolve provider packages from GitHub.

The Dockerfile also verifies that the local `meta` and `resources` packages are
present in the build context before resolving dependencies.

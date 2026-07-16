# Development and releases

## Local prerequisites

- Go version declared in `go.mod`
- Docker with BuildKit
- Access to an Omni test instance
- Access to a disposable VergeOS environment

## Build and test

```bash
go mod tidy
go mod verify
go test ./...
go build -o _out/omni-infra-provider-vergeos \
  ./cmd/omni-infra-provider-vergeos
```

Docker:

```bash
docker build --pull --no-cache \
  -t omni-infra-provider-vergeos:dev \
  .
```

Run help output:

```bash
docker run --rm omni-infra-provider-vergeos:dev --help
```

## Source layout

```text
cmd/github.com/kreove/omni-infra-provider-vergeos/    process startup and flags
internal/pkg/provider/provision.go  machine lifecycle reconciliation
internal/pkg/provider/image.go      Image Factory URL and cache handling
internal/pkg/provider/data/         Machine Class data and JSON schema
internal/pkg/provider/resources/    Omni/COSI provider state resource
api/specs/                          generated protobuf machine state
deploy/                             deployment examples
docs/                               operator and contributor documentation
```

## Changing Machine Class fields

Update both:

- `internal/pkg/provider/data/data.go`
- `internal/pkg/provider/data/schema.json`

Keep field names, types, defaults, and validation synchronized. Add tests to `helpers_test.go` or a new focused test file.

## Changing image behavior

Image identity and cache behavior live in `internal/pkg/provider/image.go`. Preserve these properties:

- Deterministic name for the same asset URL
- Different names for different schematics or Talos versions
- Validation against cache-name collisions
- Safe behavior during concurrent first-time requests
- Manual `image_file_id` override

## Live test checklist

Before publishing a release, validate:

1. Fresh provider registration in Omni
2. Automatic first image import
3. Reuse of the cached image
4. One-machine cluster creation
5. Three-control-plane cluster creation
6. Worker scale-up
7. Worker scale-down
8. Full deprovisioning without stale NICs, drives, or VMs
9. Provider restart with active machines
10. Omni restart with active machines
11. Invalid cluster ID failure
12. Invalid VNET ID failure
13. VergeOS permission failure
14. Failed Image Factory URL/import behavior
15. Talos version change
16. System extension change
17. Manual image override

## Release recommendations

- Use semantic version tags.
- Publish immutable container tags and digests.
- Generate an SBOM and provenance attestation when possible.
- Sign container images with Cosign.
- Document supported Omni and VergeOS versions in release notes.
- Never publish credentials in example files, logs, issues, or CI artifacts.

## Updating dependencies

Review changes in both Omni and the VergeOS SDK before updating:

```bash
go get github.com/siderolabs/omni/client@<version>
go get github.com/verge-io/govergeos@<version>
go mod tidy
go test ./...
```

Infrastructure-provider APIs may change between Omni releases. Treat dependency updates as compatibility changes and run the complete live test checklist.

# Validation

Checks performed in the generation environment:

- `gofmt` on all Go source files.
- JSON parsing of the MachineClass schema.
- YAML parsing of Docker Compose, Kubernetes, and GitHub Actions files.
- Shell syntax validation for validation scripts.
- Source scan confirming `image_file_id` is no longer required.
- Source scan confirming `ensureImage` precedes `syncMachine`.

A full dependency-linked Go build could not be run in the generation environment because outbound module downloads are unavailable. The Dockerfile performs `go mod tidy`, `go mod verify`, tests, and compilation in the build container.

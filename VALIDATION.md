# Validation report

Generated on 2026-07-14.

Passed in the generation workspace:

- All Go files parse and are formatted with `gofmt`.
- MachineClass JSON schema parses successfully.
- Docker Compose, Kubernetes, and GitHub Actions YAML parse successfully.
- Shell script syntax check passes.
- Basic secret-pattern scan found no embedded credentials.
- Provisioning calls were cross-checked against the official `govergeos` v0.3.0 VM, drive, NIC, cluster, network, and file service types.

Not completed in this workspace:

- `go test ./...` and a final linked binary build. The installed compiler is Go 1.23.2, while the reference Omni provider dependency set requires Go 1.26.2.

Observed local test result (exit code 1):

```text
go: go.mod requires go >= 1.26.2 (running go 1.23.2; GOTOOLCHAIN=local)
```

The included GitHub Actions workflow and Dockerfile use the required Go generation and are the intended dependency-level compile checks.

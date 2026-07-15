# Support

This is a community project. It is not covered by official Sidero Labs or Verge.io support unless those vendors explicitly state otherwise.

## Before requesting help

Review:

- [Installation](docs/installation.md)
- [Configuration reference](docs/configuration.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Compatibility and limitations](docs/compatibility.md)

## Bug reports

Include:

- Provider release tag or commit
- Omni version
- VergeOS version
- Talos version
- Deployment method: Docker Compose or Kubernetes
- Sanitized Machine Class provider data
- Sanitized provider logs
- Relevant VergeOS task/event output
- Expected and actual behavior
- Whether the issue occurs with automatic images or `image_file_id`

Do not post credentials, join tokens, private keys, or complete cloud-init data.

## Feature requests

Describe the operational problem, not only the proposed implementation. Useful requests include:

- Additional architectures
- Image cache garbage collection
- Private Image Factory authentication
- More VergeOS VM options
- Guest-agent support
- Metrics and health endpoints
- Existing-VM reconciliation

## Vendor product issues

Use the appropriate vendor support channel when the problem reproduces independently of this provider:

- Omni and Talos documentation: <https://docs.siderolabs.com/>
- VergeOS documentation: <https://docs.verge.io/>

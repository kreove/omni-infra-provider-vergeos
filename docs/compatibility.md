# Compatibility and limitations

## Release status

This provider is a community alpha. It has completed successful end-to-end lifecycle testing in a live Omni and VergeOS environment, including automatic image imports and scaling.

It has not been certified by Sidero Labs or Verge.io.

## Build-time dependencies

The current source tree declares:

- Go `1.26.2`
- Omni client `v1.8.0`
- VergeOS Go SDK `v0.3.0`

See `go.mod` for the authoritative dependency versions.

## Platform support

| Capability | Status |
| --- | --- |
| `amd64` | Supported |
| `arm64` | Not supported |
| UEFI | Supported and enabled by default |
| Secure Boot | Disabled |
| NoCloud join config | Supported |
| Automatic Image Factory import | Supported |
| Existing VergeOS file override | Supported |
| Multi-VNET Machine Classes | Supported |
| Multiple VergeOS clusters | Supported through Machine Class data |
| Multiple independent VergeOS clouds | Run separate provider IDs/instances |
| Distributed provider replicas | Not supported; run one replica per provider ID |
| Automatic cached-image garbage collection | Not implemented |
| Authenticated Image Factory headers/tokens | Not implemented |
| Existing VM CPU/RAM resize reconciliation | Not implemented |
| Existing VM VNET migration | Not implemented |
| VergeOS guest-agent integration | Unverified/disabled by current VM settings |

## Image import readiness

The VergeOS SDK resource used by the provider does not expose a dedicated import-completion state. The provider currently treats a file as ready when:

```text
file ID is valid AND filesize > 0
```

A failed import may leave a zero-size cache object that must be removed manually after confirming it is not referenced.

## Image cache lifecycle

Cached `omni-talos-*.qcow2` files persist after machines are deleted. This is intentional and improves subsequent provisioning speed. Operators must currently clean unused images manually.

## Existing-machine changes

Machine Class changes to the following fields affect newly created VMs only:

- CPU cores and CPU type
- Memory
- Disk size and interface
- VNET and NIC interface
- VergeOS cluster
- Machine type
- UEFI

To apply these changes, use an Omni rollout that provisions replacement machines and deprovisions the old ones.

## Omni and Talos compatibility

Select Talos and Kubernetes versions through Omni. Use versions supported by your Omni release and follow Omni's supported upgrade paths.

The provider uses APIs from the Omni client version declared in `go.mod`. A new Omni release may require rebuilding or updating the provider dependencies.

## VergeOS compatibility

No formal minimum VergeOS release is declared yet. The target VergeOS release must support the API operations used by `govergeos v0.3.0`, including:

- API-key bearer authentication
- VM create/read/delete and power operations
- VM drive and NIC lifecycle operations
- File lookup and server-side URL import
- NoCloud cloud-init files

Report the exact VergeOS release when filing compatibility issues.

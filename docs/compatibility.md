# Compatibility and limitations

## Release status

This provider is a community alpha. It has completed successful end-to-end lifecycle testing in a live Omni and VergeOS environment, including automatic image imports and scaling.

> [!IMPORTANT]
> That testing predates the move to Omni's installation media API. How the provider obtains an image, names the cache entry, and validates a cached file all changed, and **the image path has not been re-validated against a live VergeOS instance since**. VM creation, scaling and deprovisioning are unaffected by that change.

It has not been certified by Sidero Labs or Verge.io.

## Build-time dependencies

The current source tree declares:

- Go `1.26.7`
- Omni client `v1.11.0`
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
| Image Factory authenticated by token in the URL | Supported |
| Image Factory authenticated by request headers | Not supported; see below |
| Existing VM CPU/RAM resize reconciliation | Not implemented |
| Existing VM VNET migration | Not implemented |
| VergeOS guest-agent integration | Configurable with `guest_agent`; pair with the `siderolabs/qemu-guest-agent` extension |

## Image import readiness

The VergeOS SDK resource used by the provider does not expose a dedicated import-completion state. The provider currently treats a file as ready when:

```text
file ID is valid AND filesize > 0
```

A failed import may leave a zero-size cache object that must be removed manually after confirming it is not referenced.

## Image Factory authentication

VergeOS downloads the image itself, from a URL the provider hands it. VergeOS has nowhere to put a request header, so the provider asks Omni for a *standalone* URL — one carrying any authentication inside the URL.

A factory that authenticates with a token in the URL therefore works. One that requires request headers does not: the provider fails the Machine Request with an explanation rather than handing VergeOS a URL it cannot fetch, which would otherwise surface as an opaque download failure inside VergeOS.

## Image cache lifecycle

Cached `omni-talos-*.qcow2` files persist after machines are deleted. This is intentional and improves subsequent provisioning speed. Operators must currently clean unused images manually.

Cached files are shared: they are not owned by any one Machine Request and are never removed during deprovisioning. Every VM gets its own imported boot drive from the shared file. Garbage collection must therefore be a separate operation that first confirms **no VM drive references the file**.

## Not yet validated

Live-environment behaviour that has not been exercised, beyond the image-path caveat above:

- How `filesize` transitions during a server-side URL import on your VergeOS release — the provider's readiness check depends on it.
- Recovery from a failed or interrupted URL import.
- Concurrent first-time requests for the same image. The provider serializes these in-process, but two provider instances sharing one VergeOS cloud are not covered.
- Cached-image garbage collection, which is not implemented at all.

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

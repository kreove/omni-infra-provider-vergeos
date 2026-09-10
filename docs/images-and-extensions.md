# Images and system extensions

## Image selection is controlled by Omni

In automatic mode, the provider does not select a fixed generic Talos image. Omni supplies:

- The Talos version requested by the cluster
- The resolved Image Factory schematic
- The machine join configuration

The schematic represents image-affecting configuration such as system extensions. The provider builds this URL:

```text
https://factory.talos.dev/image/<schematic>/<talos-version>/nocloud-amd64.qcow2
```

The configured Image Factory base URL may be public or self-hosted.

## Cache identity

The complete image URL is hashed into a deterministic name:

```text
omni-talos-<24-hex-character-hash>.qcow2
```

A new file is created when any of these change:

- Schematic
- Talos version
- Architecture
- Image Factory base URL

Cached files are shared across machines and clusters using the same image identity.

## Configure extensions in a cluster template

Cluster-wide extensions:

```yaml
kind: Cluster
name: vergeos-example
kubernetes:
  version: v1.36.1
talos:
  version: v1.13.2
systemExtensions:
  - siderolabs/hello-world-service
---
kind: ControlPlane
machineClass:
  name: vergeos-control-plane
  size: 3
---
kind: Workers
name: workers
machineClass:
  name: vergeos-workers
  size: 3
```

Omni also supports applying extension configuration at machine-set or individual-machine scope. Consult the current Omni cluster-template and extensions documentation for the exact scoping rules supported by your release:

- <https://docs.siderolabs.com/omni/infrastructure-and-extensions/install-talos-linux-extensions>
- <https://docs.siderolabs.com/omni/omni-cluster-setup/how-configuration-works-in-omni>
- <https://docs.siderolabs.com/omni/reference/cluster-templates>

## Configure extensions in the Omni UI

Use the Extensions controls on the cluster, machine set, or machine in Omni. When Omni changes the resolved extension configuration, it creates a new image identity and performs the applicable Talos upgrade workflow.

The provider does not need to be restarted when extensions change.

## First use of a new schematic

1. Omni creates a Machine Request with a schematic and Talos version.
2. The provider asks Omni for the installation medium and receives a URL, the schematic ID, and a storage key.
3. The provider searches VergeOS Files for the cache name derived from that storage key.
4. If absent, the provider creates a VergeOS file pointing at the URL.
5. VergeOS downloads the QCOW2 directly.
6. The provider waits for the file to become ready.
7. The provider creates the VM boot disk from the cached file.

Subsequent machines skip the download.

## Manual image override

Set `image_file_id` only when you intentionally want to bypass automatic selection:

```yaml
image_file_id: 123
```

With a manual override:

- The selected VergeOS file is used for every Machine Request using that Machine Class.
- Changes to Talos version or extensions do not change the selected file.
- The provider does not validate that the image contains the requested extensions.
- The operator is responsible for image lifecycle and compatibility.

Manual mode is useful for testing, disconnected environments, or emergency rollback, but automatic mode is recommended for normal Omni-managed clusters.

## Self-hosted Image Factory

Configure it **in Omni**. The provider has no Image Factory setting of its own: it asks Omni for the installation medium and Omni returns a URL.

VergeOS must trust the factory's TLS certificate and be able to resolve and reach the hostname, because **VergeOS performs the download, not the provider**. That is also why the provider asks Omni for a *standalone* URL — one carrying any authentication inside the URL itself, since VergeOS has nowhere to put a request header.

A factory that authenticates with a token in the URL therefore works. One that requires request headers does not, and the provider says so explicitly rather than handing VergeOS a URL it cannot fetch.

Official self-hosted Image Factory guide:

- <https://docs.siderolabs.com/omni/self-hosted/run-image-factory-on-prem>

## Cache cleanup

The provider intentionally does not delete cached image files during VM deprovisioning because they may be shared by other machines or future scale-up operations.

For now, clean unused files manually:

1. Filter VergeOS Files for names beginning with `omni-talos-`.
2. Confirm no VM drive references the file.
3. Retain images required for active clusters or rollback.
4. Delete only files that are no longer referenced.

Automatic garbage collection is not implemented.

## Guest-agent extension

The provider currently creates VergeOS VMs with the VergeOS guest-agent option disabled. Installing a Talos QEMU guest-agent extension may not, by itself, enable all VergeOS guest-agent integrations. Treat guest-agent support as unverified until the provider and VergeOS VM setting are updated and tested together.

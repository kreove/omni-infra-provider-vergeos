# Omni Infrastructure Provider for VergeOS

Alpha Sidero Omni infrastructure provider that creates and removes Talos VMs on VergeOS. The provider follows the lifecycle pattern of Sidero Labs' KubeVirt provider and uses the official VergeOS Go SDK.

This version adds automatic Talos Image Factory imports and a shared image cache in VergeOS.

## Provisioning flow

For each Omni `MachineRequest`, the provider:

1. Validates the MachineClass data.
2. Generates and records the Omni Talos schematic ID and Talos version.
3. Validates the target VergeOS cluster and VNET.
4. Resolves a manual `image_file_id`, or builds the exact Image Factory QCOW2 URL.
5. Reuses an existing cached VergeOS file or asks VergeOS to import it directly from Image Factory.
6. Waits until VergeOS reports a non-zero file size.
7. Creates the VM, imports the shared image into a per-VM boot disk, adds the NIC, injects Omni join configuration through NoCloud, and powers on the VM.

When Omni deprovisions a machine, the provider removes its NICs, drives, and VM. Shared cached Talos files are preserved for future machines.

## Automatic image cache

When `image_file_id` is omitted, the provider builds an image URL from:

- Omni schematic ID
- Talos version selected by Omni
- MachineClass architecture
- `TALOS_IMAGE_FACTORY_BASE_URL`

Example URL:

```text
https://factory.talos.dev/image/<schematic>/v1.12.4/nocloud-amd64.qcow2
```

The complete URL is SHA-256 hashed into a deterministic VergeOS filename such as:

```text
omni-talos-5de8191a1f041eec6ef1fc31.qcow2
```

Machines using the same schematic, Talos version, architecture, and factory reuse the same file. A different Talos version or extension set produces a different cache entry.

The download is performed by VergeOS, not by the provider container. VergeOS therefore needs DNS and outbound HTTPS access to the configured Image Factory.

## Manual image override

`image_file_id` remains supported. Set it to an existing VergeOS Files ID to bypass automatic importing:

```yaml
cluster_id: 1
vnet_id: 42
image_file_id: 123
architecture: amd64
cores: 4
memory: 8192
disk_size: 32
preferred_tier: "3"
disk_interface: virtio-scsi
network_interface: virtio
uefi: true
```

## Recommended MachineClass

Automatic image mode:

```yaml
cluster_id: 1
vnet_id: 42
architecture: amd64
cores: 4
memory: 8192
disk_size: 32
preferred_tier: "3"
cpu_type: host
disk_interface: virtio-scsi
network_interface: virtio
uefi: true
```

`image_file_id` is intentionally absent.

## Configuration

Required:

```dotenv
OMNI_ENDPOINT=https://omni.example.internal
OMNI_SERVICE_ACCOUNT_KEY=...
VERGEOS_ENDPOINT=https://verge.example.internal
VERGEOS_API_KEY=...
```

Optional:

```dotenv
TALOS_IMAGE_FACTORY_BASE_URL=https://factory.talos.dev
```

The default Image Factory URL is `https://factory.talos.dev`.

Available flags include:

```text
--image-factory-base-url
--vergeos-insecure-skip-verify
--insecure-skip-verify
```

## Docker Compose

```yaml
services:
  omni-infra-provider-vergeos:
    image: kreove/omni-infra-provider-vergeos:latest
    restart: unless-stopped
    environment:
      OMNI_ENDPOINT: ${OMNI_ENDPOINT}
      OMNI_SERVICE_ACCOUNT_KEY: ${OMNI_VERGEOS_SERVICE_ACCOUNT_KEY}
      VERGEOS_ENDPOINT: ${VERGEOS_ENDPOINT}
      VERGEOS_API_KEY: ${VERGEOS_API_KEY}
      TALOS_IMAGE_FACTORY_BASE_URL: https://factory.talos.dev
    command:
      - --vergeos-insecure-skip-verify
```

Rebuild after replacing the source:

```bash
docker build --no-cache \
  -t kreove/omni-infra-provider-vergeos:latest \
  .
```

Then recreate the provider:

```bash
docker compose up -d --force-recreate omni-infra-provider-vergeos
docker compose logs -f omni-infra-provider-vergeos
```

## Expected first-image logs

```text
started Talos image import
waiting for Talos image import
created VergeOS VM
machine is running
```

Subsequent machines using the same image should skip the import and proceed directly to VM provisioning.

## Failed or stuck imports

The current VergeOS SDK file model does not expose a dedicated import status. This provider treats a file as ready when its `filesize` becomes greater than zero.

If a failed import leaves a zero-size `omni-talos-*.qcow2` entry, remove that file manually from VergeOS and retry the Omni machine request. Do not delete cached files that are referenced by VM drives.

## VergeOS permissions

The API key needs:

- Read access to clusters and VNETs.
- Read and create access to Files for automatic image mode.
- Create, read, and delete access to VMs, VM drives, and VM NICs.
- Permission to power VMs on and off.

The provider does not delete shared cached image files.

## Build

```bash
docker build --pull --no-cache \
  -t omni-infra-provider-vergeos:autoimage \
  .
```

The Dockerfile runs `go mod tidy`, `go mod verify`, and `go build` with Go 1.26.2.

## Source layout

- `cmd/omni-infra-provider-vergeos/main.go`: startup, credentials, and Image Factory configuration.
- `internal/pkg/provider/provision.go`: machine lifecycle steps.
- `internal/pkg/provider/image.go`: Image Factory URL creation, VergeOS import, and cache reconciliation.
- `internal/pkg/provider/data`: MachineClass data and JSON schema.
- `internal/pkg/provider/resources`: provider state stored in Omni/COSI.

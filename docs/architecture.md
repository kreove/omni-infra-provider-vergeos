# Architecture and lifecycle

## Components

```mermaid
flowchart TB
    O[Omni] <-->|Provider API| P[VergeOS infrastructure provider]
    P <-->|REST API| V[VergeOS]
    V -->|HTTPS download| F[Talos Image Factory]
    VM[Talos VM] -->|SideroLink / Omni APIs| O
    V --> VM
```

The provider is a stateless reconciliation service with a small amount of machine lifecycle state stored through Omni's infrastructure-provider framework.

## Provisioning steps

The provider registers these ordered steps:

1. `validateRequest`
2. `ensureTarget`
3. `ensureImage`
4. `syncMachine`

### `validateRequest`

- Validates the Machine Request name length.
- Decodes provider data.
- Applies defaults.
- Validates cluster, VNET, architecture, CPU, memory, disk, and image override values.

### `ensureTarget`

- Confirms the VergeOS cluster exists.
- Confirms the VergeOS VNET exists.

No VM is created until target validation succeeds.

### `ensureImage`

Asks Omni for the installation medium: a NoCloud QCOW2 disk image for the requested architecture. Omni ensures the schematic exists on the factory it is configured with and returns a URL, the schematic ID, and a storage key. The serial console kernel argument `console=ttyS0,38400n8` is applied here, and the schematic and Talos version are recorded in provider machine state.

There is no separate `createSchematic` step. Resolving the medium ensures the schematic and reports its ID in the same call, so a separate step would ask Omni for the same medium twice per reconcile — and the URL it returns is short-lived, so it belongs in the step that hands it to VergeOS.

Because VergeOS performs the download itself, the provider asks Omni for a **standalone URL**: one that needs no request headers, since VergeOS has nowhere to put them. If the configured factory still requires headers, the provider fails with an explanation rather than handing VergeOS a URL it cannot fetch.

The cache name comes from the medium's storage key. The URL is never used to derive it, never logged, and never written into the file description: it can carry credentials or a download token, so a name derived from it would change when those rotate and orphan the file already imported under the old name.

Manual mode:

- Reads the specified `image_file_id`.

Automatic mode:

- Builds the exact Image Factory QCOW2 URL.
- Derives a deterministic cache filename.
- Reuses a valid cached file or starts a VergeOS URL import.
- Waits until the file reports a non-zero size.
- Stores the selected VergeOS file ID in provider state.

A per-process mutex reduces duplicate concurrent imports for the same cache name. This lock is not distributed, which is why a single provider replica is recommended.

### `syncMachine`

- Finds the VM by Omni Machine Request ID.
- Creates the VM if missing.
- Ensures `disk0` exists and is imported from the selected cached file.
- Ensures `nic0` exists on the selected VNET.
- Powers on the VM.

The VM receives:

- Omni Machine Join Config as NoCloud `user-data`
- Instance metadata as `meta-data`
- Minimal NoCloud `network-config`

Network addressing is expected to come from the selected VNET, normally through DHCP.

## Idempotency

Repeated reconciliation uses deterministic identifiers:

| Resource | Identity |
| --- | --- |
| VM | Omni Machine Request ID as VM name |
| Boot disk | `disk0` |
| Primary NIC | `nic0` |
| Cached image | Hash of the complete Image Factory asset URL |

If the provider restarts after partially completing an operation, it inspects VergeOS and continues from the existing resources.

## Deprovisioning

```mermaid
flowchart LR
    A[Omni releases Machine Request] --> B{VM exists?}
    B -- No --> G[Complete]
    B -- Yes --> C[Power off]
    C --> D[Delete NICs]
    D --> E[Delete drives]
    E --> F[Delete VM]
    F --> G
```

Cached Talos files are intentionally not deleted.

## Provider state

The machine state records:

- VergeOS VM UUID
- Selected VergeOS image file ID
- Omni schematic ID
- Talos version

The generated protobuf resource follows the same broad pattern used by other Omni infrastructure providers.

## Security boundaries

- The Omni service-account key authorizes the provider to operate as its registered provider ID.
- The VergeOS API key controls infrastructure access.
- Machine Join Config is sent to VergeOS as VM cloud-init data.
- VergeOS downloads images directly from Image Factory.
- The provider exposes no inbound network service.

Protect the Omni key and VergeOS API key as high-value secrets.

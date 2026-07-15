# Design mapping

| KubeVirt provider step | VergeOS implementation |
|---|---|
| Validate Kubernetes resource name | Validate Omni request name and MachineClass values |
| Generate Image Factory schematic | Same Omni `GenerateSchematicID` call |
| CDI imports Image Factory QCOW2 | VergeOS Files imports the exact QCOW2 URL server-side |
| Reuse shared source DataVolume | Reuse a deterministic `omni-talos-*.qcow2` VergeOS file |
| Clone DataVolume into per-VM PVC | Create a VM drive with `media=import` from the shared file |
| Create VirtualMachine | Create a VergeOS `vms` entry |
| CloudInitNoCloud volume | Inline VergeOS cloud-init files with Omni join config, metadata, and network config |
| Default pod network interface | Create a VM NIC on the selected VNET |
| Set VM running | Invoke VergeOS VM power-on action |
| Delete VirtualMachine | Power off; delete NICs, drives, and VM |

## Identifier handling

VergeOS exposes two identifiers for a VM:

- The VM row `$key`, used for VM reads, power actions, and deletion.
- The internal `machine` ID, used by VM drives and VM NICs.

The provider intentionally keeps these separate.

## Image cache identity

The source URL contains the Omni schematic ID, Talos version, and architecture. The provider hashes that complete URL and uses the first 96 bits of SHA-256 in the VergeOS filename. The full URL remains in the VergeOS file metadata and is checked when reusing a cache entry.

## Image ownership

Shared VergeOS Files objects are not owned by an individual machine request and are never deleted during machine deprovisioning. Every VM receives its own imported/resized boot drive. Image garbage collection must therefore be a separate operation that verifies no drive references the file.

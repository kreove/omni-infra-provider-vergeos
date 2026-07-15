# Design mapping

| KubeVirt provider step | VergeOS implementation |
|---|---|
| Validate Kubernetes resource name | Validate Omni request name and MachineClass values |
| Generate Image Factory schematic | Same Omni `GenerateSchematicID` call |
| CDI imports Image Factory QCOW2 | Validate a configured shared VergeOS Files ID |
| Clone DataVolume into per-VM PVC | Create `machine_drives` entry with `media=import` |
| Create VirtualMachine | Create VergeOS `vms` entry |
| CloudInitNoCloud volume | Inline VergeOS `cloudinit_files` with Omni join config, metadata, and network config |
| Default pod network interface | Create `machine_nics` entry on selected VNET |
| Set VM running | Invoke VergeOS VM power-on action |
| Delete VirtualMachine | Power off; delete NICs, drives, and VM |

## Identifier handling

VergeOS exposes two identifiers for a VM:

- The VM row `$key`, used for VM reads, power actions, and deletion.
- The internal `machine` ID, used by `machine_drives` and `machine_nics`.

The provider intentionally keeps these separate.

## Image strategy

The shared VergeOS Files object is never deleted by machine deprovisioning. Each VM receives its own imported/resized boot disk. Automatic Image Factory URL ingestion should be implemented as a separate image resolver so it can poll VergeOS import tasks without complicating machine lifecycle reconciliation.

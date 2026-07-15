# Automatic Talos images

## What changed

- `image_file_id` is optional.
- `ensureImage` runs after cluster/VNET validation and before VM creation.
- VergeOS imports the QCOW2 directly from Talos Image Factory.
- Images are cached by a deterministic hash of the complete source URL.
- Per-machine disks are still imported from the shared VergeOS file.
- Cached files are not deleted during machine deprovisioning.

## First test

1. Add `TALOS_IMAGE_FACTORY_BASE_URL=https://factory.talos.dev` to the provider container.
2. Remove `image_file_id` from the test MachineClass.
3. Request one disposable machine.
4. Watch provider logs and VergeOS Files/Tasks.
5. Confirm an `omni-talos-*.qcow2` file appears before the VM is created.
6. Scale to a second machine and confirm the existing file is reused.

## Cache identity

The cache key contains the Image Factory URL, which itself contains the schematic, Talos version, and architecture. This means extension changes and Talos upgrades automatically generate separate cached files.

## Cleanup

Image cleanup is intentionally manual in this release. Before deleting a cached image, confirm no VM drive references it.

# Implementation status

This repository is an **alpha integration scaffold**, not a production-certified provider.

Implemented:

- Omni infrastructure-provider registration and MachineClass schema.
- VergeOS API-key or username/password authentication.
- Idempotent VM lookup by Omni request name.
- VM creation with CPU, RAM, cluster, UEFI, VNC plus serial-port access, and NoCloud data.
- Omni join configuration injection as cloud-init `user-data`.
- Target cluster, VNET, and image validation before VM creation.
- Boot-disk import from a shared VergeOS Files object, including import polling recovery.
- VNET NIC creation.
- Power-on and deletion lifecycle.
- Explicit NIC and per-VM disk cleanup while preserving the shared image file.

Known gaps:

- It has not been exercised against a live VergeOS 26.x system or an Omni account.
- Automatic download of the exact Image Factory schematic is not implemented yet.
  The MachineClass must reference a pre-imported Talos `nocloud-amd64.qcow2` Files ID.
- Custom Talos system extensions requested through Omni will not be present unless the referenced image was built with those extensions.
- Only `amd64` is exposed in the alpha schema.
- VM hardware changes after initial creation are not reconciled.
- VergeOS NoCloud delivery and boot behavior must still be confirmed on the target VergeOS release.
- A full dependency-level compile was not possible in the offline generation workspace: it had Go 1.23.2, while the current Omni client requires Go 1.26.2. The included CI workflow performs that check after the repository is pushed.

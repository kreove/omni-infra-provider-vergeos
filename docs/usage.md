# Using the provider

## Provision a cluster

After the provider is connected and a VergeOS-backed Machine Class exists, create a cluster from the Omni UI or with a cluster template.

Example:

```yaml
kind: Cluster
name: vergeos-production
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
name: general
machineClass:
  name: vergeos-workers
  size: 3
```

Replace the version values with versions offered and supported by your Omni instance.

```bash
omnictl cluster template validate -f cluster.yaml
omnictl cluster template sync -f cluster.yaml --verbose
omnictl cluster template status -f cluster.yaml
```

Official cluster-template reference:

- <https://docs.siderolabs.com/omni/reference/cluster-templates>

## What happens during the first provisioning request

For the first machine using a given image identity, provisioning takes longer because VergeOS downloads the QCOW2 from Image Factory.

Provider log progression:

```text
started Talos image import
waiting for Talos image import
created VergeOS VM
machine is running
```

The provider waits until the VergeOS Files object reports a non-zero logical size before creating the boot disk.

## Reusing an image

The cache identity includes:

- Image Factory base URL
- Omni schematic ID
- Talos version
- Architecture

Machines using the same combination reuse the same `omni-talos-*.qcow2` file. Separate Machine Classes can therefore share an image while still using different CPU, memory, disk, VNET, and cluster settings.

## Scale a machine set

Change the `size` value in the cluster template and sync it again, or use the corresponding Omni UI controls.

Scale up:

```yaml
kind: Workers
name: general
machineClass:
  name: vergeos-workers
  size: 5
```

Scale down:

```yaml
kind: Workers
name: general
machineClass:
  name: vergeos-workers
  size: 2
```

When Omni releases a dynamically provisioned machine, the provider:

1. Powers off the VM.
2. Deletes its NICs.
3. Deletes its drives.
4. Deletes the VM object.
5. Leaves shared cached Talos files in VergeOS.

## Change Talos versions

Upgrade Talos through Omni. The new Talos version produces a different Image Factory URL and cache filename. VergeOS imports the new image once, after which machines can reuse it during the rollout.

Official upgrade guide:

- <https://docs.siderolabs.com/omni/cluster-management/upgrading-clusters>

## Add or change system extensions

Configure extensions in Omni rather than pinning `image_file_id`. Extension changes produce a new schematic and therefore a new cached VergeOS image.

See [Images and system extensions](images-and-extensions.md).

## Use different VM sizes

Create multiple provider-backed Machine Classes. For example:

### Control plane

```yaml
cluster_id: 1
vnet_id: 42
architecture: amd64
cores: 4
memory: 8192
disk_size: 40
preferred_tier: "2"
cpu_type: host
uefi: true
```

### General workers

```yaml
cluster_id: 1
vnet_id: 42
architecture: amd64
cores: 8
memory: 16384
disk_size: 80
preferred_tier: "3"
cpu_type: host
uefi: true
```

### Storage workers on another VNET

```yaml
cluster_id: 1
vnet_id: 55
architecture: amd64
cores: 8
memory: 32768
disk_size: 160
preferred_tier: "2"
cpu_type: host
uefi: true
```

## Move between VergeOS clusters or VNETs

`cluster_id` and `vnet_id` are evaluated when a VM is created. Changing them in a Machine Class does not migrate existing VMs. Replace or scale the machine set so Omni provisions new machines with the new values, then removes the old machines through its normal rollout and deprovisioning workflow.

## Provider restarts

Provisioning is designed to be idempotent:

- Existing VMs are found by Omni Machine Request name.
- Existing `disk0` and `nic0` resources are reused.
- Cached files are found by deterministic name.
- Deletion tolerates resources that are already absent.

A provider restart should resume reconciliation. Still, validate restart behavior in your environment before production use.

## Logs

Docker Compose:

```bash
docker compose logs -f --tail=200 omni-infra-provider-vergeos
```

Kubernetes:

```bash
kubectl -n omni-infra-provider-vergeos logs -f \
  deployment/omni-infra-provider-vergeos
```

Useful log fields include the provider ID, VergeOS endpoint, image URL, cached file ID, VM ID, and internal VergeOS machine ID.

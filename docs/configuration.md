# Configuration reference

The provider accepts environment variables and equivalent command-line flags. Command-line flags take precedence over environment-derived defaults.

## Provider configuration

| Environment variable | Flag | Required | Default | Description |
| --- | --- | ---: | --- | --- |
| `OMNI_ENDPOINT` | `--omni-api-endpoint` | Yes | none | Omni API base URL |
| `OMNI_SERVICE_ACCOUNT_KEY` | `--omni-service-account-key` | Normally | none | Infrastructure-provider service account key |
| n/a | `--id` | No | `vergeos` | Provider ID registered in Omni |
| n/a | `--provider-name` | No | `VergeOS` | Display name in Omni |
| n/a | `--provider-description` | No | alpha description | Display description in Omni |
| `VERGEOS_ENDPOINT` or `VERGEOS_HOST` | `--vergeos-endpoint` | Yes | none | VergeOS base URL, including scheme |
| `VERGEOS_API_KEY` | `--vergeos-api-key` | One auth method | none | Preferred VergeOS authentication method |
| `VERGEOS_USERNAME` | `--vergeos-username` | One auth method | none | Username when API-key auth is not used |
| `VERGEOS_PASSWORD` | `--vergeos-password` | One auth method | none | Password when API-key auth is not used |
| `TALOS_IMAGE_FACTORY_BASE_URL` | `--image-factory-base-url` | No | `https://factory.talos.dev` | Public or private Image Factory base URL |
| n/a | `--vergeos-timeout` | No | `3m` | VergeOS API request timeout |
| n/a | `--vergeos-insecure-skip-verify` | No | `false` | Skip VergeOS TLS verification |
| n/a | `--insecure-skip-verify` | No | `false` | Skip Omni TLS verification |

### TLS

Use trusted certificates whenever possible. The insecure flags disable certificate verification and should be limited to temporary tests or isolated environments.

For an internal certificate authority, the better approach is to add its CA certificate to a custom runtime image rather than disabling verification.

### Authentication priority

The provider uses VergeOS credentials in this order:

1. `VERGEOS_API_KEY`
2. `VERGEOS_USERNAME` and `VERGEOS_PASSWORD`

If an API key is set, username/password values are ignored.

## Machine Class provider data

The schema is defined in `internal/pkg/provider/data/schema.json` and registered with Omni at startup.

| Field | Required | Default | Description |
| --- | ---: | --- | --- |
| `cluster_id` | Yes | none | VergeOS cluster object ID |
| `vnet_id` | Yes | none | VergeOS VNET object ID |
| `image_file_id` | No | `0` | Existing VergeOS Files ID; `0` or absent enables automatic images |
| `architecture` | Yes in schema | `amd64` | Only `amd64` is currently supported |
| `cores` | Yes | UI default `4` | Virtual CPU core count |
| `memory` | Yes | UI default `8192` | Memory in MiB; minimum 2048 |
| `disk_size` | Yes | UI default `32` | Boot disk size in GiB; minimum 5 |
| `preferred_tier` | No | VergeOS default | Storage tier used for cached files and VM disks |
| `cpu_type` | No | VergeOS default | QEMU CPU model, for example `host` |
| `machine_type` | No | VergeOS default | QEMU machine type |
| `disk_interface` | No | `virtio-scsi` | `virtio-scsi`, `virtio`, `nvme`, or `ahci` |
| `network_interface` | No | `virtio` | `virtio`, `e1000`, `e1000e`, or `vmxnet3` |
| `uefi` | No | `true` | Enable UEFI firmware |

### Recommended automatic-image configuration

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

### Manual image override

```yaml
cluster_id: 1
vnet_id: 42
image_file_id: 123
architecture: amd64
cores: 4
memory: 8192
disk_size: 32
disk_interface: virtio-scsi
network_interface: virtio
uefi: true
```

A manual image override bypasses automatic schematic-based image selection. The operator is responsible for ensuring that the image is compatible with the Talos version, Omni join workflow, architecture, and extensions requested by the cluster.

## VM settings applied by the provider

The provider currently creates VMs with:

- Linux guest type
- UEFI configurable from Machine Class data
- Secure Boot disabled
- VNC console
- Serial port enabled
- NoCloud cloud-init data source
- Omni Machine Join Config in `user-data`
- Boot disk named `disk0` with order `0`
- Primary NIC named `nic0` with order `0`

Existing VMs are not resized or otherwise reconciled when Machine Class CPU, RAM, disk, or interface settings change. Those settings apply to newly provisioned machines.

## Multiple provider instances

To manage more than one isolated VergeOS cloud, run separate containers:

```yaml
services:
  vergeos-site-a:
    image: ${PROVIDER_IMAGE}
    environment:
      OMNI_ENDPOINT: ${OMNI_ENDPOINT}
      OMNI_SERVICE_ACCOUNT_KEY: ${SITE_A_OMNI_KEY}
      VERGEOS_ENDPOINT: https://site-a.example.com
      VERGEOS_API_KEY: ${SITE_A_VERGEOS_KEY}
    command: ["--id", "vergeos-site-a"]

  vergeos-site-b:
    image: ${PROVIDER_IMAGE}
    environment:
      OMNI_ENDPOINT: ${OMNI_ENDPOINT}
      OMNI_SERVICE_ACCOUNT_KEY: ${SITE_B_OMNI_KEY}
      VERGEOS_ENDPOINT: https://site-b.example.com
      VERGEOS_API_KEY: ${SITE_B_VERGEOS_KEY}
    command: ["--id", "vergeos-site-b"]
```

Create matching Omni infrastructure providers named `vergeos-site-a` and `vergeos-site-b`.

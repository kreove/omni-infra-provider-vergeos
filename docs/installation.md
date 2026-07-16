# Installation

This guide deploys the provider as a Docker Compose service. A Kubernetes example is also included in `deploy/kubernetes.yaml`.

## 1. Register an infrastructure provider in Omni

The provider registers under the ID `vergeos` by default. The Omni infrastructure-provider service account must use the same ID.

Using `omnictl`:

```bash
omnictl infraprovider create vergeos
```

The command returns values similar to:

```dotenv
OMNI_ENDPOINT=https://omni.example.com
OMNI_SERVICE_ACCOUNT_KEY=eyJ...
```

Store the key as a secret. To use another provider ID, start the binary with `--id <provider-id>` and create the Omni infrastructure provider using that same ID.

For multiple independent VergeOS clouds, run one provider instance per cloud and give each instance a unique provider ID and Omni service account.

Official Omni reference:

- <https://docs.siderolabs.com/omni/infrastructure-and-extensions/infrastructure-providers>
- <https://docs.siderolabs.com/omni/reference/cli>

## 2. Create a VergeOS service account and API key

Create a dedicated VergeOS user for the provider rather than using an administrator's personal account. Generate an API key for that user and store it securely; VergeOS displays the complete key only when it is created.

The provider performs the following operations:

| Resource | Required operations |
| --- | --- |
| Clusters | list/read |
| VNETs or networks | list/read |
| Files | list/read/create |
| Virtual machines | list/read/create/modify/delete and power operations |
| VM drives | list/read/create/delete |
| VM network interfaces | list/read/create/delete |

Permission names and scopes can differ between VergeOS releases. Start with permissions scoped to the target cloud, cluster, VNET, and storage resources, then reduce them after validating all lifecycle operations.

Official VergeOS references:

- <https://docs.verge.io/product-guide/system/api-keys/>
- <https://docs.verge.io/product-guide/system/permissions/>

## 3. Record the VergeOS object IDs

Each Machine Class needs:

- `cluster_id`: the VergeOS cluster object ID
- `vnet_id`: the VergeOS VNET object ID

These are VergeOS API/database object IDs. `vnet_id` is not a VLAN number.

Use the VergeOS UI/API documentation to identify the relevant objects. Confirm that the selected VNET provides the addressing, DNS, gateway, and routing needed by Talos VMs to reach Omni.

## 4. Obtain the provider image

### Use a published image

Set the full image reference in `deploy/.env`:

```dotenv
PROVIDER_IMAGE=ghcr.io/kreove/omni-infra-provider-vergeos:VERSION
```

Pin a release tag or digest in production. Do not rely on `latest` for controlled upgrades.

### Build locally

From the repository root:

```bash
docker build --pull --no-cache \
  -t omni-infra-provider-vergeos:local \
  .
```

Then set:

```dotenv
PROVIDER_IMAGE=omni-infra-provider-vergeos:local
```

## 5. Configure Docker Compose

```bash
cp deploy/example.env deploy/.env
chmod 600 deploy/.env
```

Populate `deploy/.env`:

```dotenv
PROVIDER_IMAGE=omni-infra-provider-vergeos:local
OMNI_ENDPOINT=https://omni.example.com
OMNI_SERVICE_ACCOUNT_KEY=replace-me
VERGEOS_ENDPOINT=https://vergeos.example.com
VERGEOS_API_KEY=replace-me
TALOS_IMAGE_FACTORY_BASE_URL=https://factory.talos.dev
VERGEOS_INSECURE_SKIP_VERIFY=false
OMNI_INSECURE_SKIP_VERIFY=false
```

Both endpoint values must include `https://` or `http://`. A hostname without a URL scheme causes `unsupported protocol scheme` errors.

Start the service:

```bash
cd deploy
docker compose up -d
docker compose ps
docker compose logs -f omni-infra-provider-vergeos
```

Expected startup log fields include:

```text
starting VergeOS infrastructure provider
provider_id=vergeos
vergeos_endpoint=https://vergeos.example.com
image_factory_base_url=https://factory.talos.dev
```

## 6. Verify connectivity

Confirm the exact environment received by the container:

```bash
docker inspect omni-infra-provider-vergeos \
  --format '{{range .Config.Env}}{{println .}}{{end}}' \
  | grep -E '^(OMNI_ENDPOINT|VERGEOS_ENDPOINT|TALOS_IMAGE_FACTORY_BASE_URL)='
```

The provider needs outbound connectivity to:

- The Omni API endpoint
- The VergeOS API endpoint

VergeOS itself needs outbound DNS and HTTPS access to the configured Image Factory. The provider container does not proxy the QCOW2 download.

The provisioned Talos VMs must be able to reach the Omni addresses and ports configured for your instance. Use the official Omni firewall requirements for your hosted or self-hosted deployment:

- <https://docs.siderolabs.com/omni/omni-cluster-setup/omni-firewall-egress-requirement>

## 7. Create the Machine Class

In Omni, open the registered VergeOS infrastructure provider and create a dynamic Machine Class or machine request. Omni renders the provider-specific form from the schema published by this service.

Recommended provider data:

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

Leave `image_file_id` unset for automatic image selection.

## 8. Run a smoke test

Start with one disposable control-plane machine and no workers. Watch both systems:

```bash
docker compose logs -f omni-infra-provider-vergeos
```

In VergeOS, monitor Files, Tasks, and Virtual Machines. The expected first-time sequence is:

1. The provider registers the machine request.
2. VergeOS imports an `omni-talos-*.qcow2` file.
3. The provider creates the VM.
4. The provider imports the file into `disk0`.
5. The provider adds `nic0` on the selected VNET.
6. The provider powers on the VM.
7. The Talos machine connects to Omni.

Once that works, test scale-up and scale-down before using the provider for important clusters.

## Kubernetes deployment

Edit the image and Secret values in `deploy/kubernetes.yaml`, then apply it:

```bash
kubectl apply -f deploy/kubernetes.yaml
kubectl -n omni-infra-provider-vergeos logs -f deployment/omni-infra-provider-vergeos
```

Run exactly one replica for a provider ID. The current in-process image-import lock is not distributed between replicas.

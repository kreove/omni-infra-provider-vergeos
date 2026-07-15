# Troubleshooting

Start with the provider logs:

```bash
docker compose logs -f --tail=200 omni-infra-provider-vergeos
```

## `unsupported protocol scheme ""`

Example:

```text
Get "vergeos.example.com/version.json": unsupported protocol scheme ""
```

The endpoint is missing its URL scheme.

Wrong:

```dotenv
VERGEOS_ENDPOINT=vergeos.example.com
```

Correct:

```dotenv
VERGEOS_ENDPOINT=https://vergeos.example.com
```

Check the actual container environment:

```bash
docker inspect omni-infra-provider-vergeos \
  --format '{{range .Config.Env}}{{println .}}{{end}}' \
  | grep VERGEOS_ENDPOINT
```

## Invalid Image Factory URL containing `$(`

Example:

```text
invalid Image Factory URL "$(TALOS_IMAGE_FACTORY_BASE_URL:?... )"
```

Docker Compose uses `${VAR}` syntax, not `$(VAR)`.

Use:

```yaml
TALOS_IMAGE_FACTORY_BASE_URL: ${TALOS_IMAGE_FACTORY_BASE_URL:-https://factory.talos.dev}
```

Then recreate the container:

```bash
docker compose up -d --force-recreate omni-infra-provider-vergeos
```

## Provider is not shown as connected in Omni

Check:

1. The Omni infrastructure-provider ID matches `--id`.
2. The service account key belongs to that provider ID.
3. `OMNI_ENDPOINT` includes the URL scheme.
4. The provider container can resolve and reach Omni.
5. TLS verification is succeeding.

Recreate credentials if necessary:

```bash
omnictl infraprovider renewkey vergeos
```

Check the exact syntax supported by your `omnictl` version with:

```bash
omnictl infraprovider --help
```

## VergeOS returns `401` or `403`

The API key is invalid, expired, IP-restricted, or lacks permission for the attempted operation.

Check the dedicated user's effective permissions for:

- clusters
- networks/VNETs
- files
- virtual machines
- VM drives
- VM NICs
- power operations

VergeOS API keys inherit the permissions of their associated user.

## Image import remains at zero size

The provider currently considers a cached file ready when VergeOS reports `filesize > 0`. A failed import can leave a zero-size `omni-talos-*.qcow2` file.

1. Inspect the VergeOS task/event associated with the import.
2. Verify VergeOS DNS and outbound HTTPS access to Image Factory.
3. Verify the image URL from the provider log is reachable from the VergeOS environment.
4. Delete the failed zero-size file if no VM drive references it.
5. Retry the Omni Machine Request.

Do not delete a cached file that is referenced by active VM drives.

## Image Factory returns `404`

Check:

- The Talos version requested by Omni exists in that Image Factory.
- The schematic was successfully generated.
- The factory base URL is correct.
- The architecture is `amd64`.
- A reverse proxy is not stripping the `/image/...` path.

## VM is created but does not join Omni

Check the VM's VNC or serial console and verify:

- The VM boots from `disk0` instead of PXE.
- UEFI is enabled if required by the image.
- The NIC is connected to the intended VNET.
- DHCP or static addressing works.
- DNS and the default gateway are present.
- The VM can reach the Omni SideroLink and API endpoints.
- Firewall rules allow the ports configured by your Omni deployment.

The provider injects the Omni Machine Join Config as NoCloud `user-data`.

## VM falls into the UEFI shell

Inspect the VergeOS VM:

- `disk0` exists and completed importing.
- The disk has boot order `0`.
- The interface is supported by the Talos image, preferably `virtio-scsi`.
- UEFI is enabled.
- The cached QCOW2 is not zero bytes or corrupted.

Delete the failed machine from Omni and allow the provider to recreate it after fixing the Machine Class or image.

## Duplicate VM name

The provider expects exactly one VergeOS VM per Omni Machine Request name. If multiple VMs share the same name, provisioning fails deliberately.

Remove or rename the unmanaged duplicate. Do not arbitrarily delete the VM that is currently registered with Omni.

## Deprovisioning is stuck

The provider removes resources in this order:

1. Power off VM
2. Delete NICs
3. Delete drives
4. Delete VM

Inspect the logs for the exact failing object and verify delete permission. VergeOS may also block deletion while a drive import or another asynchronous operation is still running.

## TLS certificate errors

Preferred fixes:

- Use a certificate issued by a CA trusted by the container.
- Build a runtime image containing your internal CA.

Temporary test options:

```yaml
command:
  - --vergeos-insecure-skip-verify
  - --insecure-skip-verify
```

The first flag affects VergeOS. The second affects Omni.

## Build tries to download local provider packages

Confirm `go.mod` declares:

```text
module omni-infra-provider-vergeos
```

Confirm no placeholder imports remain:

```bash
grep -R 'github.com/your-org/omni-infra-provider-vergeos' \
  --include='*.go' .
```

The command should return nothing.

## Collect information for an issue

Include:

- Provider release tag or commit
- Omni version
- VergeOS version
- Talos version
- Sanitized Machine Class provider data
- Relevant provider logs
- Relevant VergeOS task/event details
- Whether the image was automatic or specified by `image_file_id`

Remove service-account keys, API keys, join tokens, internal credentials, and other secrets before posting logs.

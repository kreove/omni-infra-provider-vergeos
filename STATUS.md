# Status

## Working lifecycle already validated by the user

- Omni creates VergeOS VMs.
- Scaling up creates additional machines.
- Scaling down removes machines.
- Deprovisioning removes VM drives, NICs, and VMs.

## Added in this package

- Automatic Talos Image Factory URL generation.
- Server-side VergeOS URL imports.
- Shared deterministic image cache.
- Optional manual `image_file_id` override.
- Unit tests for image URL/cache identity helpers.

## Still requires live validation

- Exact `filesize` transition behavior during URL imports on the installed VergeOS release.
- Recovery from a failed or interrupted URL import.
- Concurrent first-time requests for the same image.
- Private/authenticated Image Factory endpoints.
- Automated garbage collection of unused cached images.

## Known limitation

The VergeOS SDK file resource does not expose a dedicated asynchronous import status in the fields used by this provider. Readiness is currently based on `filesize > 0`. A failed zero-size cache entry must be removed manually before retrying.

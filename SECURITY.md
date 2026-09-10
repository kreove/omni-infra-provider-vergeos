# Security policy

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability involving credentials, authorization bypass, secret disclosure, unsafe image fetching, or destructive infrastructure behavior.

Use the repository's private security-advisory feature or contact the maintainer through the private security address published with the repository. Include:

- Affected version or commit
- Reproduction steps
- Security impact
- Suggested mitigation, if known

Do not include live Omni service-account keys, VergeOS API keys, Talos join tokens, or other credentials.

## Supported versions

Until the project reaches a stable release, security fixes are provided for the latest published release only.

## Secret handling

Treat these values as secrets:

- `OMNI_SERVICE_ACCOUNT_KEY`
- `VERGEOS_API_KEY`
- `VERGEOS_PASSWORD`
- Machine Join Config and embedded join tokens

Store secrets in a container secret store, protected environment file, or Kubernetes Secret. Do not commit them to source control.

## Image Factory URL security

The provider does not accept an Image Factory URL from any source. It asks Omni for the installation medium it needs, and Omni returns the URL. Machine Class data cannot redirect VergeOS to an arbitrary host, and neither can provider configuration.

**The returned URL may contain credentials**, as userinfo or as a download token in the query string, depending on how the factory Omni talks to is configured. It is handed to VergeOS, which performs the download, so it necessarily reaches the VergeOS API — but the provider keeps it out of everywhere else:

- It is never written to a log line.
- It is never stored in the VergeOS file description, which records the Talos version, architecture and schematic instead. VergeOS shows descriptions to anyone who can list files.
- It is never used to derive the cached file name, which comes from the medium's storage key.

The provider also asks Omni for a *standalone* URL, so any authentication travels inside the URL rather than in request headers VergeOS cannot send.

Operators should:

- Trust only controlled certificate authorities.
- Restrict outbound access from VergeOS where appropriate.
- Review private Image Factory access controls in Omni.

## Least privilege

Use a dedicated VergeOS API user scoped to the resources required by this provider. Avoid administrator credentials for long-running deployments.

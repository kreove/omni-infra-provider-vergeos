# Changelog

All notable changes to this project should be documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and releases should use [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Public installation, configuration, usage, troubleshooting, architecture, compatibility, and development documentation.

## [0.2.0-alpha] - 2026-07-15

### Added

- Dynamic VergeOS VM provisioning from Sidero Omni.
- Clean scale-up, scale-down, and VM deprovisioning.
- Automatic Talos Image Factory URL generation.
- Server-side VergeOS image imports.
- Shared deterministic image caching.
- Optional `image_file_id` override.
- Docker Compose and Kubernetes deployment examples.

### Fixed

- Local Go module imports no longer resolve through a placeholder GitHub path.
- Docker builds generate and verify dependency sums before compilation.

### Known limitations

- `amd64` only.
- Cached-image readiness is based on non-zero VergeOS file size.
- No automatic image cache garbage collection.

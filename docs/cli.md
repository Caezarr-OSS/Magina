# Magina CLI Documentation

## Overview

Magina is a CLI tool for managing and migrating OCI images between registries.

## Available Commands

### `magina export`

Exports images from a source registry to the local host.

```bash
magina export -c <config-file> [flags]
```

**Flags:**
- `-c, --config` : BRMS configuration file (required)
- `-v, --verbose` : Verbosity level (0-3)
- `--clean-on-error` : Clean up images on error
- `--resume` : Continue operation even after errors

**BRMS Format:**
```brms
[protocol://export-host|]
image1:tag1|newimage1:tag1
```

### `magina import`

Imports local images to a destination registry.

```bash
magina import -c <config-file> [flags]
```

**Flags:**
- `-c, --config` : BRMS configuration file (required)
- `-v, --verbose` : Verbosity level (0-3)
- `--clean-on-error` : Clean up images on error
- `--resume` : Continue operation even after errors

**BRMS Format:**
```brms
[|protocol://import-host]
image1:tag1|newimage1:tag1
```

### `magina transfer`

Performs a complete image transfer between two registries.

```bash
magina transfer -c <config-file> [flags]
```

**Flags:**
- `-c, --config` : BRMS configuration file (required)
- `-v, --verbose` : Verbosity level (0-3)
- `--clean-on-error` : Clean up images on error
- `--resume` : Continue operation even after errors

**BRMS Format:**
```brms
[protocol://source-host|protocol://dest-host]
image1:tag1|newimage1:tag1
```

### `magina validate`

Validates a BRMS configuration file.

```bash
magina validate -c <config-file>
```

**Flags:**
- `-c, --config` : BRMS configuration file (required)
- `-v, --verbose` : Verbosity level (0-3)

## Verbosity Levels

- `0` : Silent (errors only)
- `1` : Normal (success + errors)
- `2` : Detailed (progress + metadata)
- `3` : Debug (all information)

## Detailed BRMS Format

### General Structure
```brms
[source|destination]
image_mapping1
image_mapping2
!exclusion1
```

### Examples

#### Simple Export
```brms
[https://registry.company.com|]
app/backend:1.0.0|backend:local
app/frontend:1.0.0|frontend:local
```

#### Simple Import
```brms
[|https://docker.io]
backend:local|company/backend:latest
frontend:local|company/frontend:latest
```

#### Transfer with Exclusions
```brms
[https://registry.company.com|https://docker.io]
!test/*
!dev/*
app/backend:1.0.0|company/backend:latest
app/frontend:1.0.0|company/frontend:latest
```

## Local Image Storage

Magina stores images locally without requiring a container runtime (Docker/Podman). Images are stored as files in the local file system using the OCI standard format.

### Storage Format
Exported images are stored in the native OCI format, making them compatible with any OCI runtime (Docker, Podman, etc.) if you wish to use them later.

### Common Errors
- `failed to parse source image reference` : Invalid image format
- `failed to load source image` : Connection or authentication error to registry
- `failed to save image locally` : Local write issue (permissions, disk space)

Note: Magina does not check for the presence of a container runtime as it does not need one to function.

## Return Codes

- `0` : Success
- `1` : General error
- `2` : Configuration error
- `3` : Connection error
- `4` : Authentication error

## Environment Variables

```bash
# Proxy
HTTP_PROXY="http://proxy.company.com:3128"
HTTPS_PROXY="http://proxy.company.com:3128"
NO_PROXY="localhost,127.0.0.1"

# Docker
DOCKER_HOST="tcp://localhost:2375"
DOCKER_CERT_PATH="/path/to/certs"
DOCKER_TLS_VERIFY="1"
```

## Usage Examples

### Export with Cleanup
```bash
magina export -c config.brms --clean-on-error -v 1
```

### Import with Resume
```bash
magina import -c config.brms --resume -v 2
```

### Complete Transfer in Debug Mode
```bash
magina transfer -c config.brms --clean-on-error --resume -v 3
```

### Simple Validation
```bash
magina validate -c config.brms

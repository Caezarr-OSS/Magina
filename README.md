# Magina

Magina is a Command Line Interface (CLI) tool for managing and migrating OCI (Open Container Initiative) images between different registries. It uses a custom BRMS (Block Relation Mapping Syntax) configuration format to define image mappings.

![Magina Logo](assets/img/magina-logo.png)

## Features

- Export images from source registry
- Import images to destination registry
- Complete image transfer between registries
- Registry authentication support
- BRMS configuration validation
- Automatic cleanup on error
- Error recovery support
- Detailed configurable logging
- Works without container runtime (Docker or Podman not required)

## Prerequisites

### System
- Windows, Linux, or macOS (amd64 or arm64)
- 1GB minimum disk space
- 512MB minimum RAM (1GB recommended)

### Network
- Internet access for remote registries
- Ports 80/443 accessible
- Proxy configuration if needed

### Note on Container Runtimes
Magina works autonomously and doesn't require a container runtime (Docker, Podman, etc.) as it interacts directly with registries via the standard OCI protocol. However, if you want to use the images locally after downloading them, you'll need an OCI-compatible runtime like Docker or Podman.

## Installation

### From Releases
```bash
# Download the latest version from GitHub Releases
# Replace <platform> with linux, darwin, or windows
# Replace <arch> with amd64 or arm64
curl -LO https://github.com/Caezarr-OSS/magina/releases/latest/download/magina-<platform>-<arch>.tar.gz
tar xzf magina-<platform>-<arch>.tar.gz
chmod +x magina-<platform>-<arch>
mv magina-<platform>-<arch> /usr/local/bin/magina  # or another directory in PATH
```

### From Source
```bash
# Clone the repository
git clone https://github.com/Caezarr-OSS/magina.git
cd magina

# Install Task
sh -c "$(curl --location https://taskfile.dev/install.sh)"

# Build for your platform
task build

# Or build for all platforms
task release
```

## Usage

### BRMS Format
```brms
# General format
[protocol://source-registry|protocol://dest-registry]
image1:tag1|newimage1:tag1
image2:tag2|newimage2:tag2

# Concrete example
[https://registry.company.com|https://docker.io]
app/backend:1.0.0|company/backend:latest
app/frontend:1.0.0|company/frontend:latest
```

### Commands

#### Export
```bash
# Export images from source registry
magina export -c config.brms --clean-on-error -v 1
```

#### Import
```bash
# Import images to destination registry
magina import -c config.brms --resume -v 2
```

#### Transfer
```bash
# Complete transfer between registries
magina transfer -c config.brms --clean-on-error --resume -v 1
```

#### Validate
```bash
# Validate a configuration file
magina validate -c config.brms
```

### Global Options
- `-c, --config` : BRMS configuration file (required)
- `-v, --verbose` : Verbosity level (0-3)
- `--version` : Display version

### Transfer Options
- `--clean-on-error` : Clean up images on error
- `--resume` : Continue operation even after errors

## Development

### Project Structure
```
magina/
├── cmd/            # Application entry point
├── internal/       # Internal code
├── docs/          # Documentation
├── .github/       # GitHub Actions configuration
├── Taskfile.yml   # Build tasks
└── README.md      # Main documentation
```

### Development Commands
```bash
# Install tools
task install

# Tests
task test

# Linting
task lint

# Local build
task build

# Build all platforms
task release

# Generate CLI documentation
task generate-docs
```

## Known Limitations

- No support for protocol-less registries
- No operation parallelization
- No automatic retry on network error
- No persistent credential storage
- No native IPv6-only support

## Contributing

1. Fork the project
2. Create a branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is under the MIT License. See the [LICENSE](LICENSE) file for details.

## Support

For questions or issues:
1. Check the [Issues](https://github.com/Caezarr-OSS/magina/issues)
2. Open a new issue if needed
3. Include logs and configuration (without credentials)
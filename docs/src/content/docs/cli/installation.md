---
title: CLI Installation
description: Install the Fluxbase CLI tool
---

The Fluxbase CLI provides command-line access to manage your Fluxbase platform, including functions, jobs, storage, AI chatbots, and more.

## Installation Methods

### Install Script (Recommended)

The easiest way to install the Fluxbase CLI:

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash -s -- v0.0.1-rc.93
```

The script automatically detects your OS and architecture, downloads the appropriate binary, and installs it to `/usr/local/bin`.

### Manual Download

Download the latest CLI binary for your platform from the [GitHub Releases page](https://github.com/fluxbase-eu/fluxbase/releases).

#### macOS

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-darwin-arm64.tar.gz
tar -xzf fluxbase-darwin-arm64.tar.gz
sudo mv fluxbase-darwin-arm64 /usr/local/bin/fluxbase

# macOS (Intel)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-darwin-amd64.tar.gz
tar -xzf fluxbase-darwin-amd64.tar.gz
sudo mv fluxbase-darwin-amd64 /usr/local/bin/fluxbase
```

#### Linux

```bash
# Linux (x86_64)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-amd64.tar.gz
tar -xzf fluxbase-linux-amd64.tar.gz
sudo mv fluxbase-linux-amd64 /usr/local/bin/fluxbase

# Linux (ARM64)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-arm64.tar.gz
tar -xzf fluxbase-linux-arm64.tar.gz
sudo mv fluxbase-linux-arm64 /usr/local/bin/fluxbase
```

#### Windows

Download from the [releases page](https://github.com/fluxbase-eu/fluxbase/releases):

1. Download `fluxbase-windows-amd64.zip`
2. Extract the archive
3. Move `fluxbase-windows-amd64.exe` to a directory in your PATH (e.g., `C:\Program Files\Fluxbase\`)
4. Rename to `fluxbase.exe` for convenience

Or using PowerShell:

```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-windows-amd64.zip" -OutFile "fluxbase-windows-amd64.zip"
Expand-Archive -Path "fluxbase-windows-amd64.zip" -DestinationPath "."

# Move to a directory in PATH (run as Administrator)
Move-Item -Path "fluxbase-windows-amd64.exe" -Destination "C:\Program Files\Fluxbase\fluxbase.exe"
```

### From Source

If you have Go 1.25+ installed, you can build the CLI from source:

```bash
# Clone the repository
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase

# Build and install
make cli-install
```

This installs the `fluxbase` command to `/usr/local/bin`.

### Verify Installation

```bash
fluxbase version
```

## Shell Completion

Enable tab completion for your shell:

### Bash

```bash
# Add to ~/.bashrc
source <(fluxbase completion bash)

# Or install globally
fluxbase completion bash > /etc/bash_completion.d/fluxbase
```

### Zsh

```bash
# Add to ~/.zshrc
source <(fluxbase completion zsh)

# Or add to fpath
fluxbase completion zsh > "${fpath[1]}/_fluxbase"
```

### Fish

```bash
fluxbase completion fish | source

# Or install permanently
fluxbase completion fish > ~/.config/fish/completions/fluxbase.fish
```

### PowerShell

```powershell
fluxbase completion powershell | Out-String | Invoke-Expression
```

## Next Steps

- [Getting Started](/cli/getting-started) - Configure and authenticate
- [Command Reference](/cli/commands) - Full command documentation
- [Configuration](/cli/configuration) - Configuration options

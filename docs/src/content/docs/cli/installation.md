---
title: CLI Installation
description: Install the Fluxbase CLI tool
---

The Fluxbase CLI (`fluxbase`) provides command-line access to manage your Fluxbase platform, including functions, jobs, storage, AI chatbots, and more.

## Requirements

- macOS, Linux, or Windows
- Network access to your Fluxbase server
- (Optional) [Deno](https://deno.land/) for local function bundling

## Installation Methods

### Install Script (Recommended)

The easiest way to install the Fluxbase CLI:

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash -s -- v2026.1.13
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

Expected output:

```
fluxbase version 0.0.1
commit: abc1234
built: 2024-01-15T10:30:00Z
```

## Updating

### Using the Install Script

Run the install script again to update to the latest version:

```bash
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash
```

### Checking for Updates

Compare your version with the latest release:

```bash
# Your current version
fluxbase version

# Check latest release on GitHub
curl -s https://api.github.com/repos/fluxbase-eu/fluxbase/releases/latest | grep tag_name
```

## Uninstallation

### macOS / Linux

```bash
sudo rm /usr/local/bin/fluxbase
rm -rf ~/.fluxbase  # Remove configuration (optional)
```

### Windows

1. Delete the `fluxbase.exe` binary from your installation directory
2. Remove `%USERPROFILE%\.fluxbase` directory (optional, removes configuration)

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

## Troubleshooting

### "command not found" Error

The binary isn't in your PATH. Either:

1. Move the binary to a directory in your PATH:
   ```bash
   sudo mv fluxbase /usr/local/bin/
   ```

2. Or add the installation directory to your PATH:
   ```bash
   # Add to ~/.bashrc or ~/.zshrc
   export PATH="$PATH:/path/to/fluxbase/directory"
   ```

### Permission Denied

If you get a permission error during installation:

```bash
# macOS/Linux: Install with sudo
sudo curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | sudo bash

# Or install to a user directory
curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash -s -- --prefix ~/.local
```

### macOS Gatekeeper Warning

If macOS blocks the binary ("cannot be opened because the developer cannot be verified"):

```bash
# Remove the quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/fluxbase
```

### Connectivity Issues

If commands fail with connection errors:

1. Check your server URL:
   ```bash
   fluxbase config view
   ```

2. Test connectivity:
   ```bash
   curl -v https://your-server.com/health
   ```

3. Enable debug mode for detailed output:
   ```bash
   fluxbase --debug auth status
   ```

## Next Steps

- [Getting Started](/cli/getting-started/) - Configure and authenticate
- [Command Reference](/cli/commands/) - Full command documentation
- [Configuration](/cli/configuration/) - Configuration options
- [Workflows](/cli/workflows/) - Common workflows and CI/CD integration

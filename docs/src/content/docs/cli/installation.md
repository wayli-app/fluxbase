---
title: CLI Installation
description: Install the Fluxbase CLI tool
---

The Fluxbase CLI provides command-line access to manage your Fluxbase platform, including functions, jobs, storage, AI chatbots, and more.

## Installation Methods

### From Source (Recommended for Development)

If you have Go installed, you can build the CLI from source:

```bash
# Clone the repository
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase

# Build and install
make cli-install
```

This installs the `fluxbase` command to `/usr/local/bin`.

### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/fluxbase-eu/fluxbase/releases).

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-darwin-arm64
chmod +x fluxbase-darwin-arm64
sudo mv fluxbase-darwin-arm64 /usr/local/bin/fluxbase

# macOS (Intel)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-darwin-amd64
chmod +x fluxbase-darwin-amd64
sudo mv fluxbase-darwin-amd64 /usr/local/bin/fluxbase

# Linux (x86_64)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-amd64
chmod +x fluxbase-linux-amd64
sudo mv fluxbase-linux-amd64 /usr/local/bin/fluxbase

# Linux (ARM64)
curl -LO https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-arm64
chmod +x fluxbase-linux-arm64
sudo mv fluxbase-linux-arm64 /usr/local/bin/fluxbase
```

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

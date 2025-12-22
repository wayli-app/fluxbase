#!/bin/bash
set -e

echo "🚀 Setting up Fluxbase development environment..."

# Fix Docker socket permissions (for docker-outside-of-docker)
if [ -S /var/run/docker.sock ]; then
  echo "🐳 Fixing Docker socket permissions..."
  DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
  if ! getent group docker > /dev/null 2>&1; then
    sudo groupadd -g "$DOCKER_GID" docker 2>/dev/null || sudo groupmod -g "$DOCKER_GID" docker 2>/dev/null || true
  fi
  sudo usermod -aG docker vscode 2>/dev/null || true
  # Also ensure socket is accessible (some hosts have restrictive permissions)
  sudo chmod 666 /var/run/docker.sock 2>/dev/null || true
  echo "✅ Docker socket permissions fixed"
fi

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL..."
until pg_isready -h postgres -U postgres; do
  sleep 1
done
echo "✅ PostgreSQL is ready"

# Create test database if it doesn't exist
echo "📊 Creating test database..."
PGPASSWORD=postgres psql -h postgres -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'fluxbase_test'" | grep -q 1 || \
  PGPASSWORD=postgres psql -h postgres -U postgres -c "CREATE DATABASE fluxbase_test;"
echo "✅ Test database ready"

# Install Go dependencies
echo "📦 Installing Go dependencies..."
cd /workspace
go mod download
go mod tidy

# Install Go tools (in case they're not in the image)
echo "🔧 Ensuring Go tools are installed..."
go install -v golang.org/x/tools/gopls@latest 2>/dev/null || true
go install -v github.com/go-delve/delve/cmd/dlv@latest 2>/dev/null || true
go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2 2>/dev/null || true
go install -v github.com/cosmtrek/air@latest 2>/dev/null || true
go install -v -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest 2>/dev/null || true

# Create .env file if it doesn't exist
if [ ! -f /workspace/.env ]; then
  echo "📝 Creating .env file from .env.example..."
  cp /workspace/.env.example /workspace/.env
  echo "✅ .env file created"
fi

# Create storage directory
echo "📁 Creating storage directory..."
mkdir -p /workspace/storage
echo "✅ Storage directory ready"

# Create SDK symlinks for edge functions
echo "🔗 Creating SDK symlinks for edge functions..."
sudo ln -sfn /workspace/sdk /fluxbase-sdk
sudo ln -sfn /workspace/sdk-react /fluxbase-sdk-react
echo "✅ SDK symlinks created at /fluxbase-sdk and /fluxbase-sdk-react"

# Verify Deno installation (should be installed in Dockerfile)
if command -v deno &> /dev/null; then
  echo "✅ Deno $(deno --version | head -n1) is available"
else
  echo "⚠️  Deno not found - edge functions bundling may fail"
fi

# Run migrations
echo "🗄️  Running database migrations..."
cd /workspace
make migrate-up || echo "⚠️  Migrations may have already been run"

# Install documentation dependencies
if [ -f /workspace/docs/package.json ]; then
  echo "📚 Installing documentation dependencies..."
  cd /workspace/docs
  npm install
  cd /workspace
  echo "✅ Documentation dependencies installed"
fi

# Build the project to verify everything works
echo "🔨 Building project..."
cd /workspace
go build -o /tmp/fluxbase cmd/fluxbase/main.go && rm /tmp/fluxbase
echo "✅ Project builds successfully"

# Build and install the Fluxbase CLI
echo "🛠️  Building Fluxbase CLI..."
cd /workspace
go build -ldflags="-X github.com/fluxbase-eu/fluxbase/cli/cmd.Version=dev" -o /go/bin/fluxbase-cli cli/main.go
echo "✅ CLI built successfully"

# Create symlinks for convenient CLI access
echo "🔗 Creating CLI symlinks..."
sudo ln -sf /go/bin/fluxbase-cli /usr/local/bin/fluxbase
sudo ln -sf /go/bin/fluxbase-cli /usr/local/bin/fb
echo "✅ CLI available as 'fluxbase' and 'fb' commands"

# Generate shell completions for zsh
echo "⌨️  Setting up shell completions..."
mkdir -p /home/vscode/.zsh/completions
/go/bin/fluxbase-cli completion zsh > /home/vscode/.zsh/completions/_fluxbase
/go/bin/fluxbase-cli completion zsh > /home/vscode/.zsh/completions/_fb

# Add completion setup to .zshrc if not already present
if ! grep -q "fluxbase completions" /home/vscode/.zshrc; then
  cat >> /home/vscode/.zshrc << 'EOF'

# Fluxbase CLI completions
fpath=(/home/vscode/.zsh/completions $fpath)
autoload -Uz compinit && compinit -u
EOF
fi
echo "✅ Shell completions configured"

# SQLTools configuration for PostgreSQL
echo "🔧 Configuring SQLTools..."
mkdir -p /home/vscode/.config/Code/User
cat > /home/vscode/.config/Code/User/settings.json << 'EOF'
{
  "sqltools.connections": [
    {
      "previewLimit": 50,
      "server": "postgres",
      "port": 5432,
      "driver": "PostgreSQL",
      "name": "Fluxbase Dev",
      "database": "fluxbase_dev",
      "username": "postgres",
      "password": "postgres"
    },
    {
      "previewLimit": 50,
      "server": "postgres",
      "port": 5432,
      "driver": "PostgreSQL",
      "name": "Fluxbase Test",
      "database": "fluxbase_test",
      "username": "postgres",
      "password": "postgres"
    }
  ]
}
EOF

echo ""
echo "✨ Development environment ready!"
echo ""
echo "📝 Quick Start:"
echo "  - Run app with hot-reload: make dev"
echo "  - Run tests: make test"
echo "  - View docs: make docs-dev"
echo "  - Run database migrations: make migrate-up"
echo ""
echo "🖥️  CLI Commands (use 'fluxbase' or 'fb'):"
echo "  - fluxbase auth login      # Authenticate with server"
echo "  - fluxbase functions list  # List edge functions"
echo "  - fluxbase jobs list       # List background jobs"
echo "  - fluxbase --help          # See all commands"
echo ""
echo "🔗 Services:"
echo "  - Fluxbase API: http://localhost:8080"
echo "  - Admin UI: http://localhost:5050/admin/"
echo "  - MailHog: http://localhost:8025"
echo "  - MinIO Console: http://localhost:9001"
echo "  - Documentation: http://localhost:4321 (when running)"
echo ""
echo "💡 Tips:"
echo "  - Use 'make help' to see all available commands"
echo "  - Rebuild CLI after changes: make cli && sudo cp build/fluxbase /usr/local/bin/fluxbase"
echo "  - Read .claude/instructions.md for development guidelines"
echo ""

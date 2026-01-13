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
go install -v github.com/vladopajic/go-test-coverage/v2@latest 2>/dev/null || true

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

# Install Claude Code CLI
echo "🤖 Installing Claude Code CLI..."
sudo npm install -g @anthropic-ai/claude-code 2>/dev/null || true
if command -v claude &> /dev/null; then
  echo "✅ Claude Code CLI installed"
else
  echo "⚠️  Claude Code CLI installation failed - you can install it manually with: npm install -g @anthropic-ai/claude-code"
fi

# Configure Claude MCP server for Fluxbase
# The .mcp.json file uses FLUXBASE_SERVICE_ROLE_KEY which is generated at runtime
# We need to add a helper script to fetch and configure the key when Fluxbase is running
echo "🔌 Configuring Claude MCP server..."
mkdir -p /home/vscode/.local/bin/
cat > /home/vscode/.local/bin/configure-claude-mcp << 'SCRIPT'
#!/bin/bash
# Configure Claude MCP server with Fluxbase service role key
# Run this after starting Fluxbase with 'make dev'

set -e

# Load .env file if it exists (export all variables)
if [ -f /workspace/.env ]; then
  set -a  # automatically export all variables
  source /workspace/.env
  set +a
fi

# Check if Fluxbase is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
  echo "❌ Fluxbase is not running. Start it with 'make dev' first."
  exit 1
fi

# Check if MCP is enabled
MCP_HEALTH=$(curl -s http://localhost:8080/mcp/health 2>/dev/null || echo '{"status":"not found"}')
if echo "$MCP_HEALTH" | grep -q '"status":"not found"'; then
  echo "❌ MCP server is not enabled. Enable it in fluxbase.yaml:"
  echo "   mcp:"
  echo "     enabled: true"
  exit 1
fi

# Get the service role key from environment
if [ -z "$FLUXBASE_SERVICE_ROLE_KEY" ]; then
  echo "⚠️  FLUXBASE_SERVICE_ROLE_KEY not found."
  echo ""
  echo "   Add it to your .env file:"
  echo "   FLUXBASE_SERVICE_ROLE_KEY=your-jwt-token"
  echo ""
  echo "   Or generate one with the Fluxbase CLI:"
  echo "   fluxbase auth generate-service-key"
  exit 1
fi

# Configure Claude MCP server using the CLI
if command -v claude &> /dev/null; then
  echo "🔧 Configuring Claude MCP server..."

  # Remove existing fluxbase server if it exists
  claude mcp remove fluxbase 2>/dev/null || true

  # Add the Fluxbase MCP server with HTTP transport
  claude mcp add --transport http fluxbase http://localhost:8080/mcp \
    --header "Authorization: Bearer $FLUXBASE_SERVICE_ROLE_KEY"

  echo "✅ Claude MCP server configured successfully!"
  echo ""
  echo "   You can now use Claude Code to interact with Fluxbase."
  echo "   Try: 'claude' and ask about your database tables."
else
  echo "⚠️  Claude CLI not found. Install with: npm install -g @anthropic-ai/claude-code"
fi
SCRIPT
mkdir -p /home/vscode/.local/bin
chmod +x /home/vscode/.local/bin/configure-claude-mcp
echo "✅ Claude MCP configuration helper installed (run 'configure-claude-mcp' after starting Fluxbase)"

# Set up git pre-commit hook
echo "🪝 Setting up git pre-commit hook..."
cat > /workspace/.git/hooks/pre-commit << 'HOOK'
#!/bin/bash

# Pre-commit hook for Fluxbase
# Runs linting checks before allowing commits

set -e

echo "Running pre-commit checks..."

# Run golangci-lint on Go files
echo "Running golangci-lint..."
if ! golangci-lint run ./...; then
    echo ""
    echo "❌ golangci-lint failed. Please fix the issues above before committing."
    exit 1
fi

echo "✅ All pre-commit checks passed!"
HOOK
chmod +x /workspace/.git/hooks/pre-commit
echo "✅ Git pre-commit hook configured"

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
echo "🤖 AI Assistant:"
echo "  - Claude Code CLI: claude"
echo "  - Claude VSCode extension is pre-installed"
echo "  - Configure MCP: configure-claude-mcp (after 'make dev')"
echo ""
echo "💡 Tips:"
echo "  - Use 'make help' to see all available commands"
echo "  - Rebuild CLI after changes: make cli && sudo cp build/fluxbase /usr/local/bin/fluxbase"
echo "  - Read .claude/instructions.md for development guidelines"
echo "  - Pre-commit hook runs golangci-lint automatically"
echo ""

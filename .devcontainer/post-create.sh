#!/bin/bash
set -e

echo "🚀 Setting up Fluxbase development environment..."

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL..."
until pg_isready -h postgres -U postgres; do
  sleep 1
done
echo "✅ PostgreSQL is ready"

# Create test database if it doesn't exist
echo "📊 Creating test database..."
psql -h postgres -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'fluxbase_test'" | grep -q 1 || \
  psql -h postgres -U postgres -c "CREATE DATABASE fluxbase_test;"
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
go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || true
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
echo "🔗 Services:"
echo "  - Fluxbase API: http://localhost:8080"
echo "  - pgAdmin: http://localhost:5050"
echo "  - MailHog: http://localhost:8025"
echo "  - Documentation: http://localhost:3000 (when running)"
echo ""
echo "💡 Tips:"
echo "  - Use 'make help' to see all available commands"
echo "  - Check TODO.md for the implementation plan"
echo "  - Read .claude/instructions.md for development guidelines"
echo ""

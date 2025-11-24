#!/bin/bash
# Generate and create a migration service key
# This script generates a secure random service key and inserts it into the database

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DB_HOST="${POSTGRES_HOST:-postgres}"
DB_PORT="${POSTGRES_PORT:-5432}"
DB_NAME="${POSTGRES_DB:-fluxbase_dev}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"

echo -e "${GREEN}=== Migration Service Key Generator ===${NC}"
echo ""

# Generate a secure random key
echo "Generating secure service key..."
SERVICE_KEY="sk_migrations_$(openssl rand -base64 32 | tr -d '=+/' | cut -c1-32)"
KEY_PREFIX="${SERVICE_KEY:0:16}"

echo -e "${GREEN}✓ Generated service key${NC}"
echo ""

# Ask for confirmation
echo -e "${YELLOW}This will create a new service key in the database.${NC}"
echo "Key prefix: ${KEY_PREFIX}..."
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# Insert into database
echo "Creating service key in database..."

PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v service_key="$SERVICE_KEY" \
    -v key_prefix="$KEY_PREFIX" \
    << EOF
-- Insert the service key
INSERT INTO auth.service_keys (
    name,
    description,
    key_hash,
    key_prefix,
    scopes,
    enabled,
    expires_at
) VALUES (
    'Migration Service Key',
    'Dedicated service key for executing database migrations from application container',
    crypt(:'service_key', gen_salt('bf')),
    :'key_prefix',
    ARRAY['*'],
    true,
    NOW() + INTERVAL '1 year'
)
RETURNING
    id,
    name,
    key_prefix,
    scopes,
    enabled,
    created_at,
    expires_at;
EOF

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ Service key created successfully!${NC}"
    echo ""
    echo -e "${YELLOW}=== IMPORTANT: Save this key securely ===${NC}"
    echo ""
    echo "Service Key:"
    echo -e "${GREEN}${SERVICE_KEY}${NC}"
    echo ""
    echo "Add to your application environment variables:"
    echo "FLUXBASE_MIGRATIONS_SERVICE_KEY=${SERVICE_KEY}"
    echo ""
    echo "Add to your Fluxbase configuration:"
    echo "FLUXBASE_MIGRATIONS_ENABLED=true"
    echo "FLUXBASE_MIGRATIONS_REQUIRE_SERVICE_KEY=true"
    echo "FLUXBASE_MIGRATIONS_ALLOWED_IP_RANGES=172.16.0.0/12,10.0.0.0/8"
    echo ""
    echo -e "${RED}WARNING: This key cannot be recovered. Store it securely!${NC}"
    echo "The key will expire after 1 year."
    echo ""
else
    echo -e "${RED}✗ Failed to create service key${NC}"
    exit 1
fi

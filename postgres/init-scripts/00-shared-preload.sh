#!/bin/bash
# Configure shared_preload_libraries for extensions that require preloading
# This script runs during PostgreSQL initialization

set -e

cat >> "$PGDATA/postgresql.conf" << EOF

# =============================================================================
# Fluxbase Extension Configuration
# =============================================================================

# Preload extensions that require shared memory allocation
shared_preload_libraries = 'pg_stat_statements,pg_cron,pgaudit'

# pg_stat_statements configuration
pg_stat_statements.track = all
pg_stat_statements.max = 10000

# pg_cron configuration
cron.database_name = '${POSTGRES_DB:-postgres}'

# pgaudit configuration (disabled by default, enable via SQL)
pgaudit.log = 'none'
EOF

echo "Fluxbase: Configured shared_preload_libraries in postgresql.conf"

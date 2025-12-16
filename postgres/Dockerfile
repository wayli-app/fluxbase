# Fluxbase Postgres - PostgreSQL with popular extensions pre-installed
# https://github.com/fluxbase-eu/fluxbase
#
# Build args:
#   POSTGRES_VERSION - Full version (e.g., 18.1)
#   POSTGRES_MAJOR   - Major version (e.g., 18)
#
# Usage:
#   docker build --build-arg POSTGRES_VERSION=18.1 --build-arg POSTGRES_MAJOR=18 -t fluxbase-postgres:18.1 .

ARG POSTGRES_VERSION=18.1
FROM postgres:${POSTGRES_VERSION}

ARG POSTGRES_MAJOR=18

LABEL org.opencontainers.image.source="https://github.com/fluxbase-eu/fluxbase"
LABEL org.opencontainers.image.description="PostgreSQL ${POSTGRES_MAJOR} with extensions for Fluxbase"
LABEL org.opencontainers.image.licenses="MIT"

# Add PGDG repo for latest extension packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/pgdg.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/pgdg.gpg] http://apt.postgresql.org/pub/repos/apt $(. /etc/os-release && echo $VERSION_CODENAME)-pgdg main" \
       > /etc/apt/sources.list.d/pgdg.list \
    && rm -rf /var/lib/apt/lists/*

# Install extension packages (version-specific)
# These extensions require external packages to be installed
RUN apt-get update && apt-get install -y --no-install-recommends \
    # PostGIS - Geospatial support (includes raster, topology, tiger geocoder, address_standardizer)
    postgresql-${POSTGRES_MAJOR}-postgis-3 \
    postgresql-${POSTGRES_MAJOR}-postgis-3-scripts \
    # pgvector - Vector similarity search for AI/ML
    postgresql-${POSTGRES_MAJOR}-pgvector \
    # pg_cron - Job scheduling
    postgresql-${POSTGRES_MAJOR}-cron \
    # pgsql-http - HTTP client for PostgreSQL
    postgresql-${POSTGRES_MAJOR}-http \
    # pgTAP - Unit testing framework
    postgresql-${POSTGRES_MAJOR}-pgtap \
    # pgAudit - Audit logging
    postgresql-${POSTGRES_MAJOR}-pgaudit \
    # pg_repack - Table reorganization without locks
    postgresql-${POSTGRES_MAJOR}-repack \
    # pgRouting - Geospatial routing
    postgresql-${POSTGRES_MAJOR}-pgrouting \
    # HypoPG - Hypothetical indexes
    postgresql-${POSTGRES_MAJOR}-hypopg \
    # RUM - RUM index access method
    postgresql-${POSTGRES_MAJOR}-rum \
    && rm -rf /var/lib/apt/lists/*

# Built-in contrib extensions (no install needed, just CREATE EXTENSION):
# - uuid-ossp       : UUID generation functions
# - pgcrypto        : Cryptographic functions
# - pg_trgm         : Trigram text similarity
# - btree_gin       : GIN index support for btree types
# - btree_gist      : GiST index support for btree types
# - hstore          : Key-value data type
# - ltree           : Hierarchical tree-like data type
# - citext          : Case-insensitive text type
# - unaccent        : Text search dictionary for accent removal
# - pg_stat_statements : Query performance statistics

# Copy initialization scripts
COPY init-scripts/ /docker-entrypoint-initdb.d/

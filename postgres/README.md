# Fluxbase Postgres

Custom PostgreSQL Docker image with 48+ extensions pre-installed for Fluxbase.

## Pre-installed Extensions

### Via PGDG apt packages

| Extension                | Description                        |
| ------------------------ | ---------------------------------- |
| `postgis`                | Geospatial objects and queries     |
| `postgis_raster`         | Raster data support                |
| `postgis_topology`       | Topology support                   |
| `postgis_tiger_geocoder` | US Census TIGER geocoding          |
| `address_standardizer`   | Address parsing                    |
| `vector`                 | Vector similarity search for AI/ML |
| `pg_cron`                | Job scheduling inside PostgreSQL   |
| `http`                   | HTTP client for PostgreSQL         |
| `pgtap`                  | Unit testing framework             |
| `pgaudit`                | Audit logging                      |
| `pg_repack`              | Table reorganization without locks |
| `pgrouting`              | Geospatial routing                 |
| `hypopg`                 | Hypothetical indexes               |
| `rum`                    | RUM index access method            |

### Built-in contrib extensions

| Extension            | Category     | Description                    |
| -------------------- | ------------ | ------------------------------ |
| `uuid-ossp`          | Core         | UUID generation functions      |
| `pgcrypto`           | Core         | Cryptographic functions        |
| `pg_trgm`            | Core         | Trigram text similarity        |
| `btree_gin`          | Indexing     | GIN index support              |
| `btree_gist`         | Indexing     | GiST index support             |
| `hstore`             | Data Types   | Key-value data type            |
| `ltree`              | Data Types   | Hierarchical tree-like data    |
| `citext`             | Data Types   | Case-insensitive text          |
| `unaccent`           | Text Search  | Accent removal for text search |
| `pg_stat_statements` | Monitoring   | Query performance statistics   |
| `intarray`           | Data Types   | Integer array functions        |
| `seg`                | Data Types   | Line segment type              |
| `isn`                | Data Types   | ISBN/ISSN/EAN types            |
| `cube`               | Data Types   | Multi-dimensional cube         |
| `dict_int`           | Text Search  | Integer dictionary             |
| `dict_xsyn`          | Text Search  | Synonym dictionary             |
| `fuzzystrmatch`      | Text Search  | Fuzzy string matching          |
| `earthdistance`      | Geospatial   | Earth distance calculations    |
| `bloom`              | Indexing     | Bloom filter indexes           |
| `postgres_fdw`       | Foreign Data | PostgreSQL FDW                 |
| `dblink`             | Foreign Data | Cross-database queries         |
| `tablefunc`          | Utilities    | Crosstab/pivot functions       |
| `sslinfo`            | Utilities    | SSL connection info            |
| `autoinc`            | Triggers     | Auto-increment triggers        |
| `insert_username`    | Triggers     | Username tracking              |
| `moddatetime`        | Triggers     | Modification timestamps        |
| `refint`             | Triggers     | Referential integrity          |
| `tcn`                | Triggers     | Change notifications           |
| `tsm_system_rows`    | Sampling     | Row-based sampling             |
| `tsm_system_time`    | Sampling     | Time-based sampling            |
| `pgstattuple`        | Monitoring   | Tuple-level statistics         |
| `pgrowlocks`         | Monitoring   | Row lock info                  |
| `pg_prewarm`         | Performance  | Buffer cache warming           |
| `pg_walinspect`      | Monitoring   | WAL inspection                 |

## Version Management

Supported PostgreSQL versions are defined in `versions.json`:

```json
{
  "versions": [
    { "major": 18, "version": "18.1", "imageVersion": "1", "latest": true }
  ]
}
```

- **major**: PostgreSQL major version (used for apt package names)
- **version**: Full version string (used for base image tag)
- **imageVersion**: Independent image version (increment when updating extensions or Dockerfile without changing Postgres version)
- **latest**: If true, this version gets the `:latest` tag

## Building Locally

```bash
# Build for Postgres 18.1
docker build \
  --build-arg POSTGRES_VERSION=18.1 \
  --build-arg POSTGRES_MAJOR=18 \
  -t fluxbase-postgres:18.1 .
```

## Image Tags

Images are published to `ghcr.io/fluxbase-eu/fluxbase-postgres`:

- `:18.1` - Specific PostgreSQL version (rolling, updates with new imageVersion)
- `:18.1-v1` - Immutable tag (PostgreSQL version + image version)
- `:18` - Latest 18.x (rolling)
- `:latest` - Latest stable (version marked with `latest: true`)

## Enabling Extensions

Extensions are installed but not enabled by default. Enable them via the Fluxbase Admin UI or with SQL:

```sql
-- Core extensions (enabled by Fluxbase automatically)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Optional extensions (enable via Fluxbase Admin UI or API)
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_cron";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pgaudit";
CREATE EXTENSION IF NOT EXISTS "hypopg";
-- ... and many more
```

## Shared Preload Libraries

The following extensions are preloaded at startup (configured in `init-scripts/00-shared-preload.sh`):

- `pg_stat_statements` - Query statistics
- `pg_cron` - Job scheduling
- `pgaudit` - Audit logging (disabled by default, enable via SQL)

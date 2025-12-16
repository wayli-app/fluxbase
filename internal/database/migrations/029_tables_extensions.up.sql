--
-- EXTENSIONS SCHEMA TABLES
-- Track available and enabled PostgreSQL extensions
--

-- Available extensions catalog (populated by Fluxbase)
CREATE TABLE IF NOT EXISTS dashboard.available_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL CHECK (category IN (
        'core', 'geospatial', 'ai_ml', 'monitoring', 'scheduling',
        'data_types', 'text_search', 'indexing', 'networking', 'testing',
        'maintenance', 'performance', 'foreign_data', 'triggers', 'sampling', 'utilities'
    )),
    is_core BOOLEAN DEFAULT false,
    requires_restart BOOLEAN DEFAULT false,
    documentation_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_available_extensions_category ON dashboard.available_extensions(category);
CREATE INDEX IF NOT EXISTS idx_available_extensions_is_core ON dashboard.available_extensions(is_core);

COMMENT ON TABLE dashboard.available_extensions IS 'Catalog of PostgreSQL extensions available in Fluxbase';
COMMENT ON COLUMN dashboard.available_extensions.name IS 'PostgreSQL extension name used in CREATE EXTENSION';
COMMENT ON COLUMN dashboard.available_extensions.is_core IS 'Core extensions are always enabled and cannot be disabled';
COMMENT ON COLUMN dashboard.available_extensions.requires_restart IS 'Extension requires PostgreSQL restart after enabling';

-- Track enabled extensions
CREATE TABLE IF NOT EXISTS dashboard.enabled_extensions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extension_name TEXT NOT NULL REFERENCES dashboard.available_extensions(name) ON DELETE CASCADE,
    enabled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enabled_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    disabled_at TIMESTAMPTZ,
    disabled_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Only one active record per extension
CREATE UNIQUE INDEX IF NOT EXISTS idx_enabled_extensions_active ON dashboard.enabled_extensions(extension_name) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_enabled_extensions_name ON dashboard.enabled_extensions(extension_name);

COMMENT ON TABLE dashboard.enabled_extensions IS 'Tracks which extensions are currently enabled';
COMMENT ON COLUMN dashboard.enabled_extensions.is_active IS 'Whether this extension is currently enabled';
COMMENT ON COLUMN dashboard.enabled_extensions.error_message IS 'Error message if enabling/disabling failed';

-- =============================================================================
-- CORE EXTENSIONS (always enabled, cannot be disabled)
-- =============================================================================
INSERT INTO dashboard.available_extensions (name, display_name, description, category, is_core, requires_restart) VALUES
    ('uuid-ossp', 'UUID Functions', 'UUID generation functions including uuid_generate_v4()', 'core', true, false),
    ('pgcrypto', 'Cryptographic Functions', 'Cryptographic functions including gen_random_uuid() and password hashing', 'core', true, false),
    ('pg_trgm', 'Trigram Text Search', 'Trigram-based text similarity and indexing for fuzzy search', 'core', true, false),
    ('btree_gin', 'BTree GIN Index', 'GIN index support for btree-indexable data types', 'core', true, false)
ON CONFLICT (name) DO NOTHING;

-- =============================================================================
-- PGDG EXTENSIONS (installed via apt packages)
-- =============================================================================
INSERT INTO dashboard.available_extensions (name, display_name, description, category, is_core, requires_restart, documentation_url) VALUES
    -- Geospatial
    ('postgis', 'PostGIS', 'Geographic objects and spatial queries for location-based applications', 'geospatial', false, false, 'https://postgis.net/docs/'),
    ('postgis_topology', 'PostGIS Topology', 'Topology support for PostGIS spatial data', 'geospatial', false, false, 'https://postgis.net/docs/Topology.html'),
    ('postgis_raster', 'PostGIS Raster', 'Raster data support for PostGIS', 'geospatial', false, false, 'https://postgis.net/docs/RT_reference.html'),
    ('postgis_tiger_geocoder', 'Tiger Geocoder', 'US Census TIGER data geocoding', 'geospatial', false, false, 'https://postgis.net/docs/Extras.html#Tiger_Geocoder'),
    ('address_standardizer', 'Address Standardizer', 'Parse and standardize address strings', 'geospatial', false, false, 'https://postgis.net/docs/Extras.html#Address_Standardizer'),
    ('address_standardizer_data_us', 'US Address Data', 'US address standardization reference data', 'geospatial', false, false, 'https://postgis.net/docs/Extras.html#Address_Standardizer'),
    ('pgrouting', 'pgRouting', 'Geospatial routing and network analysis', 'geospatial', false, false, 'https://pgrouting.org/'),
    -- AI/ML
    ('vector', 'pgvector', 'Vector similarity search for AI/ML embeddings and RAG applications', 'ai_ml', false, false, 'https://github.com/pgvector/pgvector'),
    -- Monitoring
    ('pg_stat_statements', 'Query Statistics', 'Track planning and execution statistics of SQL statements', 'monitoring', false, true, 'https://www.postgresql.org/docs/current/pgstatstatements.html'),
    ('pgaudit', 'Audit Logging', 'Session and object audit logging for compliance', 'monitoring', false, true, 'https://www.pgaudit.org/'),
    -- Scheduling
    ('pg_cron', 'Cron Scheduler', 'Run periodic jobs inside PostgreSQL using cron syntax', 'scheduling', false, true, 'https://github.com/citusdata/pg_cron'),
    -- Networking
    ('http', 'HTTP Client', 'Make HTTP requests from SQL for webhooks and API calls', 'networking', false, false, 'https://github.com/pramsey/pgsql-http'),
    -- Testing
    ('pgtap', 'pgTAP', 'Unit testing framework for PostgreSQL', 'testing', false, false, 'https://pgtap.org/'),
    -- Maintenance
    ('pg_repack', 'Table Repack', 'Reorganize tables without exclusive locks', 'maintenance', false, false, 'https://reorg.github.io/pg_repack/'),
    -- Performance
    ('hypopg', 'Hypothetical Indexes', 'Create hypothetical indexes to test query plans', 'performance', false, false, 'https://hypopg.readthedocs.io/'),
    -- Indexing
    ('rum', 'RUM Index', 'RUM index access method for full-text search', 'indexing', false, false, 'https://github.com/postgrespro/rum')
ON CONFLICT (name) DO NOTHING;

-- =============================================================================
-- POSTGRESQL CONTRIB EXTENSIONS (built-in, no apt install needed)
-- =============================================================================
INSERT INTO dashboard.available_extensions (name, display_name, description, category, is_core, requires_restart, documentation_url) VALUES
    -- Data Types
    ('hstore', 'HStore', 'Key-value data type for storing sets of key/value pairs', 'data_types', false, false, 'https://www.postgresql.org/docs/current/hstore.html'),
    ('ltree', 'Ltree', 'Hierarchical tree-like data type for representing label paths', 'data_types', false, false, 'https://www.postgresql.org/docs/current/ltree.html'),
    ('citext', 'Case-Insensitive Text', 'Case-insensitive character string type', 'data_types', false, false, 'https://www.postgresql.org/docs/current/citext.html'),
    ('intarray', 'Integer Arrays', 'Functions and operators for integer arrays', 'data_types', false, false, 'https://www.postgresql.org/docs/current/intarray.html'),
    ('seg', 'Line Segments', 'Data type for line segments or floating-point intervals', 'data_types', false, false, 'https://www.postgresql.org/docs/current/seg.html'),
    ('isn', 'ISN Types', 'Data types for ISBN, ISSN, EAN, and other product codes', 'data_types', false, false, 'https://www.postgresql.org/docs/current/isn.html'),
    ('cube', 'Cube', 'Multi-dimensional cube data type', 'data_types', false, false, 'https://www.postgresql.org/docs/current/cube.html'),
    -- Text Search
    ('unaccent', 'Unaccent', 'Text search dictionary for removing accents from characters', 'text_search', false, false, 'https://www.postgresql.org/docs/current/unaccent.html'),
    ('dict_int', 'Integer Dictionary', 'Text search dictionary for integers', 'text_search', false, false, 'https://www.postgresql.org/docs/current/dict-int.html'),
    ('dict_xsyn', 'Synonym Dictionary', 'Extended synonym dictionary for text search', 'text_search', false, false, 'https://www.postgresql.org/docs/current/dict-xsyn.html'),
    ('fuzzystrmatch', 'Fuzzy String Match', 'Functions for fuzzy string matching (soundex, levenshtein)', 'text_search', false, false, 'https://www.postgresql.org/docs/current/fuzzystrmatch.html'),
    -- Indexing
    ('btree_gist', 'BTree GiST Index', 'GiST index support for btree-indexable types and exclusion constraints', 'indexing', false, false, 'https://www.postgresql.org/docs/current/btree-gist.html'),
    ('bloom', 'Bloom Filter', 'Bloom filter index access method', 'indexing', false, false, 'https://www.postgresql.org/docs/current/bloom.html'),
    -- Geospatial (contrib)
    ('earthdistance', 'Earth Distance', 'Calculate great-circle distances on Earth', 'geospatial', false, false, 'https://www.postgresql.org/docs/current/earthdistance.html'),
    -- Foreign Data
    ('postgres_fdw', 'PostgreSQL FDW', 'Foreign data wrapper for remote PostgreSQL servers', 'foreign_data', false, false, 'https://www.postgresql.org/docs/current/postgres-fdw.html'),
    ('dblink', 'DB Link', 'Connect to other PostgreSQL databases within queries', 'foreign_data', false, false, 'https://www.postgresql.org/docs/current/dblink.html'),
    -- Utilities
    ('tablefunc', 'Table Functions', 'Functions for crosstab and pivot tables', 'utilities', false, false, 'https://www.postgresql.org/docs/current/tablefunc.html'),
    ('sslinfo', 'SSL Info', 'Information about SSL certificates of connections', 'utilities', false, false, 'https://www.postgresql.org/docs/current/sslinfo.html'),
    -- Triggers
    ('autoinc', 'Auto Increment', 'Functions for auto-incrementing fields', 'triggers', false, false, 'https://www.postgresql.org/docs/current/contrib-spi.html'),
    ('insert_username', 'Insert Username', 'Track which user inserted a row', 'triggers', false, false, 'https://www.postgresql.org/docs/current/contrib-spi.html'),
    ('moddatetime', 'Mod Datetime', 'Track modification timestamps automatically', 'triggers', false, false, 'https://www.postgresql.org/docs/current/contrib-spi.html'),
    ('refint', 'Referential Integrity', 'Functions for implementing referential integrity', 'triggers', false, false, 'https://www.postgresql.org/docs/current/contrib-spi.html'),
    ('tcn', 'Change Notifications', 'Triggered change notifications via NOTIFY', 'triggers', false, false, 'https://www.postgresql.org/docs/current/tcn.html'),
    -- Sampling
    ('tsm_system_rows', 'Row Sampling', 'TABLESAMPLE method based on row count', 'sampling', false, false, 'https://www.postgresql.org/docs/current/tsm-system-rows.html'),
    ('tsm_system_time', 'Time Sampling', 'TABLESAMPLE method based on time spent', 'sampling', false, false, 'https://www.postgresql.org/docs/current/tsm-system-time.html'),
    -- Monitoring (contrib)
    ('pgstattuple', 'Tuple Statistics', 'Show tuple-level statistics for tables', 'monitoring', false, false, 'https://www.postgresql.org/docs/current/pgstattuple.html'),
    ('pgrowlocks', 'Row Locks', 'Show row-level locking information', 'monitoring', false, false, 'https://www.postgresql.org/docs/current/pgrowlocks.html'),
    ('pg_walinspect', 'WAL Inspect', 'Functions to inspect WAL contents', 'monitoring', false, false, 'https://www.postgresql.org/docs/current/pgwalinspect.html'),
    -- Performance (contrib)
    ('pg_prewarm', 'Buffer Prewarm', 'Prewarm relation data into buffer cache', 'performance', false, false, 'https://www.postgresql.org/docs/current/pgprewarm.html')
ON CONFLICT (name) DO NOTHING;

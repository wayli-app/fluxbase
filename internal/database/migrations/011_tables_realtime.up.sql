--
-- REALTIME SCHEMA TABLES
-- Realtime subscriptions and change tracking
--

-- Realtime schema registry table
CREATE TABLE IF NOT EXISTS realtime.schema_registry (
    id SERIAL PRIMARY KEY,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    realtime_enabled BOOLEAN DEFAULT true,
    events TEXT[] DEFAULT ARRAY['INSERT', 'UPDATE', 'DELETE'],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schema_name, table_name)
);

-- Register functions.execution_logs for realtime (table created in functions migration)
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES ('functions', 'execution_logs', true, ARRAY['INSERT'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events;

-- Register jobs schema tables for realtime (tables created in jobs migration)
-- Note: jobs.functions is excluded because code fields exceed pg_notify's 8KB limit
INSERT INTO realtime.schema_registry (schema_name, table_name, realtime_enabled, events)
VALUES
    ('jobs', 'queue', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'workers', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'function_files', true, ARRAY['INSERT', 'UPDATE', 'DELETE']),
    ('jobs', 'execution_logs', true, ARRAY['INSERT'])
ON CONFLICT (schema_name, table_name) DO UPDATE
SET realtime_enabled = true,
    events = EXCLUDED.events;

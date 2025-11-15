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

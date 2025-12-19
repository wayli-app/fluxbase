-- Add support for custom log categories
-- This migration adds a custom_category column and a new partition for custom categories

-- Add custom_category column to the parent table
-- This column stores the user-defined category name when category='custom'
ALTER TABLE logging.entries ADD COLUMN IF NOT EXISTS custom_category TEXT;

-- Drop the existing category constraint
ALTER TABLE logging.entries DROP CONSTRAINT IF EXISTS valid_category;

-- Add updated category constraint that includes 'custom'
ALTER TABLE logging.entries ADD CONSTRAINT valid_category
    CHECK (category IN ('system', 'http', 'security', 'execution', 'ai', 'custom'));

-- Create the custom partition for user-defined categories
CREATE TABLE IF NOT EXISTS logging.entries_custom
    PARTITION OF logging.entries FOR VALUES IN ('custom');

-- Index for custom category lookups within the custom partition
CREATE INDEX IF NOT EXISTS idx_logging_entries_custom_category
    ON logging.entries (custom_category)
    WHERE custom_category IS NOT NULL;

-- Comment on the new column
COMMENT ON COLUMN logging.entries.custom_category IS 'User-defined category name when category=custom';

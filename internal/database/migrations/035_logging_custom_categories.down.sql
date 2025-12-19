-- Revert custom log categories support

-- Drop the custom category index
DROP INDEX IF EXISTS logging.idx_logging_entries_custom_category;

-- Drop the custom partition (will delete all custom category logs)
DROP TABLE IF EXISTS logging.entries_custom;

-- Drop the updated category constraint
ALTER TABLE logging.entries DROP CONSTRAINT IF EXISTS valid_category;

-- Restore the original category constraint
ALTER TABLE logging.entries ADD CONSTRAINT valid_category
    CHECK (category IN ('system', 'http', 'security', 'execution', 'ai'));

-- Remove the custom_category column
ALTER TABLE logging.entries DROP COLUMN IF EXISTS custom_category;

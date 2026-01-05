-- Remove read_only column from ai.providers

ALTER TABLE ai.providers
DROP COLUMN IF EXISTS read_only;

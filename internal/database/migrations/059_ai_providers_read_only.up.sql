-- Add read_only column to ai.providers
-- This column indicates providers configured via environment/YAML that cannot be modified via API

ALTER TABLE ai.providers
ADD COLUMN IF NOT EXISTS read_only BOOLEAN DEFAULT false;

COMMENT ON COLUMN ai.providers.read_only IS
'True if provider is configured via environment/YAML and cannot be modified via API';

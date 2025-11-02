-- Rollback: Remove Webhook Triggering System

-- Remove webhook triggers from tables
SELECT auth.remove_webhook_trigger('auth', 'users');

-- Drop helper functions
DROP FUNCTION IF EXISTS auth.remove_webhook_trigger(TEXT, TEXT);
DROP FUNCTION IF EXISTS auth.create_webhook_trigger(TEXT, TEXT);
DROP FUNCTION IF EXISTS auth.queue_webhook_event();

-- Drop webhook_events table
DROP TABLE IF EXISTS auth.webhook_events CASCADE;

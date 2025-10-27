-- Drop webhooks tables
DROP TRIGGER IF EXISTS webhook_updated_at ON auth.webhooks;
DROP FUNCTION IF EXISTS auth.update_webhook_updated_at();

DROP INDEX IF EXISTS auth.idx_webhook_deliveries_created_at;
DROP INDEX IF EXISTS auth.idx_webhook_deliveries_next_retry;
DROP INDEX IF EXISTS auth.idx_webhook_deliveries_status;
DROP INDEX IF EXISTS auth.idx_webhook_deliveries_webhook_id;

DROP TABLE IF EXISTS auth.webhook_deliveries;
DROP TABLE IF EXISTS auth.webhooks;

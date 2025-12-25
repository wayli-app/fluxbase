-- Add disable_execution_logs column to all resource types
-- This allows users to suppress execution log creation via @fluxbase:disable-execution-logs annotation

-- Edge Functions
ALTER TABLE functions.edge_functions
    ADD COLUMN IF NOT EXISTS disable_execution_logs BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN functions.edge_functions.disable_execution_logs IS 'When true, execution logs are not created for this function (from @fluxbase:disable-execution-logs annotation)';

-- Jobs
ALTER TABLE jobs.functions
    ADD COLUMN IF NOT EXISTS disable_execution_logs BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN jobs.functions.disable_execution_logs IS 'When true, execution logs are not created for this job (from @fluxbase:disable-execution-logs annotation)';

-- RPC Procedures
ALTER TABLE rpc.procedures
    ADD COLUMN IF NOT EXISTS disable_execution_logs BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN rpc.procedures.disable_execution_logs IS 'When true, execution logs are not created for this procedure (from @fluxbase:disable-execution-logs annotation)';

-- AI Chatbots
ALTER TABLE ai.chatbots
    ADD COLUMN IF NOT EXISTS disable_execution_logs BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN ai.chatbots.disable_execution_logs IS 'When true, execution logs are not created for this chatbot (from @fluxbase:disable-execution-logs annotation)';

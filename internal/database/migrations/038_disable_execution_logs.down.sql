-- Remove disable_execution_logs columns

ALTER TABLE functions.edge_functions DROP COLUMN IF EXISTS disable_execution_logs;
ALTER TABLE jobs.functions DROP COLUMN IF EXISTS disable_execution_logs;
ALTER TABLE rpc.procedures DROP COLUMN IF EXISTS disable_execution_logs;
ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS disable_execution_logs;

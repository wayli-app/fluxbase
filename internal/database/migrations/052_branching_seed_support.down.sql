-- Rollback seed data support from branching feature

-- Drop seed execution tracking table
DROP TABLE IF EXISTS branching.seed_execution_log;

-- Restore activity_log constraint without 'seeding' action
ALTER TABLE branching.activity_log
DROP CONSTRAINT IF EXISTS activity_log_action_check;

ALTER TABLE branching.activity_log
ADD CONSTRAINT activity_log_action_check
CHECK (action IN ('created', 'cloned', 'migrated', 'reset', 'deleted',
                  'status_changed', 'access_granted', 'access_revoked'));

-- Remove seeds_path column from branches table
ALTER TABLE branching.branches
DROP COLUMN IF EXISTS seeds_path;

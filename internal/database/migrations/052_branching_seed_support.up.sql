-- Add seed data support to branching feature

-- Add seeds_path column to branches table to store custom seed directory per branch
ALTER TABLE branching.branches
ADD COLUMN seeds_path TEXT;

-- Update activity_log constraint to include 'seeding' action
ALTER TABLE branching.activity_log
DROP CONSTRAINT IF EXISTS activity_log_action_check;

ALTER TABLE branching.activity_log
ADD CONSTRAINT activity_log_action_check
CHECK (action IN ('created', 'cloned', 'migrated', 'reset', 'deleted',
                  'status_changed', 'access_granted', 'access_revoked', 'seeding'));

-- Create seed execution tracking table
CREATE TABLE branching.seed_execution_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id UUID REFERENCES branching.branches(id) ON DELETE CASCADE,
    seed_file_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('started', 'success', 'failed')),
    error_message TEXT,
    executed_at TIMESTAMPTZ DEFAULT NOW(),
    duration_ms INTEGER,
    CONSTRAINT seed_execution_unique UNIQUE (branch_id, seed_file_name)
);

CREATE INDEX idx_seed_execution_branch_id ON branching.seed_execution_log(branch_id);
CREATE INDEX idx_seed_execution_status ON branching.seed_execution_log(status);

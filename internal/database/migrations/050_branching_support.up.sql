-- Database branching support for isolated development/testing environments
CREATE SCHEMA IF NOT EXISTS branching;

-- Branch metadata table
CREATE TABLE branching.branches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    database_name TEXT NOT NULL,
    status TEXT DEFAULT 'creating' CHECK (status IN ('creating', 'ready', 'migrating', 'error', 'deleting', 'deleted')),
    type TEXT DEFAULT 'preview' CHECK (type IN ('main', 'preview', 'persistent')),
    parent_branch_id UUID REFERENCES branching.branches(id),
    data_clone_mode TEXT DEFAULT 'schema_only' CHECK (data_clone_mode IN ('schema_only', 'full_clone', 'seed_data')),
    github_pr_number INTEGER,
    github_pr_url TEXT,
    github_repo TEXT,
    error_message TEXT,
    created_by UUID REFERENCES auth.users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    CONSTRAINT branches_name_unique UNIQUE (name)
);

-- Track migration history per branch
CREATE TABLE branching.migration_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id UUID REFERENCES branching.branches(id) ON DELETE CASCADE,
    migration_version BIGINT NOT NULL,
    migration_name TEXT,
    applied_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT migration_history_unique UNIQUE (branch_id, migration_version)
);

-- Activity log for branch operations
CREATE TABLE branching.activity_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id UUID REFERENCES branching.branches(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN ('created', 'cloned', 'migrated', 'reset', 'deleted', 'status_changed')),
    status TEXT NOT NULL CHECK (status IN ('started', 'success', 'failed')),
    details JSONB,
    error_message TEXT,
    executed_by UUID REFERENCES auth.users(id),
    executed_at TIMESTAMPTZ DEFAULT NOW(),
    duration_ms INTEGER
);

-- GitHub integration configuration per repository
CREATE TABLE branching.github_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository TEXT UNIQUE NOT NULL,
    auto_create_on_pr BOOLEAN DEFAULT true,
    auto_delete_on_merge BOOLEAN DEFAULT true,
    default_data_clone_mode TEXT DEFAULT 'schema_only' CHECK (default_data_clone_mode IN ('schema_only', 'full_clone', 'seed_data')),
    webhook_secret TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Branch access permissions (for multi-tenant scenarios)
CREATE TABLE branching.branch_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id UUID REFERENCES branching.branches(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    access_level TEXT DEFAULT 'read' CHECK (access_level IN ('read', 'write', 'admin')),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    granted_by UUID REFERENCES auth.users(id),
    CONSTRAINT branch_access_unique UNIQUE (branch_id, user_id)
);

-- Indexes for common queries
CREATE INDEX idx_branches_status ON branching.branches(status);
CREATE INDEX idx_branches_type ON branching.branches(type);
CREATE INDEX idx_branches_created_by ON branching.branches(created_by);
CREATE INDEX idx_branches_github_pr ON branching.branches(github_repo, github_pr_number) WHERE github_pr_number IS NOT NULL;
CREATE INDEX idx_branches_expires_at ON branching.branches(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_activity_log_branch_id ON branching.activity_log(branch_id);
CREATE INDEX idx_activity_log_executed_at ON branching.activity_log(executed_at);
CREATE INDEX idx_branch_access_user_id ON branching.branch_access(user_id);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION branching.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER branches_updated_at
    BEFORE UPDATE ON branching.branches
    FOR EACH ROW EXECUTE FUNCTION branching.update_updated_at();

CREATE TRIGGER github_config_updated_at
    BEFORE UPDATE ON branching.github_config
    FOR EACH ROW EXECUTE FUNCTION branching.update_updated_at();

-- Insert the main branch record (represents the default database)
INSERT INTO branching.branches (name, slug, database_name, status, type)
VALUES ('Main', 'main', current_database(), 'ready', 'main');

-- Job functions table (job definitions/templates)
CREATE TABLE jobs.job_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    description TEXT,
    code TEXT,                        -- Bundled code for execution
    original_code TEXT,               -- Pre-bundle source code
    is_bundled BOOLEAN DEFAULT false,
    bundle_error TEXT,
    enabled BOOLEAN DEFAULT true,
    schedule TEXT,                    -- Cron schedule (optional, for scheduled jobs)
    timeout_seconds INTEGER DEFAULT 300,
    memory_limit_mb INTEGER DEFAULT 256,
    max_retries INTEGER DEFAULT 0,
    progress_timeout_seconds INTEGER DEFAULT 60,
    allow_net BOOLEAN DEFAULT true,
    allow_env BOOLEAN DEFAULT true,
    allow_read BOOLEAN DEFAULT false,
    allow_write BOOLEAN DEFAULT false,
    version INTEGER DEFAULT 1,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(name, namespace)
);

COMMENT ON TABLE jobs.job_functions IS 'Job function definitions (templates for jobs)';
COMMENT ON COLUMN jobs.job_functions.code IS 'Bundled JavaScript/TypeScript code';
COMMENT ON COLUMN jobs.job_functions.original_code IS 'Original source code before bundling';
COMMENT ON COLUMN jobs.job_functions.schedule IS 'Cron expression for scheduled execution';

CREATE INDEX idx_job_functions_namespace ON jobs.job_functions(namespace);
CREATE INDEX idx_job_functions_enabled ON jobs.job_functions(enabled) WHERE enabled = true;

-- Job execution queue (job instances/runs)
CREATE TABLE jobs.job_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace TEXT NOT NULL,
    job_function_id UUID REFERENCES jobs.job_functions(id) ON DELETE SET NULL,
    job_name TEXT NOT NULL,           -- Denormalized for performance
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    payload JSONB,                     -- Job input data
    result JSONB,                      -- Job output data
    progress JSONB,                    -- { percent: 0-100, message: "...", data: {...} }
    priority INTEGER DEFAULT 0,
    max_duration_seconds INTEGER,
    progress_timeout_seconds INTEGER,
    max_retries INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    logs TEXT,
    worker_id UUID,                    -- Will reference jobs.workers after it's created
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ,          -- For delayed jobs
    started_at TIMESTAMPTZ,
    last_progress_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

COMMENT ON TABLE jobs.job_queue IS 'Job execution queue and history';
COMMENT ON COLUMN jobs.job_queue.status IS 'Job execution status';
COMMENT ON COLUMN jobs.job_queue.priority IS 'Higher numbers = higher priority';
COMMENT ON COLUMN jobs.job_queue.progress IS 'Current progress state (for running jobs)';

CREATE INDEX idx_job_queue_status ON jobs.job_queue(status);
CREATE INDEX idx_job_queue_status_priority ON jobs.job_queue(status, priority DESC, created_at ASC);
CREATE INDEX idx_job_queue_namespace ON jobs.job_queue(namespace);
CREATE INDEX idx_job_queue_created_by ON jobs.job_queue(created_by);
CREATE INDEX idx_job_queue_created_at ON jobs.job_queue(created_at DESC);
CREATE INDEX idx_job_queue_scheduled_at ON jobs.job_queue(scheduled_at) WHERE scheduled_at IS NOT NULL AND status = 'pending';

-- Worker registry
CREATE TABLE jobs.workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    hostname TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'draining', 'stopped')),
    max_concurrent_jobs INTEGER DEFAULT 5,
    current_job_count INTEGER DEFAULT 0,
    last_heartbeat_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

COMMENT ON TABLE jobs.workers IS 'Active worker registry';
COMMENT ON COLUMN jobs.workers.status IS 'Worker status: active=accepting jobs, draining=finishing current jobs, stopped=shut down';
COMMENT ON COLUMN jobs.workers.last_heartbeat_at IS 'Last heartbeat timestamp for health monitoring';

CREATE INDEX idx_workers_status ON jobs.workers(status);
CREATE INDEX idx_workers_heartbeat ON jobs.workers(last_heartbeat_at);

-- Now add foreign key from job_queue to workers
ALTER TABLE jobs.job_queue ADD CONSTRAINT fk_job_queue_worker
    FOREIGN KEY (worker_id) REFERENCES jobs.workers(id) ON DELETE SET NULL;

-- Supporting files for multi-file jobs
CREATE TABLE jobs.job_function_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_function_id UUID NOT NULL REFERENCES jobs.job_functions(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(job_function_id, file_path)
);

COMMENT ON TABLE jobs.job_function_files IS 'Supporting files for multi-file job functions';

CREATE INDEX idx_job_function_files_function_id ON jobs.job_function_files(job_function_id);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION jobs.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_job_functions_updated_at
    BEFORE UPDATE ON jobs.job_functions
    FOR EACH ROW
    EXECUTE FUNCTION jobs.update_updated_at_column();

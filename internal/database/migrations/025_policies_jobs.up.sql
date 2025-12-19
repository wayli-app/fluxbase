-- Enable RLS on jobs tables
ALTER TABLE jobs.functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.workers ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.function_files ENABLE ROW LEVEL SECURITY;

-- Functions: Admin/Service role only
DROP POLICY IF EXISTS "Service role can manage functions" ON jobs.functions;
CREATE POLICY "Service role can manage functions"
    ON jobs.functions FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Queue: Users can only see/manage their own jobs
DROP POLICY IF EXISTS "Users can read their own jobs" ON jobs.queue;
CREATE POLICY "Users can read their own jobs"
    ON jobs.queue FOR SELECT
    TO authenticated
    USING (created_by = auth.uid());

DROP POLICY IF EXISTS "Users can submit jobs" ON jobs.queue;
CREATE POLICY "Users can submit jobs"
    ON jobs.queue FOR INSERT
    TO authenticated
    WITH CHECK (created_by = auth.uid());

DROP POLICY IF EXISTS "Users can cancel their own pending/running jobs" ON jobs.queue;
CREATE POLICY "Users can cancel their own pending/running jobs"
    ON jobs.queue FOR UPDATE
    TO authenticated
    USING (created_by = auth.uid() AND status IN ('pending', 'running'))
    WITH CHECK (status = 'cancelled');

DROP POLICY IF EXISTS "Service role can manage all jobs" ON jobs.queue;
CREATE POLICY "Service role can manage all jobs"
    ON jobs.queue FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Workers: Service role only
DROP POLICY IF EXISTS "Service role can manage workers" ON jobs.workers;
CREATE POLICY "Service role can manage workers"
    ON jobs.workers FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Function Files: Service role only (follows functions)
DROP POLICY IF EXISTS "Service role can manage function files" ON jobs.function_files;
CREATE POLICY "Service role can manage function files"
    ON jobs.function_files FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Note: Execution logs are now stored in the central logging schema (logging.entries)

-- ============================================
-- Dashboard Admin Policies
-- Dashboard admins can read all jobs data
-- ============================================
DROP POLICY IF EXISTS "Dashboard admins can read all jobs" ON jobs.queue;
CREATE POLICY "Dashboard admins can read all jobs"
    ON jobs.queue FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

DROP POLICY IF EXISTS "Dashboard admins can read all functions" ON jobs.functions;
CREATE POLICY "Dashboard admins can read all functions"
    ON jobs.functions FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

DROP POLICY IF EXISTS "Dashboard admins can read all workers" ON jobs.workers;
CREATE POLICY "Dashboard admins can read all workers"
    ON jobs.workers FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

DROP POLICY IF EXISTS "Dashboard admins can read all function files" ON jobs.function_files;
CREATE POLICY "Dashboard admins can read all function files"
    ON jobs.function_files FOR SELECT
    TO authenticated
    USING (auth.role() = 'dashboard_admin');

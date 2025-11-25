-- Enable RLS on jobs tables
ALTER TABLE jobs.job_functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.job_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.workers ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs.job_function_files ENABLE ROW LEVEL SECURITY;

-- Job Functions: Admin/Service role only
CREATE POLICY "Service role can manage job functions"
    ON jobs.job_functions FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Job Queue: Users can only see/manage their own jobs
CREATE POLICY "Users can read their own jobs"
    ON jobs.job_queue FOR SELECT
    TO authenticated
    USING (created_by = auth.uid());

CREATE POLICY "Users can submit jobs"
    ON jobs.job_queue FOR INSERT
    TO authenticated
    WITH CHECK (created_by = auth.uid());

CREATE POLICY "Users can cancel their own pending/running jobs"
    ON jobs.job_queue FOR UPDATE
    TO authenticated
    USING (created_by = auth.uid() AND status IN ('pending', 'running'))
    WITH CHECK (status = 'cancelled');

CREATE POLICY "Service role can manage all jobs"
    ON jobs.job_queue FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Workers: Service role only
CREATE POLICY "Service role can manage workers"
    ON jobs.workers FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

-- Job Function Files: Service role only (follows job functions)
CREATE POLICY "Service role can manage job function files"
    ON jobs.job_function_files FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

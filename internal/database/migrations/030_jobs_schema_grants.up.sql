-- Grant schema usage to authenticated role (needed for dashboard_admin access)
GRANT USAGE ON SCHEMA jobs TO authenticated;

-- Grant SELECT on all jobs tables to authenticated role
-- This allows dashboard_admin to view jobs data in the admin UI
GRANT SELECT ON ALL TABLES IN SCHEMA jobs TO authenticated;

-- Grant default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    GRANT SELECT ON TABLES TO authenticated;

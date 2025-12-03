-- Revoke permissions for jobs.execution_logs table
REVOKE ALL ON jobs.execution_logs FROM authenticated;
REVOKE ALL ON jobs.execution_logs FROM service_role;
REVOKE ALL ON SEQUENCE jobs.execution_logs_id_seq FROM service_role;

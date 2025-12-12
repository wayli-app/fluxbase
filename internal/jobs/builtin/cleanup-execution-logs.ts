/**
 * Built-in: Cleanup Execution Logs
 *
 * Cleans up old execution logs from functions, RPC, and jobs tables.
 * Each log type has its own configurable retention period via environment variables:
 *
 * - FLUXBASE_JOBS_FUNCTIONS_LOGS_RETENTION_DAYS (default: 30)
 * - FLUXBASE_JOBS_RPC_LOGS_RETENTION_DAYS (default: 30)
 * - FLUXBASE_JOBS_JOBS_LOGS_RETENTION_DAYS (default: 30)
 *
 * Set any retention value to 0 to skip cleanup for that log type.
 * Parent execution records are preserved - only log entries are deleted.
 *
 * @fluxbase:enabled false
 * @fluxbase:schedule 0 2 * * *
 * @fluxbase:timeout 3600
 * @fluxbase:require-role admin
 * @fluxbase:description Cleans up old execution logs (disabled by default)
 */

export async function handler(_req: unknown) {
  console.log("Starting execution logs cleanup job...");

  await Fluxbase.reportProgress(0, "Reading configuration...");

  // Read retention periods from environment (defaults handled by Go config)
  const functionsRetention = parseInt(
    Deno.env.get("FLUXBASE_JOBS_FUNCTIONS_LOGS_RETENTION_DAYS") || "30",
    10
  );
  const rpcRetention = parseInt(
    Deno.env.get("FLUXBASE_JOBS_RPC_LOGS_RETENTION_DAYS") || "30",
    10
  );
  const jobsRetention = parseInt(
    Deno.env.get("FLUXBASE_JOBS_JOBS_LOGS_RETENTION_DAYS") || "30",
    10
  );

  console.log(`Retention periods: functions=${functionsRetention}d, rpc=${rpcRetention}d, jobs=${jobsRetention}d`);

  const results = {
    functions_logs: { deleted: 0, skipped: false },
    rpc_logs: { deleted: 0, skipped: false },
    jobs_logs: { deleted: 0, skipped: false },
  };

  // Clean up functions execution logs
  if (functionsRetention > 0) {
    await Fluxbase.reportProgress(10, "Cleaning up functions execution logs...");
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - functionsRetention);

    console.log(`Deleting functions logs older than ${cutoffDate.toISOString()}`);

    const { data, error } = await Fluxbase.database()
      .from("functions.execution_logs")
      .delete()
      .lt("created_at", cutoffDate.toISOString())
      .select("id");

    if (error) {
      console.error("Failed to delete functions execution logs:", error);
    } else {
      results.functions_logs.deleted = data?.length || 0;
      console.log(`Deleted ${results.functions_logs.deleted} functions execution logs`);
    }
  } else {
    results.functions_logs.skipped = true;
    console.log("Skipping functions logs cleanup (retention set to 0)");
  }

  // Clean up RPC execution logs
  if (rpcRetention > 0) {
    await Fluxbase.reportProgress(40, "Cleaning up RPC execution logs...");
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - rpcRetention);

    console.log(`Deleting RPC logs older than ${cutoffDate.toISOString()}`);

    const { data, error } = await Fluxbase.database()
      .from("rpc.execution_logs")
      .delete()
      .lt("created_at", cutoffDate.toISOString())
      .select("id");

    if (error) {
      console.error("Failed to delete RPC execution logs:", error);
    } else {
      results.rpc_logs.deleted = data?.length || 0;
      console.log(`Deleted ${results.rpc_logs.deleted} RPC execution logs`);
    }
  } else {
    results.rpc_logs.skipped = true;
    console.log("Skipping RPC logs cleanup (retention set to 0)");
  }

  // Clean up jobs execution logs
  if (jobsRetention > 0) {
    await Fluxbase.reportProgress(70, "Cleaning up jobs execution logs...");
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - jobsRetention);

    console.log(`Deleting jobs logs older than ${cutoffDate.toISOString()}`);

    const { data, error } = await Fluxbase.database()
      .from("jobs.execution_logs")
      .delete()
      .lt("created_at", cutoffDate.toISOString())
      .select("id");

    if (error) {
      console.error("Failed to delete jobs execution logs:", error);
    } else {
      results.jobs_logs.deleted = data?.length || 0;
      console.log(`Deleted ${results.jobs_logs.deleted} jobs execution logs`);
    }
  } else {
    results.jobs_logs.skipped = true;
    console.log("Skipping jobs logs cleanup (retention set to 0)");
  }

  await Fluxbase.reportProgress(100, "Cleanup complete");

  const totalDeleted =
    results.functions_logs.deleted +
    results.rpc_logs.deleted +
    results.jobs_logs.deleted;

  console.log(`Cleanup complete. Total logs deleted: ${totalDeleted}`);

  return {
    success: true,
    retention_days: {
      functions: functionsRetention,
      rpc: rpcRetention,
      jobs: jobsRetention,
    },
    deleted: results,
    total_deleted: totalDeleted,
    completed_at: new Date().toISOString(),
  };
}

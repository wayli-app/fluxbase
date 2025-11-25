/**
 * Example: Cleanup Old Data Job
 *
 * This job demonstrates:
 * - Scheduled job (cron expression)
 * - Admin-only access
 * - Batch operations
 * - Simple progress reporting
 *
 * @fluxbase:require-role admin
 * @fluxbase:schedule 0 2 * * *
 * @fluxbase:timeout 1800
 * @fluxbase:description Cleans up old data (runs daily at 2 AM)
 */

export async function handler(req: any) {
  const context = Fluxbase.getJobContext();

  console.log("Starting cleanup job...");
  console.log(`Triggered by: ${context.user?.email || "scheduler"}`);

  await Fluxbase.reportProgress(0, "Starting cleanup...");

  // Get retention period from payload or use default (30 days)
  const retentionDays = context.payload?.retention_days || 30;
  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - retentionDays);

  console.log(
    `Deleting data older than ${retentionDays} days (before ${cutoffDate.toISOString()})`
  );

  await Fluxbase.reportProgress(10, "Cleaning up old sessions...");

  // Delete old sessions
  const { data: deletedSessions, error: sessionsError } = await Fluxbase.database()
    .from("auth.sessions")
    .delete()
    .lt("created_at", cutoffDate.toISOString())
    .select("id");

  if (sessionsError) {
    console.error("Failed to delete sessions:", sessionsError);
  } else {
    console.log(`Deleted ${deletedSessions?.length || 0} old sessions`);
  }

  await Fluxbase.reportProgress(40, "Cleaning up old logs...");

  // Delete old activity logs
  const { data: deletedLogs, error: logsError } = await Fluxbase.database()
    .from("app.activity_logs")
    .delete()
    .lt("created_at", cutoffDate.toISOString())
    .select("id");

  if (logsError) {
    console.error("Failed to delete logs:", logsError);
  } else {
    console.log(`Deleted ${deletedLogs?.length || 0} old logs`);
  }

  await Fluxbase.reportProgress(70, "Cleaning up expired invitations...");

  // Delete expired invitations
  const { data: deletedInvitations, error: invitationsError } =
    await Fluxbase.database()
      .from("auth.user_invitations")
      .delete()
      .lt("expires_at", new Date().toISOString())
      .select("id");

  if (invitationsError) {
    console.error("Failed to delete invitations:", invitationsError);
  } else {
    console.log(`Deleted ${deletedInvitations?.length || 0} expired invitations`);
  }

  await Fluxbase.reportProgress(100, "Cleanup complete");

  return {
    success: true,
    retention_days: retentionDays,
    deleted: {
      sessions: deletedSessions?.length || 0,
      logs: deletedLogs?.length || 0,
      invitations: deletedInvitations?.length || 0,
    },
    completed_at: new Date().toISOString(),
  };
}

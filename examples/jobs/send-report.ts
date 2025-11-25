/**
 * Example: Send Report Job
 *
 * This job demonstrates:
 * - Admin-only permissions using @fluxbase:require-role
 * - Accessing user context
 * - Using environment variables
 * - Progress reporting
 * - Database access with user context
 *
 * @fluxbase:require-role admin
 * @fluxbase:timeout 600
 * @fluxbase:description Generates and sends a report to all users (admin only)
 */

export async function handler(req: any) {
  // Get job context including user info
  const context = Fluxbase.getJobContext();

  console.log(`Job started by: ${context.user?.email} (${context.user?.role})`);
  console.log(`Job ID: ${context.job_id}`);
  console.log(`Payload:`, context.payload);

  // Access environment variables (FLUXBASE_* only)
  const apiUrl = Deno.env.get("FLUXBASE_API_URL") || "http://localhost:8000";
  const appName = Deno.env.get("FLUXBASE_APP_NAME") || "Fluxbase";

  console.log(`App Name: ${appName}`);
  console.log(`API URL: ${apiUrl}`);

  // Report initial progress
  await Fluxbase.reportProgress(10, "Fetching users from database...");

  // Query database with user context (uses RLS automatically)
  const { data: users, error } = await Fluxbase.database()
    .from("auth.users")
    .select("id, email, name")
    .eq("role", "user");

  if (error) {
    throw new Error(`Database query failed: ${error.message}`);
  }

  await Fluxbase.reportProgress(30, `Found ${users?.length || 0} users`);

  // Simulate report generation
  const reportType = context.payload?.report_type || "monthly";
  const report = {
    type: reportType,
    generated_at: new Date().toISOString(),
    generated_by: context.user?.email,
    total_users: users?.length || 0,
    summary: "This is a sample report",
  };

  await Fluxbase.reportProgress(60, "Report generated, sending emails...");

  // Simulate sending emails to users
  let sent = 0;
  for (const user of users || []) {
    // Simulate email sending delay
    await new Promise((resolve) => setTimeout(resolve, 100));

    console.log(`Sending report to ${user.email}`);
    sent++;

    // Update progress as we send emails
    const progress = 60 + Math.floor((sent / (users?.length || 1)) * 40);
    await Fluxbase.reportProgress(
      progress,
      `Sent ${sent}/${users?.length} emails`
    );
  }

  await Fluxbase.reportProgress(100, "All emails sent successfully");

  // Return the result
  return {
    success: true,
    report_type: reportType,
    users_notified: sent,
    generated_at: report.generated_at,
    generated_by: report.generated_by,
  };
}

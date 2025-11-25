/**
 * Client-side example: Using Fluxbase Jobs
 *
 * This example shows how to submit, monitor, and manage jobs from your application.
 */

import { createClient } from "@fluxbase/sdk";

// Initialize client
const client = createClient(process.env.FLUXBASE_URL!, {
  apiKey: process.env.FLUXBASE_ANON_KEY!,
});

// Authenticate user
await client.auth.login({
  email: "user@example.com",
  password: "password123",
});

/**
 * Example 1: Submit a simple job and wait for completion
 */
async function submitAndWaitForJob() {
  console.log("Submitting job...");

  // Submit the job
  const { data: job, error } = await client.jobs.submit("process-user-data", {
    items: [
      { id: 1, data: "item1" },
      { id: 2, data: "item2" },
      { id: 3, data: "item3" },
    ],
  });

  if (error) {
    console.error("Failed to submit job:", error);
    return;
  }

  console.log("Job submitted:", job.id);
  console.log("Initial status:", job.status);

  // Poll for completion
  let completed = false;
  while (!completed) {
    await new Promise((resolve) => setTimeout(resolve, 2000)); // Wait 2 seconds

    const { data: updatedJob, error: statusError } = await client.jobs.get(
      job.id
    );

    if (statusError) {
      console.error("Failed to get job status:", statusError);
      break;
    }

    console.log(
      `Progress: ${updatedJob.progress_percent}% - ${updatedJob.progress_message}`
    );

    if (
      updatedJob.status === "completed" ||
      updatedJob.status === "failed" ||
      updatedJob.status === "cancelled"
    ) {
      completed = true;

      if (updatedJob.status === "completed") {
        console.log("Job completed successfully!");
        console.log("Result:", updatedJob.result);
      } else if (updatedJob.status === "failed") {
        console.error("Job failed:", updatedJob.error);
      } else {
        console.log("Job was cancelled");
      }

      if (updatedJob.logs) {
        console.log("\nJob logs:");
        console.log(updatedJob.logs);
      }
    }
  }
}

/**
 * Example 2: Submit with options (priority, scheduling)
 */
async function submitWithOptions() {
  // Submit with high priority
  const { data: highPriorityJob } = await client.jobs.submit(
    "process-user-data",
    { items: [{ id: 1 }] },
    { priority: 10 }
  );

  console.log("High priority job:", highPriorityJob?.id);

  // Schedule for later
  const scheduledTime = new Date();
  scheduledTime.setHours(scheduledTime.getHours() + 1); // Run in 1 hour

  const { data: scheduledJob } = await client.jobs.submit(
    "process-user-data",
    { items: [{ id: 2 }] },
    { scheduled: scheduledTime.toISOString() }
  );

  console.log("Scheduled job:", scheduledJob?.id);
  console.log("Will run at:", scheduledJob?.scheduled_at);
}

/**
 * Example 3: List user's jobs
 */
async function listMyJobs() {
  // List all jobs
  const { data: allJobs } = await client.jobs.list();
  console.log("Total jobs:", allJobs?.length);

  // List only running jobs
  const { data: runningJobs } = await client.jobs.list({ status: "running" });
  console.log("Running jobs:", runningJobs?.length);

  runningJobs?.forEach((job) => {
    console.log(
      `- ${job.job_name} (${job.id}): ${job.progress_percent}% - ${job.progress_message}`
    );
  });

  // List with pagination
  const { data: recentJobs } = await client.jobs.list({ limit: 10, offset: 0 });
  console.log("Recent 10 jobs:", recentJobs?.length);
}

/**
 * Example 4: Cancel a running job
 */
async function cancelJob(jobId: string) {
  const { error } = await client.jobs.cancel(jobId);

  if (error) {
    console.error("Failed to cancel job:", error);
  } else {
    console.log("Job cancelled successfully");
  }
}

/**
 * Example 5: Retry a failed job
 */
async function retryFailedJob(jobId: string) {
  const { data: newJob, error } = await client.jobs.retry(jobId);

  if (error) {
    console.error("Failed to retry job:", error);
  } else {
    console.log("Job retried, new job ID:", newJob?.id);
  }
}

/**
 * Example 6: Admin operations (requires admin authentication)
 */
async function adminOperations() {
  // Authenticate as admin
  const adminClient = createClient(process.env.FLUXBASE_URL!, {
    apiKey: process.env.FLUXBASE_SERVICE_KEY!, // Use service key
  });

  // Create a new job function
  const jobCode = `
/**
 * @fluxbase:require-role admin
 * @fluxbase:timeout 600
 */
export async function handler(req) {
  const context = Fluxbase.getJobContext();
  console.log('Running as:', context.user?.email);

  await Fluxbase.reportProgress(50, 'Processing...');

  return { success: true };
}
  `.trim();

  const { data: jobFunction, error: createError } =
    await adminClient.admin.jobs.create({
      name: "my-custom-job",
      namespace: "default",
      code: jobCode,
      enabled: true,
      timeout_seconds: 600,
    });

  if (createError) {
    console.error("Failed to create job function:", createError);
  } else {
    console.log("Job function created:", jobFunction?.name);
  }

  // List all job functions
  const { data: jobFunctions } = await adminClient.admin.jobs.list("default");
  console.log("Job functions:", jobFunctions?.map((f) => f.name));

  // Get statistics
  const { data: stats } = await adminClient.admin.jobs.getStats("default");
  console.log("Job statistics:");
  console.log(`- Pending: ${stats?.pending}`);
  console.log(`- Running: ${stats?.running}`);
  console.log(`- Completed: ${stats?.completed}`);
  console.log(`- Failed: ${stats?.failed}`);

  // List all jobs (admin view - sees all users' jobs)
  const { data: allJobs } = await adminClient.admin.jobs.listJobs({
    namespace: "default",
    limit: 20,
  });
  console.log("All jobs (admin view):", allJobs?.length);

  // List workers
  const { data: workers } = await adminClient.admin.jobs.listWorkers();
  console.log("Active workers:", workers?.length);
  workers?.forEach((worker) => {
    console.log(
      `- Worker ${worker.id}: ${worker.current_jobs} current jobs, ${worker.total_completed} completed`
    );
  });

  // Update job function
  const { data: updatedFunction } = await adminClient.admin.jobs.update(
    "default",
    "my-custom-job",
    {
      enabled: false,
      timeout_seconds: 900,
    }
  );
  console.log("Job function updated:", updatedFunction?.enabled);

  // Sync from filesystem
  const { data: syncResult } = await adminClient.admin.jobs.sync("default");
  if (syncResult) {
    console.log("Sync results:");
    console.log(`- Created: ${syncResult.summary.created}`);
    console.log(`- Updated: ${syncResult.summary.updated}`);
    console.log(`- Deleted: ${syncResult.summary.deleted}`);
  }

  // Delete job function
  const { error: deleteError } = await adminClient.admin.jobs.delete(
    "default",
    "my-custom-job"
  );
  if (!deleteError) {
    console.log("Job function deleted");
  }
}

/**
 * Example 7: Real-time job monitoring with progress updates
 */
async function monitorJobWithRealtime(jobId: string) {
  console.log("Monitoring job:", jobId);

  // Subscribe to realtime updates (if realtime is enabled)
  const channel = client.realtime.channel(`job:${jobId}`);

  channel
    .on("job_progress", (payload: any) => {
      console.log(
        `Progress update: ${payload.progress_percent}% - ${payload.progress_message}`
      );
    })
    .on("job_completed", (payload: any) => {
      console.log("Job completed:", payload.result);
      channel.unsubscribe();
    })
    .on("job_failed", (payload: any) => {
      console.error("Job failed:", payload.error);
      channel.unsubscribe();
    })
    .subscribe();
}

/**
 * Example 8: Batch job submission
 */
async function submitBatchJobs() {
  const dataFiles = ["file1.csv", "file2.csv", "file3.csv"];

  const jobs = await Promise.all(
    dataFiles.map(async (file) => {
      const { data, error } = await client.jobs.submit("process-user-data", {
        file,
      });

      if (error) {
        console.error(`Failed to submit job for ${file}:`, error);
        return null;
      }

      return data;
    })
  );

  console.log(`Submitted ${jobs.filter((j) => j !== null).length} jobs`);

  // Monitor all jobs
  const results = await Promise.all(
    jobs
      .filter((j) => j !== null)
      .map(async (job) => {
        let completed = false;
        let result = null;

        while (!completed) {
          await new Promise((resolve) => setTimeout(resolve, 2000));

          const { data } = await client.jobs.get(job!.id);

          if (
            data?.status === "completed" ||
            data?.status === "failed" ||
            data?.status === "cancelled"
          ) {
            completed = true;
            result = data;
          }
        }

        return result;
      })
  );

  console.log("All jobs completed");
  results.forEach((result, index) => {
    console.log(`Job ${index + 1}:`, result?.status, result?.result);
  });
}

// Run examples
(async () => {
  try {
    console.log("=== Example 1: Submit and wait ===");
    await submitAndWaitForJob();

    console.log("\n=== Example 2: Submit with options ===");
    await submitWithOptions();

    console.log("\n=== Example 3: List jobs ===");
    await listMyJobs();

    console.log("\n=== Example 6: Admin operations ===");
    await adminOperations();

    console.log("\n=== Example 8: Batch submission ===");
    await submitBatchJobs();
  } catch (error) {
    console.error("Error:", error);
  }
})();

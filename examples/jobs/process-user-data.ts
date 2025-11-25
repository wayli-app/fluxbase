/**
 * Example: Process User Data Job
 *
 * This job demonstrates:
 * - No special permissions required (any authenticated user can trigger)
 * - User can only access their own data via user context
 * - Long-running data processing with progress tracking
 * - Error handling and retry logic
 *
 * @fluxbase:timeout 300
 * @fluxbase:max-retries 3
 * @fluxbase:description Processes user's uploaded data
 */

export async function handler(req: any) {
  const context = Fluxbase.getJobContext();

  // User context is automatically available
  if (!context.user) {
    throw new Error("This job requires authentication");
  }

  console.log(`Processing data for user: ${context.user.email}`);

  const { items } = context.payload || {};

  if (!items || !Array.isArray(items)) {
    throw new Error("Invalid payload: items array is required");
  }

  await Fluxbase.reportProgress(0, "Starting data processing...");

  // User can only query their own data due to RLS
  const { data: userProfile, error: profileError } = await Fluxbase.database()
    .from("app.profiles")
    .select("*")
    .eq("user_id", context.user.id)
    .single();

  if (profileError) {
    throw new Error(`Failed to fetch user profile: ${profileError.message}`);
  }

  await Fluxbase.reportProgress(
    10,
    `Processing ${items.length} items for ${userProfile.name || context.user.email}`
  );

  // Process items one by one
  const results = [];
  let processed = 0;

  for (const item of items) {
    try {
      // Simulate processing delay
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Process the item
      const result = {
        id: item.id || `item-${processed}`,
        status: "processed",
        processed_at: new Date().toISOString(),
        user_id: context.user.id,
      };

      // Store result in database (with user context, so RLS applies)
      const { error: insertError } = await Fluxbase.database()
        .from("app.processed_items")
        .insert(result);

      if (insertError) {
        console.error(`Failed to store result for item ${item.id}:`, insertError);
        result.status = "failed";
      }

      results.push(result);
      processed++;

      // Update progress
      const progress = 10 + Math.floor((processed / items.length) * 90);
      await Fluxbase.reportProgress(
        progress,
        `Processed ${processed}/${items.length} items`
      );
    } catch (error) {
      console.error(`Error processing item:`, error);
      results.push({
        id: item.id,
        status: "error",
        error: error.message,
      });
      processed++;
    }
  }

  await Fluxbase.reportProgress(100, "Processing complete");

  const successCount = results.filter((r) => r.status === "processed").length;
  const failCount = results.filter((r) => r.status !== "processed").length;

  return {
    success: true,
    total_items: items.length,
    processed: successCount,
    failed: failCount,
    user_email: context.user.email,
    results,
  };
}

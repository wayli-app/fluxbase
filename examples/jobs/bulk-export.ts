/**
 * Example: Bulk Export Job
 *
 * Advanced example demonstrating:
 * - Complex multi-step workflow
 * - Error handling with partial success
 * - File generation and storage
 * - Email notification on completion
 * - Comprehensive progress tracking
 *
 * @fluxbase:timeout 1800
 * @fluxbase:max-retries 2
 * @fluxbase:description Exports user data to CSV and sends download link
 */

export async function handler(req: any) {
  const context = Fluxbase.getJobContext();

  if (!context.user) {
    throw new Error("Authentication required");
  }

  console.log(`Starting bulk export for user: ${context.user.email}`);

  const { table_name, filters, format = "csv" } = context.payload || {};

  if (!table_name) {
    throw new Error("table_name is required in payload");
  }

  // Step 1: Validate table access
  await Fluxbase.reportProgress(5, "Validating table access...");

  const allowedTables = ["profiles", "orders", "activity_logs"];
  if (!allowedTables.includes(table_name)) {
    throw new Error(`Table ${table_name} is not exportable`);
  }

  // Step 2: Count total records
  await Fluxbase.reportProgress(10, "Counting records...");

  let query = Fluxbase.database()
    .from(`app.${table_name}`)
    .select("*", { count: "exact", head: true });

  // Apply filters if provided
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      query = query.eq(key, value);
    });
  }

  // RLS automatically ensures user can only access their own data
  const { count, error: countError } = await query;

  if (countError) {
    throw new Error(`Failed to count records: ${countError.message}`);
  }

  console.log(`Found ${count} records to export`);

  if (count === 0) {
    return {
      success: true,
      message: "No records to export",
      count: 0,
    };
  }

  // Step 3: Fetch data in batches
  await Fluxbase.reportProgress(20, "Fetching data...");

  const batchSize = 1000;
  const batches = Math.ceil(count / batchSize);
  const allData: any[] = [];

  for (let i = 0; i < batches; i++) {
    const from = i * batchSize;
    const to = from + batchSize - 1;

    let batchQuery = Fluxbase.database()
      .from(`app.${table_name}`)
      .select("*")
      .range(from, to);

    // Apply same filters
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        batchQuery = batchQuery.eq(key, value);
      });
    }

    const { data, error } = await batchQuery;

    if (error) {
      console.error(`Failed to fetch batch ${i + 1}:`, error);
      throw new Error(`Failed to fetch data: ${error.message}`);
    }

    allData.push(...(data || []));

    const progress = 20 + Math.floor((i / batches) * 30);
    await Fluxbase.reportProgress(
      progress,
      `Fetched ${allData.length}/${count} records`
    );
  }

  // Step 4: Convert to requested format
  await Fluxbase.reportProgress(50, `Converting to ${format.toUpperCase()}...`);

  let exportData: string;
  let contentType: string;
  let fileExtension: string;

  if (format === "csv") {
    // Convert to CSV
    const headers = Object.keys(allData[0] || {});
    const csvRows = [
      headers.join(","), // Header row
      ...allData.map((row) =>
        headers.map((header) => {
          const value = row[header];
          // Escape quotes and wrap in quotes if contains comma
          if (value === null || value === undefined) return "";
          const stringValue = String(value);
          if (stringValue.includes(",") || stringValue.includes('"')) {
            return `"${stringValue.replace(/"/g, '""')}"`;
          }
          return stringValue;
        }).join(",")
      ),
    ];

    exportData = csvRows.join("\n");
    contentType = "text/csv";
    fileExtension = "csv";
  } else if (format === "json") {
    // Convert to JSON
    exportData = JSON.stringify(allData, null, 2);
    contentType = "application/json";
    fileExtension = "json";
  } else {
    throw new Error(`Unsupported format: ${format}`);
  }

  await Fluxbase.reportProgress(60, "Upload file to storage...");

  // Step 5: Upload to storage
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
  const fileName = `exports/${context.user.id}/${table_name}-${timestamp}.${fileExtension}`;

  // Convert string to Uint8Array for storage
  const encoder = new TextEncoder();
  const fileData = encoder.encode(exportData);

  const { data: uploadData, error: uploadError } = await Fluxbase.storage()
    .from("exports")
    .upload(fileName, fileData, {
      contentType,
      upsert: false,
    });

  if (uploadError) {
    throw new Error(`Failed to upload file: ${uploadError.message}`);
  }

  console.log(`File uploaded: ${fileName}`);

  await Fluxbase.reportProgress(75, "Generating download link...");

  // Step 6: Generate signed download URL (valid for 7 days)
  const { data: urlData, error: urlError } = await Fluxbase.storage()
    .from("exports")
    .createSignedUrl(fileName, 7 * 24 * 60 * 60); // 7 days in seconds

  if (urlError) {
    console.error("Failed to generate download URL:", urlError);
  }

  const downloadUrl = urlData?.signedUrl;

  await Fluxbase.reportProgress(85, "Saving export record...");

  // Step 7: Save export record to database
  const { error: recordError } = await Fluxbase.database()
    .from("app.exports")
    .insert({
      user_id: context.user.id,
      table_name,
      format,
      record_count: count,
      file_path: fileName,
      file_size_bytes: fileData.length,
      download_url: downloadUrl,
      filters: filters || {},
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      created_at: new Date().toISOString(),
    });

  if (recordError) {
    console.error("Failed to save export record:", recordError);
    // Continue anyway - file is uploaded
  }

  await Fluxbase.reportProgress(95, "Sending notification email...");

  // Step 8: Send email notification (optional)
  try {
    const emailHtml = `
      <h2>Your export is ready!</h2>
      <p>Hi ${context.user.email},</p>
      <p>Your export of <strong>${count} records</strong> from <strong>${table_name}</strong> is ready for download.</p>
      <p>
        <a href="${downloadUrl}" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px;">
          Download Export
        </a>
      </p>
      <p><small>This link will expire in 7 days.</small></p>
      <p>File format: ${format.toUpperCase()}<br>
      File size: ${(fileData.length / 1024).toFixed(2)} KB<br>
      Records: ${count}</p>
    `;

    // Note: This requires the email service to be configured
    // and the user to have proper email sending permissions
    await fetch(
      `${Deno.env.get("FLUXBASE_API_URL")}/api/v1/admin/email/send`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${Deno.env.get("FLUXBASE_SERVICE_KEY")}`,
        },
        body: JSON.stringify({
          to: context.user.email,
          subject: `Your ${table_name} export is ready`,
          html: emailHtml,
        }),
      }
    );

    console.log("Notification email sent");
  } catch (emailError) {
    console.error("Failed to send email:", emailError);
    // Don't fail the job if email fails
  }

  await Fluxbase.reportProgress(100, "Export complete");

  // Step 9: Return success result
  return {
    success: true,
    table_name,
    format,
    record_count: count,
    file_size_bytes: fileData.length,
    file_size_kb: (fileData.length / 1024).toFixed(2),
    download_url: downloadUrl,
    expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
    completed_at: new Date().toISOString(),
  };
}

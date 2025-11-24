/**
 * Edge Function Management Example
 *
 * This example demonstrates how to:
 * 1. Create edge functions programmatically
 * 2. Update function code and settings
 * 3. List and retrieve functions
 * 4. Delete functions
 * 5. View execution history
 */

import { createClient } from '@fluxbase/sdk';
import dotenv from 'dotenv';
import { readFileSync } from 'fs';
import { join } from 'path';

// Load environment variables
dotenv.config();

// Validate environment variables
const FLUXBASE_URL = process.env.FLUXBASE_URL;
const FLUXBASE_SERVICE_KEY = process.env.FLUXBASE_SERVICE_KEY; // Service key required for management operations

if (!FLUXBASE_URL || !FLUXBASE_SERVICE_KEY) {
  console.error('❌ Missing required environment variables');
  console.error('Please set FLUXBASE_URL and FLUXBASE_SERVICE_KEY in your .env file');
  process.exit(1);
}

// Initialize Fluxbase client with service key
const client = createClient(FLUXBASE_URL, FLUXBASE_SERVICE_KEY, {
  debug: process.env.DEBUG === 'true',
});

/**
 * Example 1: Create a new edge function
 */
async function createFunction() {
  console.log('\n📦 Example 1: Creating a new edge function\n');

  // Read function code from file
  const functionCode = readFileSync(
    join(__dirname, '../functions/hello-world.ts'),
    'utf-8'
  );

  try {
    const { data, error } = await client.functions.create({
      name: 'hello-world',
      description: 'A simple hello world function',
      code: functionCode,
      enabled: true,
      timeout_seconds: 30,
      memory_limit_mb: 128,
      allow_net: true,
      allow_env: true,
      allow_read: false,
      allow_write: false,
      allow_unauthenticated: true,
    });

    if (error) {
      console.error('❌ Error:', error);
      if (error.details) {
        console.error('Details:', error.details);
      }
      return;
    }

    console.log('✅ Function created successfully!');
    console.log('Function ID:', data.id);
    console.log('Function Name:', data.name);
    console.log('Enabled:', data.enabled);
    console.log('Is Bundled:', data.is_bundled);
    if (data.bundle_error) {
      console.warn('⚠️ Bundle warning:', data.bundle_error);
    }
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 2: Create error handling function
 */
async function createErrorHandlingFunction() {
  console.log('\n📦 Example 2: Creating error handling function\n');

  const functionCode = readFileSync(
    join(__dirname, '../functions/error-handling.ts'),
    'utf-8'
  );

  try {
    const { data, error } = await client.functions.create({
      name: 'error-handling',
      description: 'Demonstrates error handling patterns',
      code: functionCode,
      enabled: true,
      allow_unauthenticated: true,
    });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Error handling function created!');
    console.log('Function Name:', data.name);
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 3: List all functions
 */
async function listFunctions() {
  console.log('\n📦 Example 3: Listing all edge functions\n');

  try {
    const { data, error } = await client.functions.list();

    if (error) {
      console.error('❌ Error:', error);
      if (error.request_id) {
        console.error('🔍 Request ID:', error.request_id);
      }
      return;
    }

    console.log(`✅ Found ${data.length} function(s):\n`);

    data.forEach((fn, index) => {
      console.log(`${index + 1}. ${fn.name}`);
      console.log(`   Description: ${fn.description || 'N/A'}`);
      console.log(`   Enabled: ${fn.enabled}`);
      console.log(`   Timeout: ${fn.timeout_seconds}s`);
      console.log(`   Memory: ${fn.memory_limit_mb}MB`);
      console.log(`   Allow Unauthenticated: ${fn.allow_unauthenticated}`);
      console.log(`   Created: ${new Date(fn.created_at).toLocaleString()}`);
      console.log('');
    });
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 4: Get a specific function
 */
async function getFunction(name: string) {
  console.log(`\n📦 Example 4: Getting function '${name}'\n`);

  try {
    const { data, error } = await client.functions.get(name);

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Function details:');
    console.log('Name:', data.name);
    console.log('Description:', data.description);
    console.log('Enabled:', data.enabled);
    console.log('Version:', data.version);
    console.log('Is Bundled:', data.is_bundled);
    console.log('Permissions:');
    console.log('  - Network:', data.allow_net);
    console.log('  - Environment:', data.allow_env);
    console.log('  - Read:', data.allow_read);
    console.log('  - Write:', data.allow_write);
    console.log('Code length:', data.code?.length || 0, 'characters');
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 5: Update a function
 */
async function updateFunction(name: string) {
  console.log(`\n📦 Example 5: Updating function '${name}'\n`);

  try {
    const { data, error } = await client.functions.update(name, {
      description: 'Updated description',
      timeout_seconds: 60,
    });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Function updated successfully!');
    console.log('New description:', data.description);
    console.log('New timeout:', data.timeout_seconds, 'seconds');
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 6: Get execution history
 */
async function getExecutions(name: string) {
  console.log(`\n📦 Example 6: Getting execution history for '${name}'\n`);

  try {
    const { data, error } = await client.functions.getExecutions(name, { limit: 10 });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log(`✅ Found ${data.length} execution(s):\n`);

    data.forEach((exec, index) => {
      console.log(`${index + 1}. Execution at ${new Date(exec.created_at).toLocaleString()}`);
      console.log(`   Status: ${exec.status}`);
      console.log(`   Status Code: ${exec.status_code || 'N/A'}`);
      console.log(`   Duration: ${exec.duration_ms || 'N/A'}ms`);
      console.log(`   Trigger: ${exec.trigger_type}`);
      if (exec.error_message) {
        console.log(`   Error: ${exec.error_message}`);
      }
      console.log('');
    });
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 7: Delete a function
 */
async function deleteFunction(name: string) {
  console.log(`\n📦 Example 7: Deleting function '${name}'\n`);

  try {
    const { error } = await client.functions.delete(name);

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Function deleted successfully!');
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Run all examples
 */
async function main() {
  console.log('='.repeat(60));
  console.log('🚀 Fluxbase Edge Functions - Management Examples');
  console.log('='.repeat(60));

  // Create functions
  await createFunction();
  await createErrorHandlingFunction();

  // List and retrieve
  await listFunctions();
  await getFunction('hello-world');

  // Update
  await updateFunction('hello-world');

  // Get executions (after running invoke-function.ts)
  await getExecutions('hello-world');

  // Uncomment to delete functions
  // await deleteFunction('hello-world');
  // await deleteFunction('error-handling');

  console.log('\n' + '='.repeat(60));
  console.log('✨ All examples completed!');
  console.log('='.repeat(60) + '\n');
  console.log('💡 Tip: Run "npm run invoke" to test invoking these functions');
  console.log('');
}

// Run examples
main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});

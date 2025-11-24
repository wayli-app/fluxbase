/**
 * Edge Function Invocation Example
 *
 * This example demonstrates how to:
 * 1. Initialize the Fluxbase client
 * 2. Invoke edge functions
 * 3. Handle responses and errors
 * 4. Work with different function types
 */

import { createClient } from '@fluxbase/sdk';
import dotenv from 'dotenv';

// Load environment variables
dotenv.config();

// Validate environment variables
const FLUXBASE_URL = process.env.FLUXBASE_URL;
const FLUXBASE_ANON_KEY = process.env.FLUXBASE_ANON_KEY;

if (!FLUXBASE_URL || !FLUXBASE_ANON_KEY) {
  console.error('❌ Missing required environment variables');
  console.error('Please set FLUXBASE_URL and FLUXBASE_ANON_KEY in your .env file');
  process.exit(1);
}

// Initialize Fluxbase client
const client = createClient(FLUXBASE_URL, FLUXBASE_ANON_KEY, {
  // Optional: Enable debug logging
  debug: process.env.DEBUG === 'true',
});

/**
 * Example 1: Simple function invocation
 */
async function invokeHelloWorld() {
  console.log('\n📦 Example 1: Invoking hello-world function\n');

  try {
    const { data, error } = await client.functions.invoke('hello-world', {
      body: JSON.stringify({ name: 'Alice' }),
    });

    if (error) {
      console.error('❌ Error:', error);
      if (error.details) {
        console.error('Details:', error.details);
      }
      return;
    }

    console.log('✅ Success!');
    console.log('Response:', JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 2: Error handling - Invalid JSON
 */
async function testInvalidJSON() {
  console.log('\n📦 Example 2: Testing invalid JSON handling\n');

  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: 'This is not valid JSON',
    });

    if (error) {
      console.error('❌ Error (expected):', error);
      return;
    }

    console.log('Response:', JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 3: Error handling - Missing required field
 */
async function testValidationError() {
  console.log('\n📦 Example 3: Testing validation error\n');

  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: JSON.stringify({}), // Missing 'action' field
    });

    if (error) {
      console.error('❌ Error (expected):', error);
      return;
    }

    const response = JSON.parse(data.body);
    console.log('Response:', response);
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 4: Successful operation
 */
async function testSuccessfulOperation() {
  console.log('\n📦 Example 4: Testing successful operation\n');

  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: JSON.stringify({ action: 'success' }),
    });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Success!');
    console.log('Response:', JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 5: Division operation with error handling
 */
async function testDivision() {
  console.log('\n📦 Example 5: Testing division operation\n');

  // Test successful division
  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: JSON.stringify({
        action: 'divide',
        numerator: 10,
        denominator: 2,
      }),
    });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Division result:');
    console.log(JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }

  // Test division by zero
  console.log('\nTesting division by zero...\n');
  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: JSON.stringify({
        action: 'divide',
        numerator: 10,
        denominator: 0,
      }),
    });

    if (error) {
      console.error('❌ Error (expected):', error);
      return;
    }

    const response = JSON.parse(data.body);
    console.log('Response:', response);
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 6: Simulated runtime error
 */
async function testRuntimeError() {
  console.log('\n📦 Example 6: Testing runtime error\n');

  try {
    const { data, error } = await client.functions.invoke('error-handling', {
      body: JSON.stringify({ action: 'simulate_error' }),
    });

    if (error) {
      console.error('❌ Error (expected):', error);
      // The error should include request_id for debugging
      if (error.request_id) {
        console.error('🔍 Request ID for debugging:', error.request_id);
      }
      // Logs from the function execution
      if (error.logs) {
        console.error('📋 Function logs:', error.logs);
      }
      return;
    }

    console.log('Response:', JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Example 7: Custom headers
 */
async function invokeWithCustomHeaders() {
  console.log('\n📦 Example 7: Invoking with custom headers\n');

  try {
    const { data, error } = await client.functions.invoke('hello-world', {
      headers: {
        'X-Custom-Header': 'my-value',
        'X-Request-ID': 'custom-request-123',
      },
      body: JSON.stringify({ name: 'Bob' }),
    });

    if (error) {
      console.error('❌ Error:', error);
      return;
    }

    console.log('✅ Success!');
    console.log('Response:', JSON.parse(data.body));
  } catch (err) {
    console.error('❌ Unexpected error:', err);
  }
}

/**
 * Run all examples
 */
async function main() {
  console.log('='.repeat(60));
  console.log('🚀 Fluxbase Edge Functions - Invocation Examples');
  console.log('='.repeat(60));

  await invokeHelloWorld();
  await testInvalidJSON();
  await testValidationError();
  await testSuccessfulOperation();
  await testDivision();
  await testRuntimeError();
  await invokeWithCustomHeaders();

  console.log('\n' + '='.repeat(60));
  console.log('✨ All examples completed!');
  console.log('='.repeat(60) + '\n');
}

// Run examples
main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});

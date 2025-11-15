#!/usr/bin/env node
/**
 * E2E test for SDK aggregations and batch operations
 * Tests against real Fluxbase backend
 */

import { createClient } from '../sdk/dist/index.js'

const API_URL = process.env.FLUXBASE_BASE_URL || 'http://localhost:8080'

async function main() {
  console.log('🧪 Testing SDK Aggregations & Batch Operations\n')
  console.log(`API URL: ${API_URL}\n`)

  const client = createClient({
    url: API_URL,
    debug: true,
  })

  let passedTests = 0
  let failedTests = 0

  // Helper function for assertions
  function assert(condition, message) {
    if (condition) {
      console.log(`✅ ${message}`)
      passedTests++
    } else {
      console.error(`❌ ${message}`)
      failedTests++
    }
  }

  try {
    // Test 1: Count all products
    console.log('\n📊 Test 1: Count all products')
    const countResult = await client
      .from('products')
      .count('*')
      .execute()

    console.log('Count result:', JSON.stringify(countResult.data, null, 2))
    assert(countResult.data !== null, 'Count query returned data')
    assert(!countResult.error, 'Count query had no error')

    // Test 2: Count with filter
    console.log('\n📊 Test 2: Count products with filter')
    const filteredCountResult = await client
      .from('products')
      .count('*')
      .gte('price', 100)
      .execute()

    console.log('Filtered count result:', JSON.stringify(filteredCountResult.data, null, 2))
    assert(filteredCountResult.data !== null, 'Filtered count returned data')

    // Test 3: Sum aggregation
    console.log('\n📊 Test 3: Sum of prices')
    const sumResult = await client
      .from('products')
      .sum('price')
      .execute()

    console.log('Sum result:', JSON.stringify(sumResult.data, null, 2))
    assert(sumResult.data !== null, 'Sum query returned data')

    // Test 4: Average aggregation
    console.log('\n📊 Test 4: Average price')
    const avgResult = await client
      .from('products')
      .avg('price')
      .execute()

    console.log('Average result:', JSON.stringify(avgResult.data, null, 2))
    assert(avgResult.data !== null, 'Average query returned data')

    // Test 5: Min/Max aggregations
    console.log('\n📊 Test 5: Min and Max price')
    const minResult = await client
      .from('products')
      .min('price')
      .execute()

    const maxResult = await client
      .from('products')
      .max('price')
      .execute()

    console.log('Min result:', JSON.stringify(minResult.data, null, 2))
    console.log('Max result:', JSON.stringify(maxResult.data, null, 2))
    assert(minResult.data !== null, 'Min query returned data')
    assert(maxResult.data !== null, 'Max query returned data')

    // Test 6: Batch insert
    console.log('\n📦 Test 6: Batch insert')
    const testProducts = [
      { name: 'Test Product 1', price: 10, description: 'SDK Test' },
      { name: 'Test Product 2', price: 20, description: 'SDK Test' },
      { name: 'Test Product 3', price: 30, description: 'SDK Test' },
    ]

    const insertResult = await client
      .from('products')
      .insertMany(testProducts)

    console.log(`Inserted ${insertResult.count} products`)
    assert(insertResult.count === 3, 'Batch insert created 3 products')
    assert(Array.isArray(insertResult.data), 'Batch insert returned array')

    // Test 7: Batch update
    console.log('\n📦 Test 7: Batch update')
    const updateResult = await client
      .from('products')
      .like('description', '%SDK Test%')
      .updateMany({ price: 999 })

    console.log('Update result:', JSON.stringify(updateResult, null, 2))
    assert(!updateResult.error, 'Batch update had no error')

    // Test 8: Batch delete
    console.log('\n📦 Test 8: Batch delete (cleanup test data)')
    const deleteResult = await client
      .from('products')
      .like('description', '%SDK Test%')
      .deleteMany()

    console.log('Delete result:', deleteResult.status)
    assert(deleteResult.status === 204, 'Batch delete succeeded')

    // Test 9: Upsert (insert with conflict handling)
    console.log('\n📦 Test 9: Upsert operation')
    const upsertData = [
      { id: 999999, name: 'Upsert Test', price: 100, description: 'Will be upserted' },
    ]

    const upsertResult = await client
      .from('products')
      .upsert(upsertData)

    console.log('Upsert result:', upsertResult.status)
    assert(!upsertResult.error, 'Upsert had no error')

    // Cleanup upsert test
    await client.from('products').eq('id', 999999).delete()

  } catch (error) {
    console.error('\n❌ Test failed with error:', error.message)
    if (error.response) {
      console.error('Response:', error.response.data)
    }
    failedTests++
  }

  // Summary
  console.log('\n' + '='.repeat(50))
  console.log(`✅ Passed: ${passedTests}`)
  console.log(`❌ Failed: ${failedTests}`)
  console.log(`📊 Total:  ${passedTests + failedTests}`)
  console.log('='.repeat(50))

  process.exit(failedTests > 0 ? 1 : 0)
}

main()

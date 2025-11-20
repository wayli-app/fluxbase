/**
 * Fluxbase SDK - Quickstart Example
 *
 * This example demonstrates basic usage of the Fluxbase SDK including:
 * - Client initialization
 * - Authentication
 * - Database queries
 * - Realtime subscriptions
 * - File storage
 */

import { createClient } from '@fluxbase/sdk'

// Initialize the client
const client = createClient({
  url: 'http://localhost:8080',
  auth: {
    autoRefresh: true,  // Automatically refresh expired tokens
    persist: true,       // Persist auth session in localStorage
  },
  timeout: 30000,        // Request timeout in milliseconds
  debug: false,          // Enable debug logging
})

async function main() {
  console.log('🚀 Fluxbase SDK Quickstart Example\n')

  // ========================================
  // 1. AUTHENTICATION
  // ========================================
  console.log('1️⃣  Authentication')

  try {
    // Sign up a new user with metadata
    const { data, error } = await client.auth.signUp({
      email: 'demo@example.com',
      password: 'secure-password-123',
      options: {
        data: {
          name: 'Demo User',
          role: 'developer',
        },
      },
    })

    if (error) throw error

    console.log('✅ User signed up:', data.user.email)
    if (data.session) {
      console.log('✅ Access token:', data.session.access_token.substring(0, 20) + '...')
    } else {
      console.log('📧 Email confirmation required - check your inbox')
    }
  } catch (error: any) {
    // User might already exist, try signing in
    if (error.message.includes('already exists')) {
      const { data, error: signInError } = await client.auth.signIn({
        email: 'demo@example.com',
        password: 'secure-password-123',
      })
      if (signInError) throw signInError
      console.log('✅ User signed in:', data.user.email)
      console.log('✅ Access token:', data.session.access_token.substring(0, 20) + '...')
    } else {
      throw error
    }
  }

  // ========================================
  // 2. DATABASE QUERIES
  // ========================================
  console.log('\n2️⃣  Database Queries')

  // Create a products table entry
  const { data: newProduct, error: insertError } = await client
    .from('products')
    .insert({
      name: 'Laptop',
      price: 1299.99,
      category: 'electronics',
      in_stock: true,
    })
    .execute()

  if (insertError) {
    console.log('⚠️  Insert failed (table might not exist):', insertError.message)
  } else {
    console.log('✅ Product created:', newProduct)
  }

  // Query products with advanced filters
  const { data: products, error: selectError} = await client
    .from('products')
    .select('*')
    .eq('category', 'electronics')
    .gte('price', 1000)
    .not('status', 'eq', 'discontinued')  // NOT operator
    .order('price', 'desc')
    .limit(10)
    .execute()

  if (selectError) {
    console.log('⚠️  Query failed:', selectError.message)
  } else {
    console.log('✅ Found', products?.length || 0, 'electronics products')
  }

  // Advanced query with OR and match
  const { data: activeProducts } = await client
    .from('products')
    .match({ category: 'electronics', in_stock: true })  // match() shorthand
    .or('status.eq.active,status.eq.pending')  // OR operator
    .execute()

  if (activeProducts) {
    console.log('✅ Active/pending products:', activeProducts.length)
  }

  // Query with AND operator
  const { data: verifiedProducts } = await client
    .from('products')
    .and('verified.eq.true,in_stock.eq.true')  // AND operator
    .gte('rating', 4.0)
    .execute()

  if (verifiedProducts) {
    console.log('✅ Verified in-stock products:', verifiedProducts.length)
  }

  // maybeSingle() - returns null if not found (doesn't error)
  const { data: product, error: productError } = await client
    .from('products')
    .eq('id', 99999)  // Non-existent ID
    .maybeSingle()

  if (!productError && product === null) {
    console.log('✅ maybeSingle() returned null for non-existent product')
  }

  // throwOnError() - throws error instead of returning { data, error }
  try {
    const topProduct = await client
      .from('products')
      .select('*')
      .order('price', { ascending: false })
      .limit(1)
      .throwOnError()  // Returns data directly or throws
    console.log('✅ throwOnError() returned top product directly')
  } catch (error: any) {
    console.log('⚠️  Query with throwOnError failed:', error.message)
  }

  // Upsert with options
  const { data: upsertedProduct } = await client
    .from('products')
    .upsert(
      { name: 'Tablet', price: 599.99, category: 'electronics' },
      { onConflict: 'name', ignoreDuplicates: false }  // Update on name conflict
    )

  if (upsertedProduct) {
    console.log('✅ Product upserted with conflict resolution')
  }

  // ========================================
  // 3. AGGREGATIONS
  // ========================================
  console.log('\n3️⃣  Aggregations')

  const { data: stats, error: statsError } = await client
    .from('products')
    .select('category')
    .count('*')
    .groupBy('category')
    .execute()

  if (statsError) {
    console.log('⚠️  Aggregation failed:', statsError.message)
  } else {
    console.log('✅ Product stats by category:', stats)
  }

  // ========================================
  // 4. REALTIME SUBSCRIPTIONS
  // ========================================
  console.log('\n4️⃣  Realtime Subscriptions')

  // Subscribe to product changes
  const channel = client.realtime
    .channel('table:public.products')
    .on('INSERT', (payload) => {
      console.log('🆕 New product:', payload.new_record)
    })
    .on('UPDATE', (payload) => {
      console.log('📝 Product updated:', payload.new_record)
    })
    .on('DELETE', (payload) => {
      console.log('🗑️  Product deleted:', payload.old_record)
    })
    .subscribe()

  console.log('✅ Subscribed to product changes')

  // Wait a bit to receive any pending messages
  await new Promise(resolve => setTimeout(resolve, 2000))

  // Unsubscribe
  channel.unsubscribe()
  console.log('✅ Unsubscribed from product changes')

  // ========================================
  // 5. FILE STORAGE
  // ========================================
  console.log('\n5️⃣  File Storage')

  // Create a bucket
  try {
    await client.storage.createBucket('avatars', {
      public: true,
    })
    console.log('✅ Bucket created: avatars')
  } catch (error: any) {
    console.log('⚠️  Bucket creation failed (might already exist):', error.message)
  }

  // Upload a file
  const fileContent = new Blob(['Hello, Fluxbase!'], { type: 'text/plain' })
  try {
    const { data: uploadData, error: uploadError } = await client.storage
      .from('avatars')
      .upload('demo.txt', fileContent)

    if (uploadError) {
      console.log('⚠️  Upload failed:', uploadError.message)
    } else {
      console.log('✅ File uploaded:', uploadData.path)
    }
  } catch (error: any) {
    console.log('⚠️  Upload failed:', error.message)
  }

  // List files
  try {
    const { data: files, error: listError } = await client.storage
      .from('avatars')
      .list()

    if (listError) {
      console.log('⚠️  List failed:', listError.message)
    } else {
      console.log('✅ Files in bucket:', files?.length || 0)
    }
  } catch (error: any) {
    console.log('⚠️  List failed:', error.message)
  }

  // ========================================
  // 6. RPC FUNCTION CALLS
  // ========================================
  console.log('\n6️⃣  RPC Function Calls')

  const { data: sessionData } = await client.auth.getSession()
  const { data: rpcResult, error: rpcError } = await client.rpc('get_user_stats', {
    user_id: sessionData.session?.user?.id,
  })

  if (rpcError) {
    console.log('⚠️  RPC call failed (function might not exist):', rpcError.message)
  } else {
    console.log('✅ RPC result:', rpcResult)
  }

  // ========================================
  // 7. SIGN OUT
  // ========================================
  console.log('\n7️⃣  Sign Out')

  await client.auth.signOut()
  console.log('✅ User signed out')

  console.log('\n✨ Quickstart complete!')
}

// Run the example
main().catch(console.error)

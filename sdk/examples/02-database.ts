/**
 * Fluxbase SDK - Database Operations Example
 *
 * This example demonstrates advanced database operations including:
 * - CRUD operations
 * - Filtering and sorting
 * - Aggregations
 * - Batch operations
 * - Transactions
 * - Type safety with TypeScript
 */

import { createClient } from '@fluxbase/sdk'

// Define your database schema types for full type safety
interface Product {
  id?: number
  name: string
  description?: string
  price: number
  category: string
  in_stock: boolean
  created_at?: string
  updated_at?: string
}

interface Order {
  id?: number
  user_id: string
  product_id: number
  quantity: number
  total_price: number
  status: 'pending' | 'processing' | 'shipped' | 'delivered'
  created_at?: string
}

const client = createClient({
  url: 'http://localhost:8080',
  auth: {
    token: 'your-jwt-token', // Or use signIn/signUp
  },
})

async function databaseOperations() {
  console.log('📊 Database Operations Example\n')

  // ========================================
  // 1. INSERT OPERATIONS
  // ========================================
  console.log('1️⃣  Insert Operations')

  // Single insert
  const { data: product, error: insertError } = await client
    .from<Product>('products')
    .insert({
      name: 'Gaming Mouse',
      description: 'RGB gaming mouse with 16000 DPI',
      price: 79.99,
      category: 'gaming',
      in_stock: true,
    })
    .execute()

  if (insertError) {
    console.error('❌ Insert failed:', insertError)
  } else {
    console.log('✅ Product inserted:', product)
  }

  // Bulk insert
  const { data: products, error: bulkError } = await client
    .from<Product>('products')
    .insert([
      { name: 'Keyboard', price: 129.99, category: 'gaming', in_stock: true },
      { name: 'Headset', price: 99.99, category: 'gaming', in_stock: true },
      { name: 'Monitor', price: 399.99, category: 'gaming', in_stock: false },
    ])
    .execute()

  console.log('✅ Bulk insert complete:', products?.length || 0, 'products')

  // ========================================
  // 2. SELECT QUERIES
  // ========================================
  console.log('\n2️⃣  Select Queries')

  // Simple select all
  const { data: allProducts } = await client
    .from<Product>('products')
    .select('*')
    .execute()

  console.log('✅ All products:', allProducts?.length || 0)

  // Select specific columns
  const { data: names } = await client
    .from<Product>('products')
    .select('id, name, price')
    .execute()

  console.log('✅ Product names:', names?.length || 0)

  // Single row
  const { data: singleProduct } = await client
    .from<Product>('products')
    .select('*')
    .eq('id', 1)
    .single()

  console.log('✅ Single product:', singleProduct?.name)

  // ========================================
  // 3. FILTERING
  // ========================================
  console.log('\n3️⃣  Filtering')

  // Equality filter
  const { data: electronics } = await client
    .from<Product>('products')
    .select('*')
    .eq('category', 'electronics')
    .execute()

  console.log('✅ Electronics:', electronics?.length || 0)

  // Comparison filters
  const { data: expensive } = await client
    .from<Product>('products')
    .select('*')
    .gte('price', 100)
    .lt('price', 500)
    .execute()

  console.log('✅ Products $100-$500:', expensive?.length || 0)

  // Multiple filters (AND)
  const { data: available} = await client
    .from<Product>('products')
    .select('*')
    .eq('in_stock', true)
    .eq('category', 'gaming')
    .gte('price', 50)
    .execute()

  console.log('✅ Available gaming products > $50:', available?.length || 0)

  // IN filter
  const { data: categories } = await client
    .from<Product>('products')
    .select('*')
    .in('category', ['gaming', 'electronics', 'accessories'])
    .execute()

  console.log('✅ Products in multiple categories:', categories?.length || 0)

  // LIKE filter
  const { data: searchResults } = await client
    .from<Product>('products')
    .select('*')
    .like('name', '%mouse%')
    .execute()

  console.log('✅ Search "mouse":', searchResults?.length || 0)

  // IS NULL filter
  const { data: noDescription } = await client
    .from<Product>('products')
    .select('*')
    .is('description', null)
    .execute()

  console.log('✅ Products without description:', noDescription?.length || 0)

  // ========================================
  // 4. SORTING
  // ========================================
  console.log('\n4️⃣  Sorting')

  // Sort by price descending
  const { data: byPrice } = await client
    .from<Product>('products')
    .select('*')
    .order('price', 'desc')
    .execute()

  console.log('✅ Sorted by price (high to low):', byPrice?.[0]?.price)

  // Multiple sort columns
  const { data: multiSort } = await client
    .from<Product>('products')
    .select('*')
    .order('category', 'asc')
    .order('price', 'desc')
    .execute()

  console.log('✅ Sorted by category then price')

  // ========================================
  // 5. PAGINATION
  // ========================================
  console.log('\n5️⃣  Pagination')

  // Limit
  const { data: page1 } = await client
    .from<Product>('products')
    .select('*')
    .limit(10)
    .execute()

  console.log('✅ First 10 products:', page1?.length || 0)

  // Offset (page 2)
  const { data: page2 } = await client
    .from<Product>('products')
    .select('*')
    .limit(10)
    .offset(10)
    .execute()

  console.log('✅ Products 11-20:', page2?.length || 0)

  // ========================================
  // 6. UPDATE OPERATIONS
  // ========================================
  console.log('\n6️⃣  Update Operations')

  // Update single record
  const { data: updated } = await client
    .from<Product>('products')
    .update({
      price: 89.99,
      in_stock: false,
    })
    .eq('id', 1)
    .execute()

  console.log('✅ Product updated:', updated)

  // Bulk update
  const { data: bulkUpdated } = await client
    .from<Product>('products')
    .update({
      in_stock: true,
    })
    .eq('category', 'gaming')
    .execute()

  console.log('✅ Bulk update complete:', bulkUpdated?.length || 0, 'products')

  // ========================================
  // 7. UPSERT OPERATIONS
  // ========================================
  console.log('\n7️⃣  Upsert Operations')

  const { data: upserted } = await client
    .from<Product>('products')
    .upsert({
      id: 1,
      name: 'Updated Gaming Mouse',
      price: 79.99,
      category: 'gaming',
      in_stock: true,
    })
    .execute()

  console.log('✅ Product upserted:', upserted)

  // ========================================
  // 8. DELETE OPERATIONS
  // ========================================
  console.log('\n8️⃣  Delete Operations')

  // Delete with filter
  const { error: deleteError } = await client
    .from<Product>('products')
    .delete()
    .eq('in_stock', false)
    .eq('price', 0)
    .execute()

  if (!deleteError) {
    console.log('✅ Deleted out-of-stock free products')
  }

  // ========================================
  // 9. AGGREGATIONS
  // ========================================
  console.log('\n9️⃣  Aggregations')

  // Count
  const { data: count } = await client
    .from<Product>('products')
    .count('*')
    .execute()

  console.log('✅ Total products:', count)

  // Count with filter
  const { data: inStockCount } = await client
    .from<Product>('products')
    .count('*')
    .eq('in_stock', true)
    .execute()

  console.log('✅ In-stock products:', inStockCount)

  // Group by
  const { data: categoryStats } = await client
    .from<Product>('products')
    .select('category')
    .count('*')
    .avg('price')
    .groupBy('category')
    .execute()

  console.log('✅ Stats by category:', categoryStats)

  // Multiple aggregations
  const { data: priceStats } = await client
    .from<Product>('products')
    .select('category')
    .count('*')
    .sum('price')
    .avg('price')
    .min('price')
    .max('price')
    .groupBy('category')
    .execute()

  console.log('✅ Price statistics:', priceStats)

  // ========================================
  // 10. BATCH OPERATIONS
  // ========================================
  console.log('\n🔟 Batch Operations')

  // Batch insert with insertMany alias
  const { data: batch } = await client
    .from<Product>('products')
    .insertMany([
      { name: 'Item 1', price: 9.99, category: 'accessories', in_stock: true },
      { name: 'Item 2', price: 19.99, category: 'accessories', in_stock: true },
      { name: 'Item 3', price: 29.99, category: 'accessories', in_stock: true },
    ])

  console.log('✅ Batch inserted:', batch?.length || 0, 'products')

  // Batch update with updateMany alias
  const { data: batchUpdated } = await client
    .from<Product>('products')
    .eq('category', 'accessories')
    .updateMany({
      in_stock: false,
    })

  console.log('✅ Batch updated:', batchUpdated?.length || 0, 'products')

  // Batch delete with deleteMany alias
  const { data: batchDeleted } = await client
    .from<Product>('products')
    .eq('category', 'accessories')
    .eq('in_stock', false)
    .deleteMany()

  console.log('✅ Batch deleted:', batchDeleted?.length || 0, 'products')

  console.log('\n✨ Database operations complete!')
}

// Run the example
databaseOperations().catch(console.error)

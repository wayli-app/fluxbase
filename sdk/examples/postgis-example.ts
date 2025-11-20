/**
 * Fluxbase SDK - PostGIS / Geospatial Example
 *
 * This example demonstrates PostGIS spatial query capabilities:
 * - Creating tables with geometry columns
 * - Inserting geospatial data using GeoJSON
 * - Querying with spatial operators
 */

import { createClient } from '@fluxbase/sdk'
import type { GeoJSONPoint, GeoJSONPolygon } from '@fluxbase/sdk'

// Initialize the client
const client = createClient({
  url: 'http://localhost:8080',
  auth: {
    autoRefresh: true,
    persist: true,
  },
})

async function main() {
  console.log('🗺️  Fluxbase PostGIS / Geospatial Example\n')

  // ========================================
  // 1. AUTHENTICATION
  // ========================================
  console.log('1️⃣  Authentication')

  try {
    const { data, error } = await client.auth.signIn({
      email: 'demo@example.com',
      password: 'secure-password-123',
    })

    if (error) throw error
    console.log('✅ User signed in:', data.user.email)
  } catch (error: any) {
    console.log('⚠️  Sign in failed:', error.message)
    return
  }

  // ========================================
  // 2. CREATE TABLE WITH GEOMETRY COLUMN
  // ========================================
  console.log('\n2️⃣  Create Geospatial Table')

  // Note: You would typically create this table via SQL:
  // CREATE TABLE locations (
  //   id SERIAL PRIMARY KEY,
  //   name TEXT NOT NULL,
  //   location GEOMETRY(Point, 4326),
  //   area GEOMETRY(Polygon, 4326)
  // );

  console.log('ℹ️  Create table with: CREATE TABLE locations (id, name, location GEOMETRY(Point, 4326))')

  // ========================================
  // 3. INSERT GEOSPATIAL DATA
  // ========================================
  console.log('\n3️⃣  Insert Locations with GeoJSON')

  const locations = [
    {
      name: 'Golden Gate Bridge',
      location: {
        type: 'Point',
        coordinates: [-122.4783, 37.8199], // [longitude, latitude]
      } as GeoJSONPoint,
    },
    {
      name: 'Alcatraz Island',
      location: {
        type: 'Point',
        coordinates: [-122.4230, 37.8267],
      } as GeoJSONPoint,
    },
    {
      name: 'Fishermans Wharf',
      location: {
        type: 'Point',
        coordinates: [-122.4177, 37.8080],
      } as GeoJSONPoint,
    },
  ]

  try {
    const { data, error } = await client
      .from('locations')
      .insert(locations)
      .execute()

    if (error) {
      console.log('⚠️  Insert failed (table may not exist):', error.message)
    } else {
      console.log('✅ Inserted', locations.length, 'locations with geospatial data')
    }
  } catch (error: any) {
    console.log('⚠️  Insert error:', error.message)
  }

  // ========================================
  // 4. SPATIAL QUERIES
  // ========================================
  console.log('\n4️⃣  Spatial Queries')

  // Define a search area (polygon around downtown SF)
  const searchArea: GeoJSONPolygon = {
    type: 'Polygon',
    coordinates: [
      [
        [-122.5, 37.7], // Southwest corner
        [-122.5, 37.85], // Northwest corner
        [-122.35, 37.85], // Northeast corner
        [-122.35, 37.7], // Southeast corner
        [-122.5, 37.7], // Close the polygon
      ],
    ],
  }

  // Query 1: Find locations that intersect with search area
  console.log('\n📍 Query: Locations that intersect search area')
  try {
    const { data: intersecting, error } = await client
      .from('locations')
      .select('*')
      .intersects('location', searchArea)
      .execute()

    if (error) {
      console.log('⚠️  Query failed:', error.message)
    } else {
      console.log('✅ Found', intersecting?.length || 0, 'intersecting locations')
      intersecting?.forEach((loc: any) => {
        console.log('   -', loc.name, loc.location)
      })
    }
  } catch (error: any) {
    console.log('⚠️  Query error:', error.message)
  }

  // Query 2: Find locations within a specific region
  const region: GeoJSONPolygon = {
    type: 'Polygon',
    coordinates: [
      [
        [-122.5, 37.8],
        [-122.5, 37.83],
        [-122.4, 37.83],
        [-122.4, 37.8],
        [-122.5, 37.8],
      ],
    ],
  }

  console.log('\n📍 Query: Locations within specific region')
  try {
    const { data: withinRegion, error } = await client
      .from('locations')
      .select('name, location')
      .within('location', region)
      .execute()

    if (error) {
      console.log('⚠️  Query failed:', error.message)
    } else {
      console.log('✅ Found', withinRegion?.length || 0, 'locations within region')
      withinRegion?.forEach((loc: any) => {
        console.log('   -', loc.name)
      })
    }
  } catch (error: any) {
    console.log('⚠️  Query error:', error.message)
  }

  // Query 3: Check if a region contains specific points
  console.log('\n📍 Query: Regions that contain a specific point')

  const testPoint: GeoJSONPoint = {
    type: 'Point',
    coordinates: [-122.42, 37.82],
  }

  try {
    const { data: containingRegions, error } = await client
      .from('regions')
      .select('*')
      .stContains('boundary', testPoint)
      .execute()

    if (error) {
      console.log('⚠️  Query failed (table may not exist):', error.message)
    } else {
      console.log('✅ Found', containingRegions?.length || 0, 'regions containing the point')
    }
  } catch (error: any) {
    console.log('⚠️  Query error:', error.message)
  }

  // ========================================
  // 5. RETURN GEOSPATIAL DATA AS GEOJSON
  // ========================================
  console.log('\n5️⃣  Retrieve GeoJSON Data')

  try {
    const { data: allLocations, error } = await client
      .from('locations')
      .select('id, name, location')
      .limit(5)
      .execute()

    if (error) {
      console.log('⚠️  Query failed:', error.message)
    } else {
      console.log('✅ Retrieved', allLocations?.length || 0, 'locations')
      allLocations?.forEach((loc: any) => {
        console.log('   -', loc.name, ':', JSON.stringify(loc.location))
      })
    }
  } catch (error: any) {
    console.log('⚠️  Query error:', error.message)
  }

  // ========================================
  // 6. SIGN OUT
  // ========================================
  console.log('\n6️⃣  Sign Out')
  await client.auth.signOut()
  console.log('✅ User signed out')

  console.log('\n✨ PostGIS example complete!')
  console.log('\n💡 Usage Notes:')
  console.log('   - Geometry data is stored in PostGIS format')
  console.log('   - API returns/accepts GeoJSON format')
  console.log('   - Spatial queries use PostGIS functions (ST_Intersects, etc.)')
  console.log('   - Coordinates are [longitude, latitude] per GeoJSON spec')
}

// Run the example
main().catch(console.error)

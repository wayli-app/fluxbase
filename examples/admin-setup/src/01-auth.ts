/**
 * Example 1: Admin Authentication
 *
 * This example demonstrates:
 * - Admin login with email and password
 * - Token management
 * - Getting admin user info
 * - Token refresh
 * - Logout
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('Admin Authentication Example')

  const client = getClient()

  try {
    // Step 1: Admin Login
    logger.step(1, 'Admin Login')
    await authenticateAdmin(client)
    logger.success('Admin authenticated successfully')

    // Step 2: Get Admin User Info
    logger.step(2, 'Get Admin User Info')
    const adminInfo = await client.admin.me()
    logger.success('Admin info retrieved')
    logger.item(`User ID: ${adminInfo.user.id}`)
    logger.item(`Email: ${adminInfo.user.email}`)
    logger.item(`Role: ${adminInfo.user.role}`)
    logger.item(`Created: ${new Date(adminInfo.user.created_at).toLocaleDateString()}`)

    // Step 3: Verify Admin Status
    logger.step(3, 'Verify Admin Status')
    const setupStatus = await client.admin.setupStatus()
    logger.success('Setup status checked')
    logger.item(`Setup complete: ${setupStatus.setup_complete}`)
    logger.item(`Has admin: ${setupStatus.has_admin}`)

    // Step 4: Token Refresh (if needed)
    logger.step(4, 'Token Refresh')
    logger.info('Token refresh is handled automatically by the SDK')
    logger.info('You can also manually refresh if needed:')
    logger.info('await client.admin.refreshToken({ refresh_token: token })')

    logger.section('✅ Authentication Example Complete')
    logger.info('Key takeaways:')
    logger.item('Admin login requires email and password')
    logger.item('Tokens are managed automatically by the SDK')
    logger.item('Use me() to get current admin user info')
    logger.item('Check setupStatus() to verify admin configuration')

  } catch (error) {
    logger.error('Authentication failed', error)
    process.exit(1)
  }
}

main().catch(console.error)

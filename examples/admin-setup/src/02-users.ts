/**
 * Example 2: User Management
 *
 * This example demonstrates:
 * - Listing users with pagination
 * - Searching users
 * - Inviting new users
 * - Updating user roles
 * - Resetting user passwords
 * - Deleting users
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('User Management Example')

  const client = getClient()

  try {
    // Authenticate
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: List All Users
    logger.step(1, 'List All Users')
    const { users, total } = await client.admin.listUsers({
      limit: 10,
      offset: 0
    })
    logger.success(`Found ${total} users`)
    users.forEach((user, index) => {
      logger.item(`${index + 1}. ${user.email} (${user.role})`)
    })

    // Step 2: Search for Specific User
    logger.step(2, 'Search for User')
    const searchEmail = users[0]?.email || 'test@example.com'
    const searchResults = await client.admin.listUsers({
      search: searchEmail,
      limit: 5
    })
    logger.success(`Search found ${searchResults.users.length} users matching "${searchEmail}"`)

    // Step 3: Invite New User
    logger.step(3, 'Invite New User')
    try {
      const invitation = await client.admin.inviteUser({
        email: `newuser-${Date.now()}@example.com`,
        role: 'user'
      })
      logger.success('User invited successfully')
      logger.item(`Invitation ID: ${invitation.invitation.id}`)
      logger.item(`Invite URL: ${invitation.invite_url}`)
      logger.item(`Expires: ${new Date(invitation.invitation.expires_at).toLocaleString()}`)
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn('User already exists or invitation already sent')
      } else {
        throw error
      }
    }

    // Step 4: Update User Role (if users exist)
    if (users.length > 0) {
      logger.step(4, 'Update User Role')
      const targetUser = users.find(u => u.role === 'user')

      if (targetUser) {
        logger.info(`Updating role for: ${targetUser.email}`)
        await client.admin.updateUserRole(targetUser.id, { role: 'user' })
        logger.success('User role updated')
      } else {
        logger.warn('No suitable user found for role update demo')
      }
    }

    // Step 5: Reset User Password
    if (users.length > 0) {
      logger.step(5, 'Reset User Password')
      const targetUser = users[0]
      logger.info(`Resetting password for: ${targetUser.email}`)

      const result = await client.admin.resetUserPassword(targetUser.id)
      logger.success('Password reset initiated')
      logger.item(`Reset URL: ${result.reset_url}`)
      logger.info('User will receive an email with reset instructions')
    }

    // Step 6: Filter Users by Role
    logger.step(6, 'Filter Users by Role')
    const adminUsers = await client.admin.listUsers({
      role: 'admin'
    })
    logger.success(`Found ${adminUsers.users.length} admin users`)

    const regularUsers = await client.admin.listUsers({
      role: 'user'
    })
    logger.success(`Found ${regularUsers.users.length} regular users`)

    // Step 7: Pagination Example
    logger.step(7, 'Pagination Example')
    logger.info('Fetching users in pages of 5...')
    let offset = 0
    const limit = 5
    let pageNum = 1

    while (offset < Math.min(total, 15)) { // Limit to first 3 pages
      const page = await client.admin.listUsers({ limit, offset })
      logger.info(`Page ${pageNum}: ${page.users.length} users`)
      page.users.forEach(u => logger.item(`  • ${u.email}`))
      offset += limit
      pageNum++
    }

    logger.section('✅ User Management Example Complete')
    logger.info('Key takeaways:')
    logger.item('Use listUsers() with pagination for large user bases')
    logger.item('Search users by email with the search parameter')
    logger.item('Invite users with specific roles')
    logger.item('Update roles as needed for access control')
    logger.item('Reset passwords for users who forgot credentials')
    logger.item('Filter by role to find specific user types')

  } catch (error) {
    logger.error('User management failed', error)
    process.exit(1)
  }
}

main().catch(console.error)

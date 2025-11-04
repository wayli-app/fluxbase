/**
 * Example 7: User Impersonation
 *
 * This example demonstrates:
 * - Impersonating specific users
 * - Impersonating anonymous users
 * - Impersonating with service role
 * - Viewing audit trail
 * - Testing RLS policies
 * - Debugging user issues
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('User Impersonation Example')

  const client = getClient()

  try {
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: Check Current Impersonation Status
    logger.step(1, 'Check Current Impersonation Status')
    const current = await client.admin.impersonation.getCurrent()
    if (current.session) {
      logger.info(`Currently impersonating: ${current.target_user?.email || 'anonymous'}`)
    } else {
      logger.info('No active impersonation session')
    }

    // Step 2: List Users to Impersonate
    logger.step(2, 'Find User to Impersonate')
    const { users } = await client.admin.listUsers({ limit: 5 })
    logger.success(`Found ${users.length} users`)

    if (users.length === 0) {
      logger.warn('No users found - create some users first')
      logger.info('Skipping user impersonation demo')
    } else {
      users.forEach((u, i) => {
        logger.item(`${i + 1}. ${u.email} (${u.role})`)
      })

      // Step 3: Impersonate Specific User
      logger.step(3, 'Impersonate Specific User')
      const targetUser = users[0]
      logger.info(`Impersonating: ${targetUser.email}`)

      const impersonation = await client.admin.impersonation.impersonateUser({
        target_user_id: targetUser.id,
        reason: 'Example: Testing user permissions and RLS policies'
      })

      logger.success('Impersonation started')
      logger.item(`Session ID: ${impersonation.session.id}`)
      logger.item(`Target: ${impersonation.target_user?.email}`)
      logger.item(`Role: ${impersonation.session.target_role}`)
      logger.item(`Started: ${new Date(impersonation.session.started_at).toLocaleString()}`)

      // Simulate queries as the user
      logger.info('Now all database queries use this user\'s permissions')
      logger.info('RLS policies are enforced based on their role')

      // Wait a moment
      await new Promise(resolve => setTimeout(resolve, 1000))

      // Step 4: Stop Impersonation
      logger.step(4, 'Stop Impersonation')
      await client.admin.impersonation.stop()
      logger.success('Impersonation ended')
      logger.info('Returned to admin context')
    }

    // Step 5: Impersonate Anonymous User
    logger.step(5, 'Impersonate Anonymous User')
    logger.info('Testing public data access...')

    const anonImpersonation = await client.admin.impersonation.impersonateAnon({
      reason: 'Example: Testing public data access policies'
    })

    logger.success('Anonymous impersonation started')
    logger.item(`Session ID: ${anonImpersonation.session.id}`)
    logger.item(`Type: ${anonImpersonation.session.impersonation_type}`)
    logger.info('Now viewing data as unauthenticated visitor')

    await new Promise(resolve => setTimeout(resolve, 1000))

    await client.admin.impersonation.stop()
    logger.success('Anonymous impersonation ended')

    // Step 6: Impersonate with Service Role
    logger.step(6, 'Impersonate with Service Role')
    logger.info('Using elevated permissions...')

    const serviceImpersonation = await client.admin.impersonation.impersonateService({
      reason: 'Example: Administrative data operations'
    })

    logger.success('Service role impersonation started')
    logger.item(`Session ID: ${serviceImpersonation.session.id}`)
    logger.item(`Type: ${serviceImpersonation.session.impersonation_type}`)
    logger.info('Now have service-level access (may bypass RLS)')

    await new Promise(resolve => setTimeout(resolve, 1000))

    await client.admin.impersonation.stop()
    logger.success('Service impersonation ended')

    // Step 7: View Impersonation Audit Trail
    logger.step(7, 'View Impersonation Audit Trail')
    const { sessions, total } = await client.admin.impersonation.listSessions({
      limit: 10
    })

    logger.success(`Found ${total} total impersonation sessions`)
    logger.info('Recent sessions:')

    sessions.slice(0, 5).forEach(s => {
      const targetDisplay = s.target_user_id ? 'User' : s.impersonation_type
      const ended = s.ended_at ? 'Ended' : 'Active'
      logger.item(`${targetDisplay} - ${ended}`)
      logger.item(`  Reason: ${s.reason}`)
      logger.item(`  Started: ${new Date(s.started_at).toLocaleString()}`)
      if (s.ended_at) {
        logger.item(`  Ended: ${new Date(s.ended_at).toLocaleString()}`)
      }
    })

    // Step 8: Filter Audit Trail
    logger.step(8, 'Filter Audit Trail')
    logger.info('Filtering sessions by type...')

    const userSessions = await client.admin.impersonation.listSessions({
      impersonation_type: 'user',
      limit: 5
    })
    logger.success(`Found ${userSessions.sessions.length} user impersonation sessions`)

    const activeSessions = await client.admin.impersonation.listSessions({
      is_active: true
    })
    logger.success(`Found ${activeSessions.sessions.length} active sessions`)

    // Step 9: Use Case Examples
    logger.step(9, 'Common Use Cases')
    logger.info('When to use impersonation:')
    logger.item('1. Debug user-reported issues')
    logger.item('2. Test RLS policies as different roles')
    logger.item('3. Verify public data access')
    logger.item('4. Customer support investigations')
    logger.item('5. QA testing with different permissions')
    logger.item('6. Audit security policies')

    logger.section('✅ Impersonation Example Complete')
    logger.info('Key takeaways:')
    logger.item('Impersonate users to see data as they see it')
    logger.item('Test anonymous access with impersonateAnon()')
    logger.item('Use service role for admin operations')
    logger.item('Always provide a reason for accountability')
    logger.item('Complete audit trail of all sessions')
    logger.item('Stop impersonation when done')

  } catch (error) {
    logger.error('Impersonation failed', error)

    // Make sure to stop any active impersonation
    try {
      await client.admin.impersonation.stop()
    } catch {
      // Ignore errors if no session is active
    }

    process.exit(1)
  }
}

main().catch(console.error)

/**
 * Example 6: client keys & Webhooks Management
 *
 * This example demonstrates:
 * - Creating client keys
 * - Listing and managing client keys
 * - Revoking keys
 * - Creating webhooks
 * - Webhook delivery tracking
 * - Retrying failed deliveries
 * - Managing invitations
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('client keys & Webhooks Management Example')

  const client = getClient()

  try {
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: List Existing client keys
    logger.step(1, 'List Existing client keys')
    const { keys } = await client.admin.management.apiKeys.list()
    logger.success(`Found ${keys.length} client keys`)
    keys.forEach(k => {
      logger.item(`${k.name} (${k.id.substring(0, 8)}...)`)
      logger.item(`  Created: ${new Date(k.created_at).toLocaleString()}`)
      logger.item(`  Expires: ${k.expires_at ? new Date(k.expires_at).toLocaleString() : 'Never'}`)
    })

    // Step 2: Create Backend Service API Key
    logger.step(2, 'Create Backend Service API Key')
    const backendKey = await client.admin.management.apiKeys.create({
      name: `Backend Service ${Date.now()}`,
      description: 'API key for internal backend service',
      expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString() // 1 year
    })
    logger.success('Backend service API key created')
    logger.item(`Key ID: ${backendKey.key.id}`)
    logger.item(`Key: ${backendKey.key.key}`)
    logger.warn('⚠️  Save this key securely - it will not be shown again!')

    // Step 3: Create Short-Lived API Key
    logger.step(3, 'Create Short-Lived API Key')
    const tempKey = await client.admin.management.apiKeys.create({
      name: `Temporary Key ${Date.now()}`,
      description: 'Short-lived API key for testing',
      expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString() // 7 days
    })
    logger.success('Temporary API key created')
    logger.item(`Expires in 7 days`)

    // Step 4: Update API Key
    logger.step(4, 'Update API Key')
    await client.admin.management.apiKeys.update(tempKey.key.id, {
      name: `Updated Temp Key ${Date.now()}`,
      description: 'Updated description'
    })
    logger.success('API key updated')

    // Step 5: List Webhooks
    logger.step(5, 'List Existing Webhooks')
    const { webhooks } = await client.admin.management.webhooks.list()
    logger.success(`Found ${webhooks.length} webhooks`)
    webhooks.forEach(w => {
      logger.item(`${w.name}: ${w.enabled ? '✅ enabled' : '❌ disabled'}`)
      logger.item(`  URL: ${w.url}`)
      logger.item(`  Events: ${w.events.join(', ')}`)
      logger.item(`  Table: ${w.schema}.${w.table}`)
    })

    // Step 6: Create Webhook
    logger.step(6, 'Create Webhook')
    try {
      const webhook = await client.admin.management.webhooks.create({
        name: `Test Webhook ${Date.now()}`,
        url: 'https://httpbin.org/post',
        events: ['INSERT', 'UPDATE', 'DELETE'],
        table: 'users',
        schema: 'public',
        enabled: true,
        secret: `webhook_secret_${Math.random().toString(36).substring(7)}`
      })
      logger.success('Webhook created')
      logger.item(`Webhook ID: ${webhook.webhook.id}`)
      logger.item(`URL: ${webhook.webhook.url}`)
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn('Webhook already exists')
      } else {
        throw error
      }
    }

    // Step 7: View Webhook Deliveries
    if (webhooks.length > 0) {
      logger.step(7, 'View Webhook Deliveries')
      const webhook = webhooks[0]
      const { deliveries } = await client.admin.management.webhooks.getDeliveries(webhook.id, {
        limit: 10
      })
      logger.success(`Found ${deliveries.length} deliveries for ${webhook.name}`)

      if (deliveries.length > 0) {
        deliveries.slice(0, 3).forEach(d => {
          logger.item(`${d.status} - ${new Date(d.created_at).toLocaleString()}`)
          logger.item(`  Response: ${d.response_status_code}`)
        })
      }
    }

    // Step 8: Retry Failed Delivery
    if (webhooks.length > 0) {
      logger.step(8, 'Retry Failed Delivery (Example)')
      logger.info('To retry a failed webhook delivery:')
      logger.info('await client.admin.management.webhooks.retryDelivery(webhookId, deliveryId)')
    }

    // Step 9: List Invitations
    logger.step(9, 'List Invitations')
    const { invitations } = await client.admin.management.invitations.list()
    logger.success(`Found ${invitations.length} invitations`)

    if (invitations.length > 0) {
      invitations.slice(0, 3).forEach(inv => {
        logger.item(`${inv.email} (${inv.role})`)
        logger.item(`  Status: ${inv.accepted ? 'Accepted' : 'Pending'}`)
        logger.item(`  Expires: ${new Date(inv.expires_at).toLocaleString()}`)
      })
    }

    // Step 10: Revoke API Key
    logger.step(10, 'Revoke API Key')
    await client.admin.management.apiKeys.revoke(tempKey.key.id)
    logger.success('Temporary API key revoked')
    logger.info('Revoked keys can no longer be used for authentication')

    logger.section('✅ Management Example Complete')
    logger.info('Key takeaways:')
    logger.item('Create client keys for backend services')
    logger.item('Set expiration dates for security')
    logger.item('Webhooks notify you of database events')
    logger.item('Monitor webhook delivery status')
    logger.item('Retry failed deliveries')
    logger.item('Manage invitations for new users')
    logger.item('Revoke compromised keys immediately')

  } catch (error) {
    logger.error('Management operations failed', error)
    process.exit(1)
  }
}

main().catch(console.error)

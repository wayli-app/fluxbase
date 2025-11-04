/**
 * Example 4: Settings Management
 *
 * This example demonstrates:
 * - System settings (key-value storage)
 * - Application settings (feature toggles, security)
 * - Feature flags
 * - Custom configuration storage
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('Settings Management Example')

  const client = getClient()

  try {
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: List System Settings
    logger.step(1, 'List System Settings')
    const { settings } = await client.admin.settings.system.list()
    logger.success(`Found ${settings.length} system settings`)

    if (settings.length > 0) {
      settings.slice(0, 5).forEach(s => {
        logger.item(`${s.key}: ${JSON.stringify(s.value).substring(0, 50)}...`)
      })
    }

    // Step 2: Store Custom Configuration
    logger.step(2, 'Store Custom Configuration')
    await client.admin.settings.system.update('app.features', {
      value: {
        beta_mode: true,
        new_ui: false,
        analytics: true,
        ai_assistant: false
      },
      description: 'Feature flags for application'
    })
    logger.success('Feature flags stored')

    // Step 3: Store API Quotas
    logger.step(3, 'Store API Quotas')
    await client.admin.settings.system.update('api.quotas', {
      value: {
        free: { requests: 10000, storage_mb: 100 },
        pro: { requests: 100000, storage_mb: 1000 },
        enterprise: { requests: 1000000, storage_mb: 10000 }
      },
      description: 'API usage quotas by plan'
    })
    logger.success('API quotas stored')

    // Step 4: Get Specific Setting
    logger.step(4, 'Get Specific Setting')
    const features = await client.admin.settings.system.get('app.features')
    logger.success('Feature flags retrieved')
    logger.data('Features', features.value)

    // Step 5: Get Application Settings
    logger.step(5, 'Get Application Settings')
    const appSettings = await client.admin.settings.app.get()
    logger.success('App settings retrieved')
    logger.item(`Realtime: ${appSettings.features?.enable_realtime ? 'enabled' : 'disabled'}`)
    logger.item(`Storage: ${appSettings.features?.enable_storage ? 'enabled' : 'disabled'}`)
    logger.item(`Functions: ${appSettings.features?.enable_functions ? 'enabled' : 'disabled'}`)
    logger.item(`Rate limiting: ${appSettings.security?.enable_rate_limiting ? 'enabled' : 'disabled'}`)

    // Step 6: Update Application Settings
    logger.step(6, 'Update Application Settings')
    await client.admin.settings.app.update({
      features: {
        enable_realtime: true,
        enable_storage: true,
        enable_functions: false
      },
      security: {
        enable_rate_limiting: true,
        rate_limit_requests_per_minute: 60
      }
    })
    logger.success('App settings updated')

    // Step 7: Feature Flag System Example
    logger.step(7, 'Feature Flag System')
    await client.admin.settings.system.update('features.rollout', {
      value: {
        'new-dashboard': { enabled: true, percentage: 100 },
        'beta-api': { enabled: true, percentage: 10 },
        'ai-features': { enabled: false, percentage: 0 }
      },
      description: 'Gradual feature rollout configuration'
    })
    logger.success('Feature rollout configuration stored')

    // Step 8: Delete Setting
    logger.step(8, 'Delete Setting (Example)')
    logger.info('To delete a setting:')
    logger.info('await client.admin.settings.system.delete("key")')
    logger.warn('Be careful - this permanently removes the setting')

    logger.section('✅ Settings Management Example Complete')
    logger.info('Key takeaways:')
    logger.item('Use system settings for flexible key-value storage')
    logger.item('Store feature flags for gradual rollouts')
    logger.item('Configure app-wide settings (features, security)')
    logger.item('Store API quotas, limits, and configuration')
    logger.item('Settings persist across restarts')
    logger.item('Use JSON values for complex configuration')

  } catch (error) {
    logger.error('Settings management failed', error)
    process.exit(1)
  }
}

main().catch(console.error)

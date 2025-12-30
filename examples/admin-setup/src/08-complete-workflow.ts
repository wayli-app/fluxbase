/**
 * Complete Admin Workflow Example
 *
 * This example demonstrates a complete admin workflow that uses all advanced features:
 * 1. Admin authentication
 * 2. Configure OAuth providers
 * 3. Update authentication settings
 * 4. Configure application settings
 * 5. Create multi-tenant database schema
 * 6. Generate client keys
 * 7. Set up webhooks
 * 8. Store custom configuration
 * 9. Test with user impersonation
 */

import { getClient, authenticateAdmin, getEnv } from './utils/client.js'
import { logger } from './utils/logger.js'
import type { FluxbaseClient } from '@fluxbase/sdk'

async function main() {
  logger.section('Complete Admin Workflow Example')

  try {
    // Step 1: Authentication
    logger.step(1, 'Authenticate as Admin')
    const client = getClient()
    await authenticateAdmin(client)
    logger.success('Admin authenticated successfully')

    // Step 2: Configure OAuth Providers
    await configureOAuth(client)

    // Step 3: Configure Authentication Settings
    await configureAuthSettings(client)

    // Step 4: Configure Application Settings
    await configureAppSettings(client)

    // Step 5: Create Multi-Tenant Database
    await createTenantDatabase(client)

    // Step 6: Generate client keys
    await generateAPIKeys(client)

    // Step 7: Set Up Webhooks
    await setupWebhooks(client)

    // Step 8: Store Custom Configuration
    await storeCustomConfig(client)

    // Step 9: Test with User Impersonation
    await testImpersonation(client)

    logger.section('✅ Complete Workflow Finished Successfully')
    logger.info('All admin features have been configured and tested')
    logger.info('Your Fluxbase instance is now fully set up!')

  } catch (error) {
    logger.error('Workflow failed', error)
    process.exit(1)
  }
}

async function configureOAuth(client: FluxbaseClient) {
  logger.step(2, 'Configure OAuth Providers')

  try {
    // Check if GitHub credentials are available
    const githubClientId = getEnv('GITHUB_CLIENT_ID')
    const githubClientSecret = getEnv('GITHUB_CLIENT_SECRET')

    if (githubClientId && githubClientSecret) {
      logger.info('Configuring GitHub OAuth provider...')

      const github = await client.admin.oauth.providers.createProvider({
        provider_name: 'github',
        display_name: 'GitHub',
        enabled: true,
        client_id: githubClientId,
        client_secret: githubClientSecret,
        redirect_url: `${getEnv('APP_URL', false) || 'http://localhost:3000'}/auth/callback/github`,
        scopes: ['user:email', 'read:user'],
        is_custom: false
      })

      logger.success('GitHub OAuth configured')
      logger.item(`Provider ID: ${github.id}`)
    } else {
      logger.warn('Skipping GitHub OAuth (credentials not set)')
    }

    // Check if Google credentials are available
    const googleClientId = getEnv('GOOGLE_CLIENT_ID')
    const googleClientSecret = getEnv('GOOGLE_CLIENT_SECRET')

    if (googleClientId && googleClientSecret) {
      logger.info('Configuring Google OAuth provider...')

      const google = await client.admin.oauth.providers.createProvider({
        provider_name: 'google',
        display_name: 'Google',
        enabled: true,
        client_id: googleClientId,
        client_secret: googleClientSecret,
        redirect_url: `${getEnv('APP_URL', false) || 'http://localhost:3000'}/auth/callback/google',
        scopes: ['openid', 'email', 'profile'],
        is_custom: false
      })

      logger.success('Google OAuth configured')
      logger.item(`Provider ID: ${google.id}`)
    } else {
      logger.warn('Skipping Google OAuth (credentials not set)')
    }

    // List all providers
    const providers = await client.admin.oauth.providers.listProviders()
    logger.info(`Total OAuth providers: ${providers.length}`)
    providers.forEach(p => {
      logger.item(`${p.display_name}: ${p.enabled ? 'enabled' : 'disabled'}`)
    })

  } catch (error: any) {
    if (error.status === 409) {
      logger.warn('OAuth providers already exist, skipping...')
    } else {
      throw error
    }
  }
}

async function configureAuthSettings(client: FluxbaseClient) {
  logger.step(3, 'Configure Authentication Settings')

  await client.admin.oauth.authSettings.update({
    // Enable signup
    enable_signup: true,

    // Email verification
    require_email_verification: true,
    enable_magic_link: true,

    // Strong password requirements
    password_min_length: 12,
    password_require_uppercase: true,
    password_require_lowercase: true,
    password_require_number: true,
    password_require_special: true,

    // Session management
    session_timeout_minutes: 240, // 4 hours
    max_sessions_per_user: 5
  })

  logger.success('Authentication settings configured')
  logger.item('Password min length: 12')
  logger.item('Session timeout: 240 minutes')
  logger.item('Max sessions per user: 5')
}

async function configureAppSettings(client: FluxbaseClient) {
  logger.step(4, 'Configure Application Settings')

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

  logger.success('Application settings configured')
  logger.item('Realtime: enabled')
  logger.item('Storage: enabled')
  logger.item('Functions: disabled')
  logger.item('Rate limiting: 60 req/min')
}

async function createTenantDatabase(client: FluxbaseClient) {
  logger.step(5, 'Create Multi-Tenant Database Schema')

  const tenantName = 'acme_corp'
  const schemaName = `tenant_${tenantName}`

  try {
    // Create schema
    logger.info(`Creating schema: ${schemaName}`)
    await client.admin.ddl.createSchema(schemaName)
    logger.success(`Schema created: ${schemaName}`)

    // Create users table
    logger.info('Creating users table...')
    await client.admin.ddl.createTable(schemaName, 'users', [
      { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
      { name: 'email', type: 'CITEXT', nullable: false },
      { name: 'name', type: 'TEXT', nullable: false },
      { name: 'role', type: 'TEXT', defaultValue: "'member'" },
      { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
      { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
    ])
    logger.success('Users table created')

    // Create projects table
    logger.info('Creating projects table...')
    await client.admin.ddl.createTable(schemaName, 'projects', [
      { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
      { name: 'name', type: 'TEXT', nullable: false },
      { name: 'description', type: 'TEXT' },
      { name: 'owner_id', type: 'UUID', nullable: false },
      { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
      { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
    ])
    logger.success('Projects table created')

    // List all tables in schema
    const tables = await client.admin.ddl.listTables(schemaName)
    logger.info(`Tables in ${schemaName}: ${tables.length}`)
    tables.forEach(t => {
      logger.item(`${t.table_schema}.${t.table_name}`)
    })

  } catch (error: any) {
    if (error.status === 409) {
      logger.warn('Schema or tables already exist, skipping...')
    } else {
      throw error
    }
  }
}

async function generateAPIKeys(client: FluxbaseClient) {
  logger.step(6, 'Generate client keys')

  try {
    // Create backend service key
    const backendKey = await client.admin.management.apiKeys.create({
      name: 'Backend Service',
      description: 'API key for internal backend service',
      expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString() // 1 year
    })

    logger.success('Backend service API key created')
    logger.item(`Key ID: ${backendKey.key.id}`)
    logger.item(`Key: ${backendKey.key.key}`)
    logger.warn('⚠️  Save this key securely - it will not be shown again!')

    // Create integration key
    const integrationKey = await client.admin.management.apiKeys.create({
      name: 'External Integration',
      description: 'API key for third-party integration',
      expires_at: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString() // 90 days
    })

    logger.success('Integration API key created')
    logger.item(`Key ID: ${integrationKey.key.id}`)

    // List all keys
    const { keys } = await client.admin.management.apiKeys.list()
    logger.info(`Total client keys: ${keys.length}`)

  } catch (error: any) {
    if (error.status === 409) {
      logger.warn('client keys may already exist')
    } else {
      throw error
    }
  }
}

async function setupWebhooks(client: FluxbaseClient) {
  logger.step(7, 'Set Up Webhooks')

  try {
    // Create webhook for user events
    const userWebhook = await client.admin.management.webhooks.create({
      name: 'User Events Webhook',
      url: 'https://example.com/webhooks/users',
      events: ['INSERT', 'UPDATE', 'DELETE'],
      table: 'users',
      schema: 'public',
      enabled: true,
      secret: 'webhook-secret-key-' + Math.random().toString(36).substring(7)
    })

    logger.success('User events webhook created')
    logger.item(`Webhook ID: ${userWebhook.webhook.id}`)
    logger.item(`URL: ${userWebhook.webhook.url}`)

    // Create webhook for order events
    const orderWebhook = await client.admin.management.webhooks.create({
      name: 'Order Events Webhook',
      url: 'https://example.com/webhooks/orders',
      events: ['INSERT'],
      table: 'orders',
      schema: 'public',
      enabled: true,
      secret: 'webhook-secret-key-' + Math.random().toString(36).substring(7)
    })

    logger.success('Order events webhook created')
    logger.item(`Webhook ID: ${orderWebhook.webhook.id}`)

    // List all webhooks
    const { webhooks } = await client.admin.management.webhooks.list()
    logger.info(`Total webhooks: ${webhooks.length}`)

  } catch (error: any) {
    if (error.status === 409) {
      logger.warn('Webhooks may already exist')
    } else {
      throw error
    }
  }
}

async function storeCustomConfig(client: FluxbaseClient) {
  logger.step(8, 'Store Custom Configuration')

  // Store feature flags
  await client.admin.settings.system.update('app.feature_flags', {
    value: {
      beta_features: true,
      new_dashboard: false,
      advanced_analytics: true,
      ai_assistant: false
    },
    description: 'Feature flags for gradual rollout'
  })
  logger.success('Feature flags stored')

  // Store tenant configuration
  await client.admin.settings.system.update('tenants.acme_corp', {
    value: {
      name: 'Acme Corporation',
      plan: 'enterprise',
      max_users: 100,
      schema: 'tenant_acme_corp',
      created_at: new Date().toISOString()
    },
    description: 'Configuration for Acme Corp tenant'
  })
  logger.success('Tenant configuration stored')

  // Store API quotas
  await client.admin.settings.system.update('api.quotas', {
    value: {
      free: { requests_per_month: 10000, storage_gb: 1 },
      pro: { requests_per_month: 100000, storage_gb: 10 },
      enterprise: { requests_per_month: 1000000, storage_gb: 100 }
    },
    description: 'API usage quotas by plan'
  })
  logger.success('API quotas stored')

  // List all system settings
  const { settings } = await client.admin.settings.system.list()
  logger.info(`Total system settings: ${settings.length}`)
}

async function testImpersonation(client: FluxbaseClient) {
  logger.step(9, 'Test User Impersonation')

  try {
    // Get a user to impersonate
    const { users } = await client.admin.listUsers({ limit: 1 })

    if (users.length === 0) {
      logger.warn('No users found to impersonate, skipping test')
      return
    }

    const targetUser = users[0]
    logger.info(`Impersonating user: ${targetUser.email}`)

    // Start impersonation
    const impersonation = await client.admin.impersonation.impersonateUser({
      target_user_id: targetUser.id,
      reason: 'Testing admin workflow - demonstrating impersonation feature'
    })

    logger.success('Impersonation started')
    logger.item(`Session ID: ${impersonation.session.id}`)
    logger.item(`Target user: ${impersonation.target_user?.email}`)
    logger.item(`Started at: ${impersonation.session.started_at}`)

    // Simulate some queries as the user
    logger.info('Running queries as impersonated user...')
    await new Promise(resolve => setTimeout(resolve, 1000))

    // Stop impersonation
    await client.admin.impersonation.stop()
    logger.success('Impersonation ended')

    // View audit trail
    const { sessions } = await client.admin.impersonation.listSessions({
      limit: 5,
      is_active: false
    })

    logger.info(`Recent impersonation sessions: ${sessions.length}`)
    sessions.slice(0, 3).forEach(session => {
      logger.item(`${session.impersonation_type} - ${session.reason}`)
    })

  } catch (error) {
    logger.warn('Impersonation test failed (this is OK if no users exist)')
  }
}

// Run the workflow
main().catch(console.error)

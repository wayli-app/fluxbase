/**
 * Example 3: OAuth Provider Configuration
 *
 * This example demonstrates:
 * - Listing OAuth providers
 * - Creating OAuth providers (GitHub, Google, custom)
 * - Updating provider settings
 * - Enabling/disabling providers
 * - Configuring authentication settings
 * - Password policies
 * - Session management
 */

import { getClient, authenticateAdmin, getEnv } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('OAuth Configuration Example')

  const client = getClient()

  try {
    // Authenticate
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: List Existing Providers
    logger.step(1, 'List Existing OAuth Providers')
    const providers = await client.admin.oauth.providers.listProviders()
    logger.success(`Found ${providers.length} OAuth providers`)

    if (providers.length > 0) {
      providers.forEach(p => {
        logger.item(`${p.display_name}: ${p.enabled ? '✅ enabled' : '❌ disabled'}`)
        logger.item(`  Provider: ${p.provider_name}`)
        logger.item(`  Custom: ${p.is_custom ? 'Yes' : 'No'}`)
        logger.item(`  Scopes: ${p.scopes.join(', ')}`)
      })
    } else {
      logger.info('No OAuth providers configured yet')
    }

    // Step 2: Create GitHub OAuth Provider
    logger.step(2, 'Create GitHub OAuth Provider')
    const githubClientId = getEnv('GITHUB_CLIENT_ID')
    const githubClientSecret = getEnv('GITHUB_CLIENT_SECRET')

    if (githubClientId && githubClientSecret) {
      try {
        const github = await client.admin.oauth.providers.createProvider({
          provider_name: 'github',
          display_name: 'GitHub',
          enabled: true,
          client_id: githubClientId,
          client_secret: githubClientSecret,
          redirect_url: `${getEnv('APP_URL') || 'http://localhost:3000'}/auth/callback/github`,
          scopes: ['user:email', 'read:user'],
          is_custom: false
        })

        logger.success('GitHub OAuth provider created')
        logger.item(`Provider ID: ${github.id}`)
      } catch (error: any) {
        if (error.status === 409) {
          logger.warn('GitHub provider already exists')
        } else {
          throw error
        }
      }
    } else {
      logger.warn('GitHub credentials not set in .env - skipping')
      logger.info('Set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET to enable')
    }

    // Step 3: Create Google OAuth Provider
    logger.step(3, 'Create Google OAuth Provider')
    const googleClientId = getEnv('GOOGLE_CLIENT_ID')
    const googleClientSecret = getEnv('GOOGLE_CLIENT_SECRET')

    if (googleClientId && googleClientSecret) {
      try {
        const google = await client.admin.oauth.providers.createProvider({
          provider_name: 'google',
          display_name: 'Google',
          enabled: true,
          client_id: googleClientId,
          client_secret: googleClientSecret,
          redirect_url: `${getEnv('APP_URL') || 'http://localhost:3000'}/auth/callback/google`,
          scopes: ['openid', 'email', 'profile'],
          is_custom: false
        })

        logger.success('Google OAuth provider created')
        logger.item(`Provider ID: ${google.id}`)
      } catch (error: any) {
        if (error.status === 409) {
          logger.warn('Google provider already exists')
        } else {
          throw error
        }
      }
    } else {
      logger.warn('Google credentials not set in .env - skipping')
    }

    // Step 4: Create Custom OAuth2 Provider
    logger.step(4, 'Create Custom OAuth2 Provider (Example)')
    logger.info('Example of custom OAuth2 provider configuration:')
    logger.data('Custom Provider', {
      provider_name: 'custom_sso',
      display_name: 'Custom SSO',
      enabled: true,
      client_id: 'your-client-id',
      client_secret: 'your-client-secret',
      redirect_url: 'https://yourapp.com/auth/callback/custom',
      scopes: ['openid', 'profile', 'email'],
      is_custom: true,
      authorization_url: 'https://sso.example.com/oauth/authorize',
      token_url: 'https://sso.example.com/oauth/token',
      user_info_url: 'https://sso.example.com/oauth/userinfo'
    })

    // Step 5: Configure Authentication Settings
    logger.step(5, 'Configure Authentication Settings')
    const authSettings = await client.admin.oauth.authSettings.get()
    logger.success('Current authentication settings retrieved')
    logger.item(`Signup enabled: ${authSettings.enable_signup}`)
    logger.item(`Email verification: ${authSettings.require_email_verification}`)
    logger.item(`Magic link: ${authSettings.enable_magic_link}`)
    logger.item(`Password min length: ${authSettings.password_min_length}`)
    logger.item(`Session timeout: ${authSettings.session_timeout_minutes} minutes`)

    // Step 6: Update Password Policy
    logger.step(6, 'Update Password Policy')
    await client.admin.oauth.authSettings.update({
      password_min_length: 12,
      password_require_uppercase: true,
      password_require_lowercase: true,
      password_require_number: true,
      password_require_special: true
    })
    logger.success('Password policy updated')
    logger.item('Min length: 12 characters')
    logger.item('Required: uppercase, lowercase, number, special char')

    // Step 7: Configure Session Management
    logger.step(7, 'Configure Session Management')
    await client.admin.oauth.authSettings.update({
      session_timeout_minutes: 240, // 4 hours
      max_sessions_per_user: 5
    })
    logger.success('Session management configured')
    logger.item('Session timeout: 240 minutes (4 hours)')
    logger.item('Max concurrent sessions: 5')

    // Step 8: Enable/Disable Provider
    if (providers.length > 0) {
      logger.step(8, 'Enable/Disable Provider')
      const targetProvider = providers[0]
      logger.info(`Current status: ${targetProvider.enabled ? 'enabled' : 'disabled'}`)

      // Toggle status
      if (targetProvider.enabled) {
        await client.admin.oauth.providers.disableProvider(targetProvider.id)
        logger.success('Provider disabled')
      } else {
        await client.admin.oauth.providers.enableProvider(targetProvider.id)
        logger.success('Provider enabled')
      }

      // Restore original status
      if (targetProvider.enabled) {
        await client.admin.oauth.providers.enableProvider(targetProvider.id)
      } else {
        await client.admin.oauth.providers.disableProvider(targetProvider.id)
      }
      logger.info('Status restored to original')
    }

    logger.section('✅ OAuth Configuration Example Complete')
    logger.info('Key takeaways:')
    logger.item('Configure built-in providers (GitHub, Google, etc.)')
    logger.item('Create custom OAuth2 providers for enterprise SSO')
    logger.item('Set strong password policies for security')
    logger.item('Configure session timeouts and limits')
    logger.item('Enable/disable providers without deleting them')
    logger.item('Users can now sign in with configured providers')

  } catch (error) {
    logger.error('OAuth configuration failed', error)
    process.exit(1)
  }
}

main().catch(console.error)

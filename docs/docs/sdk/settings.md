---
title: Settings SDK
sidebar_position: 5
---

# Settings SDK

The Settings SDK provides comprehensive tools for managing system and application settings in your Fluxbase instance. These features allow you to:

- **System Settings**: Low-level key-value storage for custom configuration
- **Application Settings**: Type-safe management of authentication, features, email, and security settings
- **Convenience Methods**: Quick access to commonly modified settings

:::info
Settings management requires admin authentication. All operations in this guide assume you have logged in as an admin user.
:::

## Installation

The settings module is included with the Fluxbase SDK:

```bash
npm install @fluxbase/sdk
```

## Quick Start

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient(
  'http://localhost:8080',
  'your-service-role-key'
)

// Authenticate as admin
await client.admin.login({
  email: 'admin@example.com',
  password: 'admin-password'
})

// Get current application settings
const settings = await client.admin.settings.app.get()
console.log('Signup enabled:', settings.authentication.enable_signup)

// Update settings
await client.admin.settings.app.update({
  authentication: {
    enable_signup: false,
    password_min_length: 12
  },
  security: {
    enable_global_rate_limit: true
  }
})

// Use convenience methods
await client.admin.settings.app.enableSignup()
await client.admin.settings.app.setPasswordMinLength(16)
await client.admin.settings.app.setRateLimiting(true)
```

---

## Application Settings

Application settings provide a type-safe, structured way to manage common configuration options for your Fluxbase instance.

### Get Application Settings

Retrieve all application settings.

```typescript
const settings = await client.admin.settings.app.get()

console.log('Authentication:', settings.authentication)
// {
//   enable_signup: true,
//   enable_magic_link: true,
//   password_min_length: 12,
//   require_email_verification: true
// }

console.log('Features:', settings.features)
// {
//   enable_realtime: true,
//   enable_storage: true,
//   enable_functions: false
// }

console.log('Email:', settings.email)
// {
//   enabled: true,
//   provider: 'smtp'
// }

console.log('Security:', settings.security)
// {
//   enable_global_rate_limit: true
// }
```

**Returns:** Complete application settings structure with:
- `authentication`: Authentication and user management settings
- `features`: Feature flags for optional modules
- `email`: Email service configuration
- `security`: Security and rate limiting settings

### Update Application Settings

Update one or more application settings. You can update partial settings - only the fields you specify will be changed.

```typescript
// Update multiple settings
const updated = await client.admin.settings.app.update({
  authentication: {
    enable_signup: false,
    password_min_length: 16
  },
  features: {
    enable_realtime: true
  }
})

// Update single setting group
await client.admin.settings.app.update({
  security: {
    enable_global_rate_limit: true
  }
})
```

**Parameters:**
- `authentication` (optional): Partial authentication settings
- `features` (optional): Partial feature flags
- `email` (optional): Partial email configuration
- `security` (optional): Partial security settings

**Returns:** Updated application settings

### Reset Application Settings

Reset all application settings to their default values.

```typescript
const defaults = await client.admin.settings.app.reset()

// Settings are now reset to defaults:
// - Authentication: signup enabled, 12 char min password
// - Features: all enabled
// - Email: disabled
// - Security: rate limiting enabled
```

**Returns:** Application settings reset to defaults

---

## Convenience Methods

The Settings SDK provides convenience methods for commonly modified settings, allowing quick updates without dealing with the full settings structure.

### Enable/Disable Signup

Control whether new users can sign up for your application.

```typescript
// Enable user signup
await client.admin.settings.app.enableSignup()

// Disable user signup
await client.admin.settings.app.disableSignup()
```

**Use Cases:**
- Close registrations during maintenance
- Implement invite-only registration
- Temporarily restrict new user growth

### Set Password Minimum Length

Configure the minimum password length requirement (8-128 characters).

```typescript
// Set minimum password length to 16 characters
await client.admin.settings.app.setPasswordMinLength(16)

// Set to minimum allowed (8 characters)
await client.admin.settings.app.setPasswordMinLength(8)
```

**Parameters:**
- `length` (required): Minimum password length (8-128)

**Validation:**
- Throws error if length < 8 or length > 128
- Provides clear error messages for invalid values

### Set Feature Flags

Enable or disable optional features like realtime, storage, and functions.

```typescript
// Enable realtime subscriptions
await client.admin.settings.app.setFeature('realtime', true)

// Disable storage
await client.admin.settings.app.setFeature('storage', false)

// Enable functions
await client.admin.settings.app.setFeature('functions', true)
```

**Parameters:**
- `feature` (required): One of `'realtime'`, `'storage'`, or `'functions'`
- `enabled` (required): Boolean to enable/disable the feature

**Use Cases:**
- Gradually roll out new features
- Disable unused modules to reduce complexity
- Control resource usage

### Set Rate Limiting

Enable or disable global rate limiting for your API.

```typescript
// Enable rate limiting
await client.admin.settings.app.setRateLimiting(true)

// Disable rate limiting
await client.admin.settings.app.setRateLimiting(false)
```

**Parameters:**
- `enabled` (required): Boolean to enable/disable rate limiting

**Note:** This controls the global rate limiter. Individual API key rate limits are managed separately through the [Management SDK](/docs/sdk/management).

---

## System Settings

System settings provide low-level key-value storage for custom configuration. This is useful for storing application-specific settings that don't fit into the structured application settings.

### List System Settings

Retrieve all system settings.

```typescript
const { settings } = await client.admin.settings.system.list()

settings.forEach(setting => {
  console.log(`${setting.key}: ${JSON.stringify(setting.value)}`)
})

// Example output:
// custom.feature.beta: {"enabled": true}
// custom.api.external_url: {"url": "https://api.example.com"}
// custom.limits.max_uploads: {"max": 100}
```

**Returns:** Object containing:
- `settings`: Array of system settings with keys, values, descriptions, and timestamps

### Get System Setting

Retrieve a specific system setting by key.

```typescript
const setting = await client.admin.settings.system.get('custom.feature.beta')

console.log('Key:', setting.key)
console.log('Value:', setting.value)
console.log('Description:', setting.description)
console.log('Created:', setting.created_at)
console.log('Updated:', setting.updated_at)
```

**Parameters:**
- `key` (required): Setting key

**Returns:** System setting object with key, value, description, and timestamps

**Error Handling:**
```typescript
try {
  const setting = await client.admin.settings.system.get('nonexistent.key')
} catch (error) {
  console.error('Setting not found:', error)
}
```

### Update System Setting

Create or update a system setting.

```typescript
// Create new setting
await client.admin.settings.system.update('custom.feature.beta', {
  value: { enabled: true, rollout_percentage: 50 },
  description: 'Beta feature flag with gradual rollout'
})

// Update existing setting
await client.admin.settings.system.update('custom.api.external_url', {
  value: { url: 'https://newapi.example.com', timeout: 5000 }
})

// Update value only
await client.admin.settings.system.update('custom.limits.max_uploads', {
  value: { max: 200 }
})
```

**Parameters:**
- `key` (required): Setting key
- `request` (required): Object containing:
  - `value` (required): Setting value as a JSON object
  - `description` (optional): Human-readable description

**Returns:** Updated system setting

**Key Naming Convention:**
- Use dot notation for namespacing: `custom.feature.beta`
- Start with `custom.` prefix for application-specific settings
- Use lowercase with underscores: `custom.api.external_url`

### Delete System Setting

Permanently delete a system setting.

```typescript
await client.admin.settings.system.delete('custom.feature.beta')

// Setting is now permanently deleted
```

**Parameters:**
- `key` (required): Setting key to delete

**Warning:** This operation is permanent and cannot be undone. The setting and its value will be completely removed.

---

## Settings Structure Reference

### Authentication Settings

Configuration for user authentication and signup.

```typescript
interface AuthenticationSettings {
  enable_signup: boolean              // Allow new user registration
  enable_magic_link: boolean          // Allow passwordless login via email
  password_min_length: number         // Minimum password length (8-128)
  require_email_verification: boolean // Require email verification before access
}
```

**Default Values:**
- `enable_signup`: `true`
- `enable_magic_link`: `true`
- `password_min_length`: `12`
- `require_email_verification`: `true`

### Feature Settings

Feature flags for optional modules.

```typescript
interface FeatureSettings {
  enable_realtime: boolean  // Enable WebSocket realtime subscriptions
  enable_storage: boolean   // Enable file storage API
  enable_functions: boolean // Enable serverless functions (future)
}
```

**Default Values:**
- `enable_realtime`: `true`
- `enable_storage`: `true`
- `enable_functions`: `false`

### Email Settings

Email service configuration.

```typescript
interface EmailSettings {
  enabled: boolean  // Enable email sending
  provider: string  // Email provider ('smtp', 'sendgrid', etc.)
}
```

**Default Values:**
- `enabled`: `false`
- `provider`: `"smtp"`

### Security Settings

Security and rate limiting configuration.

```typescript
interface SecuritySettings {
  enable_global_rate_limit: boolean // Enable global API rate limiting
}
```

**Default Values:**
- `enable_global_rate_limit`: `true`

---

## Complete Settings Object

```typescript
interface AppSettings {
  authentication: AuthenticationSettings
  features: FeatureSettings
  email: EmailSettings
  security: SecuritySettings
}
```

---

## Common Use Cases

### 1. Close Registration During Maintenance

```typescript
// Disable signup before maintenance
await client.admin.settings.app.disableSignup()

// ... perform maintenance ...

// Re-enable signup after maintenance
await client.admin.settings.app.enableSignup()
```

### 2. Implement Invite-Only Registration

```typescript
// Disable public signup
await client.admin.settings.app.disableSignup()

// Invite users through admin API
await client.admin.inviteUser({
  email: 'newuser@example.com',
  role: 'user',
  send_email: true
})
```

### 3. Strengthen Password Requirements

```typescript
// Increase minimum password length for better security
await client.admin.settings.app.setPasswordMinLength(16)

// Verify the change
const settings = await client.admin.settings.app.get()
console.log('New min length:', settings.authentication.password_min_length)
```

### 4. Disable Unused Features

```typescript
// Disable features you're not using
await client.admin.settings.app.update({
  features: {
    enable_storage: false,
    enable_functions: false
  }
})
```

### 5. Store Custom Application Settings

```typescript
// Store custom feature flags
await client.admin.settings.system.update('custom.features.dark_mode', {
  value: { enabled: true, default: 'auto' },
  description: 'Dark mode settings for the application'
})

// Store external service configuration
await client.admin.settings.system.update('custom.services.stripe', {
  value: {
    public_key: 'pk_live_...',
    webhook_secret: 'whsec_...'
  },
  description: 'Stripe API configuration'
})

// Retrieve custom settings
const darkMode = await client.admin.settings.system.get('custom.features.dark_mode')
const stripe = await client.admin.settings.system.get('custom.services.stripe')
```

### 6. Batch Update Multiple Settings

```typescript
// Update multiple settings in one call
await client.admin.settings.app.update({
  authentication: {
    enable_signup: true,
    enable_magic_link: true,
    password_min_length: 14,
    require_email_verification: true
  },
  features: {
    enable_realtime: true,
    enable_storage: true
  },
  security: {
    enable_global_rate_limit: true
  }
})
```

### 7. Reset to Defaults After Testing

```typescript
// After testing, reset all settings to defaults
await client.admin.settings.app.reset()

console.log('Settings reset to defaults')
```

---

## Error Handling

The Settings SDK provides clear error messages for common issues.

### Invalid Password Length

```typescript
try {
  await client.admin.settings.app.setPasswordMinLength(6)
} catch (error) {
  console.error(error.message)
  // "Password minimum length must be between 8 and 128"
}
```

### Setting Not Found

```typescript
try {
  await client.admin.settings.system.get('nonexistent.key')
} catch (error) {
  console.error('Status:', error.status) // 404
  console.error('Message:', error.message)
}
```

### Unauthorized Access

```typescript
try {
  // Attempting to access settings without admin authentication
  await client.admin.settings.app.get()
} catch (error) {
  console.error('Status:', error.status) // 401
  console.error('Message:', error.message) // "Unauthorized"
}
```

### Network Errors

```typescript
try {
  await client.admin.settings.app.update({
    authentication: { enable_signup: false }
  })
} catch (error) {
  if (error.status === 408) {
    console.error('Request timeout')
  } else if (!error.status) {
    console.error('Network error:', error.message)
  }
}
```

---

## Best Practices

### 1. Use Convenience Methods When Possible

Convenience methods provide validation and clearer intent:

```typescript
// ✅ Good: Use convenience method
await client.admin.settings.app.enableSignup()

// ❌ Avoid: Manual update for simple operations
await client.admin.settings.app.update({
  authentication: { enable_signup: true }
})
```

### 2. Batch Related Updates

Reduce API calls by updating multiple settings together:

```typescript
// ✅ Good: Single API call
await client.admin.settings.app.update({
  authentication: {
    enable_signup: false,
    password_min_length: 16
  },
  security: {
    enable_global_rate_limit: true
  }
})

// ❌ Avoid: Multiple API calls
await client.admin.settings.app.disableSignup()
await client.admin.settings.app.setPasswordMinLength(16)
await client.admin.settings.app.setRateLimiting(true)
```

### 3. Namespace Custom Settings

Use clear namespacing for custom system settings:

```typescript
// ✅ Good: Clear namespacing
await client.admin.settings.system.update('custom.payments.stripe.public_key', {
  value: { key: 'pk_live_...' }
})

// ❌ Avoid: Flat keys without context
await client.admin.settings.system.update('stripe_key', {
  value: { key: 'pk_live_...' }
})
```

### 4. Add Descriptions to System Settings

Document custom settings for future reference:

```typescript
// ✅ Good: Includes description
await client.admin.settings.system.update('custom.feature.beta', {
  value: { enabled: true },
  description: 'Beta feature flag - controls access to new features'
})

// ❌ Avoid: No description
await client.admin.settings.system.update('custom.feature.beta', {
  value: { enabled: true }
})
```

### 5. Handle Errors Gracefully

Always handle potential errors, especially for user-facing operations:

```typescript
// ✅ Good: Error handling with user feedback
try {
  await client.admin.settings.app.setPasswordMinLength(length)
  showSuccess('Password requirements updated')
} catch (error) {
  if (error.message.includes('between 8 and 128')) {
    showError('Password length must be between 8 and 128 characters')
  } else {
    showError('Failed to update settings')
  }
}
```

### 6. Validate Before Updating

Check current state before making changes:

```typescript
// ✅ Good: Check current state first
const current = await client.admin.settings.app.get()

if (!current.authentication.enable_signup) {
  await client.admin.settings.app.enableSignup()
  console.log('Signup was disabled, now enabled')
} else {
  console.log('Signup already enabled')
}
```

### 7. Use Type-Safe Updates

Leverage TypeScript for compile-time validation:

```typescript
// ✅ Good: TypeScript catches invalid keys
await client.admin.settings.app.update({
  authentication: {
    enable_signup: true,
    // TypeScript error: 'invalid_key' doesn't exist
    // invalid_key: true
  }
})
```

---

## TypeScript Types

The Settings SDK is fully typed for TypeScript users.

```typescript
import type {
  AppSettings,
  AuthenticationSettings,
  FeatureSettings,
  EmailSettings,
  SecuritySettings,
  SystemSetting,
  UpdateAppSettingsRequest,
  UpdateSystemSettingRequest,
  ListSystemSettingsResponse
} from '@fluxbase/sdk'

// Type-safe settings operations
const settings: AppSettings = await client.admin.settings.app.get()

const update: UpdateAppSettingsRequest = {
  authentication: {
    enable_signup: false
  }
}

await client.admin.settings.app.update(update)
```

---

## Next Steps

- Learn about [Admin SDK](/docs/sdk/admin) for user management and authentication
- Explore [Management SDK](/docs/sdk/management) for API keys and webhooks
- Read about [Database](/docs/guides/typescript-sdk/database) operations
- Check out [Authentication](/docs/guides/authentication) for user-facing auth flows

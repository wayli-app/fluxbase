import type { FluxbaseFetch } from "./fetch";
import type {
  SystemSetting,
  UpdateSystemSettingRequest,
  ListSystemSettingsResponse,
  AppSettings,
  UpdateAppSettingsRequest,
  CustomSetting,
  CreateCustomSettingRequest,
  UpdateCustomSettingRequest,
  ListCustomSettingsResponse,
  EmailTemplate,
  EmailTemplateType,
  UpdateEmailTemplateRequest,
  ListEmailTemplatesResponse,
} from "./types";

/**
 * System Settings Manager
 *
 * Manages low-level system settings with key-value storage.
 * For application-level settings, use AppSettingsManager instead.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings.system
 *
 * // List all system settings
 * const { settings } = await settings.list()
 *
 * // Get specific setting
 * const setting = await settings.get('app.auth.enable_signup')
 *
 * // Update setting
 * await settings.update('app.auth.enable_signup', {
 *   value: { value: true },
 *   description: 'Enable user signup'
 * })
 *
 * // Delete setting
 * await settings.delete('app.auth.enable_signup')
 * ```
 */
export class SystemSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * List all system settings
   *
   * @returns Promise resolving to ListSystemSettingsResponse
   *
   * @example
   * ```typescript
   * const response = await client.admin.settings.system.list()
   * console.log(response.settings)
   * ```
   */
  async list(): Promise<ListSystemSettingsResponse> {
    const settings = await this.fetch.get<SystemSetting[]>(
      "/api/v1/admin/system/settings",
    );
    return { settings: Array.isArray(settings) ? settings : [] };
  }

  /**
   * Get a specific system setting by key
   *
   * @param key - Setting key (e.g., 'app.auth.enable_signup')
   * @returns Promise resolving to SystemSetting
   *
   * @example
   * ```typescript
   * const setting = await client.admin.settings.system.get('app.auth.enable_signup')
   * console.log(setting.value)
   * ```
   */
  async get(key: string): Promise<SystemSetting> {
    return await this.fetch.get<SystemSetting>(
      `/api/v1/admin/system/settings/${key}`,
    );
  }

  /**
   * Update or create a system setting
   *
   * @param key - Setting key
   * @param request - Update request with value and optional description
   * @returns Promise resolving to SystemSetting
   *
   * @example
   * ```typescript
   * const updated = await client.admin.settings.system.update('app.auth.enable_signup', {
   *   value: { value: true },
   *   description: 'Enable user signup'
   * })
   * ```
   */
  async update(
    key: string,
    request: UpdateSystemSettingRequest,
  ): Promise<SystemSetting> {
    return await this.fetch.put<SystemSetting>(
      `/api/v1/admin/system/settings/${key}`,
      request,
    );
  }

  /**
   * Delete a system setting
   *
   * @param key - Setting key to delete
   * @returns Promise<void>
   *
   * @example
   * ```typescript
   * await client.admin.settings.system.delete('app.auth.enable_signup')
   * ```
   */
  async delete(key: string): Promise<void> {
    await this.fetch.delete(`/api/v1/admin/system/settings/${key}`);
  }
}

/**
 * Application Settings Manager
 *
 * Manages high-level application settings with a structured API.
 * Provides type-safe access to authentication, features, email, and security settings.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings.app
 *
 * // Get all app settings
 * const appSettings = await settings.get()
 * console.log(appSettings.authentication.enable_signup)
 *
 * // Update specific settings
 * const updated = await settings.update({
 *   authentication: {
 *     enable_signup: true,
 *     password_min_length: 12
 *   }
 * })
 *
 * // Reset to defaults
 * await settings.reset()
 * ```
 */
export class AppSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Get all application settings
   *
   * Returns structured settings for authentication, features, email, and security.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * const settings = await client.admin.settings.app.get()
   *
   * console.log('Signup enabled:', settings.authentication.enable_signup)
   * console.log('Realtime enabled:', settings.features.enable_realtime)
   * console.log('Email provider:', settings.email.provider)
   * ```
   */
  async get(): Promise<AppSettings> {
    return await this.fetch.get<AppSettings>("/api/v1/admin/app/settings");
  }

  /**
   * Update application settings
   *
   * Supports partial updates - only provide the fields you want to change.
   *
   * @param request - Settings to update (partial update supported)
   * @returns Promise resolving to AppSettings - Updated settings
   *
   * @example
   * ```typescript
   * // Update authentication settings
   * const updated = await client.admin.settings.app.update({
   *   authentication: {
   *     enable_signup: true,
   *     password_min_length: 12
   *   }
   * })
   *
   * // Update multiple categories
   * await client.admin.settings.app.update({
   *   authentication: { enable_signup: false },
   *   features: { enable_realtime: true },
   *   security: { enable_global_rate_limit: true }
   * })
   * ```
   */
  async update(request: UpdateAppSettingsRequest): Promise<AppSettings> {
    return await this.fetch.put<AppSettings>(
      "/api/v1/admin/app/settings",
      request,
    );
  }

  /**
   * Reset all application settings to defaults
   *
   * This will delete all custom settings and return to default values.
   *
   * @returns Promise resolving to AppSettings - Default settings
   *
   * @example
   * ```typescript
   * const defaults = await client.admin.settings.app.reset()
   * console.log('Settings reset to defaults:', defaults)
   * ```
   */
  async reset(): Promise<AppSettings> {
    return await this.fetch.post<AppSettings>(
      "/api/v1/admin/app/settings/reset",
      {},
    );
  }

  /**
   * Enable user signup
   *
   * Convenience method to enable user registration.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.enableSignup()
   * ```
   */
  async enableSignup(): Promise<AppSettings> {
    return await this.update({
      authentication: { enable_signup: true },
    });
  }

  /**
   * Disable user signup
   *
   * Convenience method to disable user registration.
   *
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.disableSignup()
   * ```
   */
  async disableSignup(): Promise<AppSettings> {
    return await this.update({
      authentication: { enable_signup: false },
    });
  }

  /**
   * Update password minimum length
   *
   * Convenience method to set password requirements.
   *
   * @param length - Minimum password length (8-128 characters)
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setPasswordMinLength(12)
   * ```
   */
  async setPasswordMinLength(length: number): Promise<AppSettings> {
    if (length < 8 || length > 128) {
      throw new Error(
        "Password minimum length must be between 8 and 128 characters",
      );
    }

    return await this.update({
      authentication: { password_min_length: length },
    });
  }

  /**
   * Enable or disable a feature
   *
   * Convenience method to toggle feature flags.
   *
   * @param feature - Feature name ('realtime' | 'storage' | 'functions')
   * @param enabled - Whether to enable or disable the feature
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * // Enable realtime
   * await client.admin.settings.app.setFeature('realtime', true)
   *
   * // Disable storage
   * await client.admin.settings.app.setFeature('storage', false)
   * ```
   */
  async setFeature(
    feature: "realtime" | "storage" | "functions",
    enabled: boolean,
  ): Promise<AppSettings> {
    const featureKey =
      feature === "realtime"
        ? "enable_realtime"
        : feature === "storage"
          ? "enable_storage"
          : "enable_functions";

    return await this.update({
      features: { [featureKey]: enabled },
    });
  }

  /**
   * Enable or disable global rate limiting
   *
   * Convenience method to toggle global rate limiting.
   *
   * @param enabled - Whether to enable rate limiting
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setRateLimiting(true)
   * ```
   */
  async setRateLimiting(enabled: boolean): Promise<AppSettings> {
    return await this.update({
      security: { enable_global_rate_limit: enabled },
    });
  }

  /**
   * Configure SMTP email provider
   *
   * Convenience method to set up SMTP email delivery.
   *
   * @param config - SMTP configuration
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.configureSMTP({
   *   host: 'smtp.gmail.com',
   *   port: 587,
   *   username: 'your-email@gmail.com',
   *   password: 'your-app-password',
   *   use_tls: true,
   *   from_address: 'noreply@yourapp.com',
   *   from_name: 'Your App'
   * })
   * ```
   */
  async configureSMTP(config: {
    host: string;
    port: number;
    username: string;
    password: string;
    use_tls: boolean;
    from_address?: string;
    from_name?: string;
    reply_to_address?: string;
  }): Promise<AppSettings> {
    return await this.update({
      email: {
        enabled: true,
        provider: "smtp",
        from_address: config.from_address,
        from_name: config.from_name,
        reply_to_address: config.reply_to_address,
        smtp: {
          host: config.host,
          port: config.port,
          username: config.username,
          password: config.password,
          use_tls: config.use_tls,
        },
      },
    });
  }

  /**
   * Configure SendGrid email provider
   *
   * Convenience method to set up SendGrid email delivery.
   *
   * @param apiKey - SendGrid API key
   * @param options - Optional from address, name, and reply-to
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.configureSendGrid('SG.xxx', {
   *   from_address: 'noreply@yourapp.com',
   *   from_name: 'Your App'
   * })
   * ```
   */
  async configureSendGrid(
    apiKey: string,
    options?: {
      from_address?: string;
      from_name?: string;
      reply_to_address?: string;
    },
  ): Promise<AppSettings> {
    return await this.update({
      email: {
        enabled: true,
        provider: "sendgrid",
        from_address: options?.from_address,
        from_name: options?.from_name,
        reply_to_address: options?.reply_to_address,
        sendgrid: {
          api_key: apiKey,
        },
      },
    });
  }

  /**
   * Configure Mailgun email provider
   *
   * Convenience method to set up Mailgun email delivery.
   *
   * @param apiKey - Mailgun API key
   * @param domain - Mailgun domain
   * @param options - Optional EU region flag and email addresses
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.configureMailgun('key-xxx', 'mg.yourapp.com', {
   *   eu_region: false,
   *   from_address: 'noreply@yourapp.com',
   *   from_name: 'Your App'
   * })
   * ```
   */
  async configureMailgun(
    apiKey: string,
    domain: string,
    options?: {
      eu_region?: boolean;
      from_address?: string;
      from_name?: string;
      reply_to_address?: string;
    },
  ): Promise<AppSettings> {
    return await this.update({
      email: {
        enabled: true,
        provider: "mailgun",
        from_address: options?.from_address,
        from_name: options?.from_name,
        reply_to_address: options?.reply_to_address,
        mailgun: {
          api_key: apiKey,
          domain: domain,
          eu_region: options?.eu_region ?? false,
        },
      },
    });
  }

  /**
   * Configure AWS SES email provider
   *
   * Convenience method to set up AWS SES email delivery.
   *
   * @param accessKeyId - AWS access key ID
   * @param secretAccessKey - AWS secret access key
   * @param region - AWS region (e.g., 'us-east-1')
   * @param options - Optional email addresses
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.configureSES(
   *   'AKIAIOSFODNN7EXAMPLE',
   *   'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
   *   'us-east-1',
   *   {
   *     from_address: 'noreply@yourapp.com',
   *     from_name: 'Your App'
   *   }
   * )
   * ```
   */
  async configureSES(
    accessKeyId: string,
    secretAccessKey: string,
    region: string,
    options?: {
      from_address?: string;
      from_name?: string;
      reply_to_address?: string;
    },
  ): Promise<AppSettings> {
    return await this.update({
      email: {
        enabled: true,
        provider: "ses",
        from_address: options?.from_address,
        from_name: options?.from_name,
        reply_to_address: options?.reply_to_address,
        ses: {
          access_key_id: accessKeyId,
          secret_access_key: secretAccessKey,
          region: region,
        },
      },
    });
  }

  /**
   * Enable or disable email functionality
   *
   * Convenience method to toggle email system on/off.
   *
   * @param enabled - Whether to enable email
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setEmailEnabled(true)
   * ```
   */
  async setEmailEnabled(enabled: boolean): Promise<AppSettings> {
    return await this.update({
      email: { enabled },
    });
  }

  /**
   * Configure password complexity requirements
   *
   * Convenience method to set password validation rules.
   *
   * @param requirements - Password complexity requirements
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setPasswordComplexity({
   *   min_length: 12,
   *   require_uppercase: true,
   *   require_lowercase: true,
   *   require_number: true,
   *   require_special: true
   * })
   * ```
   */
  async setPasswordComplexity(requirements: {
    min_length?: number;
    require_uppercase?: boolean;
    require_lowercase?: boolean;
    require_number?: boolean;
    require_special?: boolean;
  }): Promise<AppSettings> {
    return await this.update({
      authentication: {
        password_min_length: requirements.min_length,
        password_require_uppercase: requirements.require_uppercase,
        password_require_lowercase: requirements.require_lowercase,
        password_require_number: requirements.require_number,
        password_require_special: requirements.require_special,
      },
    });
  }

  /**
   * Configure session settings
   *
   * Convenience method to set session timeout and limits.
   *
   * @param timeoutMinutes - Session timeout in minutes (0 for no timeout)
   * @param maxSessionsPerUser - Maximum concurrent sessions per user (0 for unlimited)
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * // 30 minute sessions, max 3 devices per user
   * await client.admin.settings.app.setSessionSettings(30, 3)
   * ```
   */
  async setSessionSettings(
    timeoutMinutes: number,
    maxSessionsPerUser: number,
  ): Promise<AppSettings> {
    return await this.update({
      authentication: {
        session_timeout_minutes: timeoutMinutes,
        max_sessions_per_user: maxSessionsPerUser,
      },
    });
  }

  /**
   * Enable or disable email verification requirement
   *
   * Convenience method to require email verification for new signups.
   *
   * @param required - Whether to require email verification
   * @returns Promise resolving to AppSettings
   *
   * @example
   * ```typescript
   * await client.admin.settings.app.setEmailVerificationRequired(true)
   * ```
   */
  async setEmailVerificationRequired(required: boolean): Promise<AppSettings> {
    return await this.update({
      authentication: { require_email_verification: required },
    });
  }
}

/**
 * Custom Settings Manager
 *
 * Manages custom admin-created settings with flexible key-value storage.
 * Unlike system settings, custom settings allow admins to create arbitrary configuration entries
 * with role-based editing permissions.
 *
 * @example
 * ```typescript
 * const custom = client.admin.settings.custom
 *
 * // Create a custom setting
 * const setting = await custom.create({
 *   key: 'feature.dark_mode',
 *   value: { enabled: true, theme: 'dark' },
 *   value_type: 'json',
 *   description: 'Dark mode configuration',
 *   editable_by: ['dashboard_admin', 'admin']
 * })
 *
 * // List all custom settings
 * const { settings } = await custom.list()
 *
 * // Get specific setting
 * const darkMode = await custom.get('feature.dark_mode')
 *
 * // Update setting
 * await custom.update('feature.dark_mode', {
 *   value: { enabled: false, theme: 'light' }
 * })
 *
 * // Delete setting
 * await custom.delete('feature.dark_mode')
 * ```
 */
export class CustomSettingsManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * Create a new custom setting
   *
   * @param request - Custom setting creation request
   * @returns Promise resolving to CustomSetting
   *
   * @example
   * ```typescript
   * const setting = await client.admin.settings.custom.create({
   *   key: 'api.quotas',
   *   value: { free: 1000, pro: 10000, enterprise: 100000 },
   *   value_type: 'json',
   *   description: 'API request quotas by tier',
   *   metadata: { category: 'billing' }
   * })
   * ```
   */
  async create(request: CreateCustomSettingRequest): Promise<CustomSetting> {
    return await this.fetch.post<CustomSetting>(
      "/api/v1/admin/settings/custom",
      request,
    );
  }

  /**
   * List all custom settings
   *
   * @returns Promise resolving to ListCustomSettingsResponse
   *
   * @example
   * ```typescript
   * const response = await client.admin.settings.custom.list()
   * console.log(response.settings)
   * ```
   */
  async list(): Promise<ListCustomSettingsResponse> {
    const settings = await this.fetch.get<CustomSetting[]>(
      "/api/v1/admin/settings/custom",
    );
    return { settings: Array.isArray(settings) ? settings : [] };
  }

  /**
   * Get a specific custom setting by key
   *
   * @param key - Setting key (e.g., 'feature.dark_mode')
   * @returns Promise resolving to CustomSetting
   *
   * @example
   * ```typescript
   * const setting = await client.admin.settings.custom.get('feature.dark_mode')
   * console.log(setting.value)
   * ```
   */
  async get(key: string): Promise<CustomSetting> {
    return await this.fetch.get<CustomSetting>(
      `/api/v1/admin/settings/custom/${key}`,
    );
  }

  /**
   * Update an existing custom setting
   *
   * @param key - Setting key
   * @param request - Update request with new values
   * @returns Promise resolving to CustomSetting
   *
   * @example
   * ```typescript
   * const updated = await client.admin.settings.custom.update('feature.dark_mode', {
   *   value: { enabled: false },
   *   description: 'Updated description'
   * })
   * ```
   */
  async update(
    key: string,
    request: UpdateCustomSettingRequest,
  ): Promise<CustomSetting> {
    return await this.fetch.put<CustomSetting>(
      `/api/v1/admin/settings/custom/${key}`,
      request,
    );
  }

  /**
   * Delete a custom setting
   *
   * @param key - Setting key to delete
   * @returns Promise<void>
   *
   * @example
   * ```typescript
   * await client.admin.settings.custom.delete('feature.dark_mode')
   * ```
   */
  async delete(key: string): Promise<void> {
    await this.fetch.delete(`/api/v1/admin/settings/custom/${key}`);
  }
}

/**
 * Email Template Manager
 *
 * Manages email templates for authentication and user communication.
 * Supports customizing templates for magic links, email verification, password resets, and user invitations.
 *
 * @example
 * ```typescript
 * const templates = client.admin.emailTemplates
 *
 * // List all templates
 * const { templates: allTemplates } = await templates.list()
 *
 * // Get specific template
 * const magicLink = await templates.get('magic_link')
 *
 * // Update template
 * await templates.update('magic_link', {
 *   subject: 'Sign in to ' + '{{.AppName}}',
 *   html_body: '<html>Custom template with ' + '{{.MagicLink}}' + '</html>',
 *   text_body: 'Click here: ' + '{{.MagicLink}}'
 * })
 *
 * // Test template (sends to specified email)
 * await templates.test('magic_link', 'test@example.com')
 *
 * // Reset to default
 * await templates.reset('magic_link')
 * ```
 */
export class EmailTemplateManager {
  constructor(private fetch: FluxbaseFetch) {}

  /**
   * List all email templates
   *
   * @returns Promise resolving to ListEmailTemplatesResponse
   *
   * @example
   * ```typescript
   * const response = await client.admin.emailTemplates.list()
   * console.log(response.templates)
   * ```
   */
  async list(): Promise<ListEmailTemplatesResponse> {
    const templates = await this.fetch.get<EmailTemplate[]>(
      "/api/v1/admin/email/templates",
    );
    return { templates: Array.isArray(templates) ? templates : [] };
  }

  /**
   * Get a specific email template by type
   *
   * @param type - Template type (magic_link | verify_email | reset_password | invite_user)
   * @returns Promise resolving to EmailTemplate
   *
   * @example
   * ```typescript
   * const template = await client.admin.emailTemplates.get('magic_link')
   * console.log(template.subject)
   * console.log(template.html_body)
   * ```
   */
  async get(type: EmailTemplateType): Promise<EmailTemplate> {
    return await this.fetch.get<EmailTemplate>(
      `/api/v1/admin/email/templates/${type}`,
    );
  }

  /**
   * Update an email template
   *
   * Available template variables:
   * - magic_link: `{{.MagicLink}}`, `{{.AppName}}`, `{{.ExpiryMinutes}}`
   * - verify_email: `{{.VerificationLink}}`, `{{.AppName}}`
   * - reset_password: `{{.ResetLink}}`, `{{.AppName}}`, `{{.ExpiryMinutes}}`
   * - invite_user: `{{.InviteLink}}`, `{{.AppName}}`, `{{.InviterName}}`
   *
   * @param type - Template type to update
   * @param request - Update request with subject, html_body, and optional text_body
   * @returns Promise resolving to EmailTemplate
   *
   * @example
   * ```typescript
   * const updated = await client.admin.emailTemplates.update('magic_link', {
   *   subject: 'Your Magic Link - Sign in to ' + '{{.AppName}}',
   *   html_body: '<html><body><h1>Welcome!</h1><a href="' + '{{.MagicLink}}' + '">Sign In</a></body></html>',
   *   text_body: 'Click here to sign in: ' + '{{.MagicLink}}'
   * })
   * ```
   */
  async update(
    type: EmailTemplateType,
    request: UpdateEmailTemplateRequest,
  ): Promise<EmailTemplate> {
    return await this.fetch.put<EmailTemplate>(
      `/api/v1/admin/email/templates/${type}`,
      request,
    );
  }

  /**
   * Reset an email template to default
   *
   * Removes any customizations and restores the template to its original state.
   *
   * @param type - Template type to reset
   * @returns Promise resolving to EmailTemplate - The default template
   *
   * @example
   * ```typescript
   * const defaultTemplate = await client.admin.emailTemplates.reset('magic_link')
   * ```
   */
  async reset(type: EmailTemplateType): Promise<EmailTemplate> {
    return await this.fetch.post<EmailTemplate>(
      `/api/v1/admin/email/templates/${type}/reset`,
      {},
    );
  }

  /**
   * Send a test email using the template
   *
   * Useful for previewing template changes before deploying to production.
   *
   * @param type - Template type to test
   * @param recipientEmail - Email address to send test to
   * @returns Promise<void>
   *
   * @example
   * ```typescript
   * await client.admin.emailTemplates.test('magic_link', 'test@example.com')
   * ```
   */
  async test(type: EmailTemplateType, recipientEmail: string): Promise<void> {
    await this.fetch.post(`/api/v1/admin/email/templates/${type}/test`, {
      recipient_email: recipientEmail,
    });
  }
}

/**
 * Settings Manager
 *
 * Provides access to system-level, application-level, and custom settings.
 *
 * @example
 * ```typescript
 * const settings = client.admin.settings
 *
 * // Access system settings
 * const systemSettings = await settings.system.list()
 *
 * // Access app settings
 * const appSettings = await settings.app.get()
 *
 * // Access custom settings
 * const customSettings = await settings.custom.list()
 * ```
 */
export class FluxbaseSettings {
  public system: SystemSettingsManager;
  public app: AppSettingsManager;
  public custom: CustomSettingsManager;

  constructor(fetch: FluxbaseFetch) {
    this.system = new SystemSettingsManager(fetch);
    this.app = new AppSettingsManager(fetch);
    this.custom = new CustomSettingsManager(fetch);
  }
}

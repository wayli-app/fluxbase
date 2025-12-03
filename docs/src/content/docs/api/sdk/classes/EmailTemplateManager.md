---
editUrl: false
next: false
prev: false
title: "EmailTemplateManager"
---

Email Template Manager

Manages email templates for authentication and user communication.
Supports customizing templates for magic links, email verification, password resets, and user invitations.

## Example

```typescript
const templates = client.admin.emailTemplates

// List all templates
const { templates: allTemplates } = await templates.list()

// Get specific template
const magicLink = await templates.get('magic_link')

// Update template
await templates.update('magic_link', {
  subject: 'Sign in to ' + '{{.AppName}}',
  html_body: '<html>Custom template with ' + '{{.MagicLink}}' + '</html>',
  text_body: 'Click here: ' + '{{.MagicLink}}'
})

// Test template (sends to specified email)
await templates.test('magic_link', 'test@example.com')

// Reset to default
await templates.reset('magic_link')
```

## Constructors

### new EmailTemplateManager()

> **new EmailTemplateManager**(`fetch`): [`EmailTemplateManager`](/api/sdk/classes/emailtemplatemanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`EmailTemplateManager`](/api/sdk/classes/emailtemplatemanager/)

## Methods

### get()

> **get**(`type`): `Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Get a specific email template by type

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `type` | [`EmailTemplateType`](/api/sdk/type-aliases/emailtemplatetype/) | Template type (magic_link | verify_email | reset_password | invite_user) |

#### Returns

`Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Promise resolving to EmailTemplate

#### Example

```typescript
const template = await client.admin.emailTemplates.get('magic_link')
console.log(template.subject)
console.log(template.html_body)
```

***

### list()

> **list**(): `Promise`\<[`ListEmailTemplatesResponse`](/api/sdk/interfaces/listemailtemplatesresponse/)\>

List all email templates

#### Returns

`Promise`\<[`ListEmailTemplatesResponse`](/api/sdk/interfaces/listemailtemplatesresponse/)\>

Promise resolving to ListEmailTemplatesResponse

#### Example

```typescript
const response = await client.admin.emailTemplates.list()
console.log(response.templates)
```

***

### reset()

> **reset**(`type`): `Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Reset an email template to default

Removes any customizations and restores the template to its original state.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `type` | [`EmailTemplateType`](/api/sdk/type-aliases/emailtemplatetype/) | Template type to reset |

#### Returns

`Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Promise resolving to EmailTemplate - The default template

#### Example

```typescript
const defaultTemplate = await client.admin.emailTemplates.reset('magic_link')
```

***

### test()

> **test**(`type`, `recipientEmail`): `Promise`\<`void`\>

Send a test email using the template

Useful for previewing template changes before deploying to production.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `type` | [`EmailTemplateType`](/api/sdk/type-aliases/emailtemplatetype/) | Template type to test |
| `recipientEmail` | `string` | Email address to send test to |

#### Returns

`Promise`\<`void`\>

Promise<void>

#### Example

```typescript
await client.admin.emailTemplates.test('magic_link', 'test@example.com')
```

***

### update()

> **update**(`type`, `request`): `Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Update an email template

Available template variables:
- magic_link: `{{.MagicLink}}`, `{{.AppName}}`, `{{.ExpiryMinutes}}`
- verify_email: `{{.VerificationLink}}`, `{{.AppName}}`
- reset_password: `{{.ResetLink}}`, `{{.AppName}}`, `{{.ExpiryMinutes}}`
- invite_user: `{{.InviteLink}}`, `{{.AppName}}`, `{{.InviterName}}`

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `type` | [`EmailTemplateType`](/api/sdk/type-aliases/emailtemplatetype/) | Template type to update |
| `request` | [`UpdateEmailTemplateRequest`](/api/sdk/interfaces/updateemailtemplaterequest/) | Update request with subject, html_body, and optional text_body |

#### Returns

`Promise`\<[`EmailTemplate`](/api/sdk/interfaces/emailtemplate/)\>

Promise resolving to EmailTemplate

#### Example

```typescript
const updated = await client.admin.emailTemplates.update('magic_link', {
  subject: 'Your Magic Link - Sign in to ' + '{{.AppName}}',
  html_body: '<html><body><h1>Welcome!</h1><a href="' + '{{.MagicLink}}' + '">Sign In</a></body></html>',
  text_body: 'Click here to sign in: ' + '{{.MagicLink}}'
})
```

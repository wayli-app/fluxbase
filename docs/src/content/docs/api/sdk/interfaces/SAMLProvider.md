---
editUrl: false
next: false
prev: false
title: "SAMLProvider"
---

SAML Identity Provider configuration

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `enabled` | `boolean` | Whether the provider is enabled |
| `entity_id` | `string` | Provider's entity ID (used for SP metadata) |
| `id` | `string` | Unique provider identifier (slug name) |
| `name` | `string` | Display name of the provider |
| `slo_url?` | `string` | Single Logout endpoint URL (optional) |
| `sso_url` | `string` | SSO endpoint URL |

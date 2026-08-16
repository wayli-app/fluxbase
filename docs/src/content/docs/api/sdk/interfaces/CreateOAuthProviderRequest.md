---
editUrl: false
next: false
prev: false
title: "CreateOAuthProviderRequest"
---

Request to create a new OAuth provider

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="authorization_url"></a> `authorization_url?` | `string` | - |
| <a id="client_id"></a> `client_id` | `string` | - |
| <a id="client_secret"></a> `client_secret` | `string` | - |
| <a id="display_name"></a> `display_name` | `string` | - |
| <a id="enabled"></a> `enabled` | `boolean` | - |
| <a id="is_custom"></a> `is_custom` | `boolean` | - |
| <a id="provider_name"></a> `provider_name` | `string` | - |
| <a id="redirect_url"></a> ~~`redirect_url?`~~ | `string` | :::caution[Deprecated] Use redirect_urls instead; treated as a single-element list ::: |
| <a id="redirect_urls"></a> `redirect_urls?` | `string`[] | - |
| <a id="scopes"></a> `scopes` | `string`[] | - |
| <a id="token_url"></a> `token_url?` | `string` | - |
| <a id="user_info_url"></a> `user_info_url?` | `string` | - |

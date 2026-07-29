---
editUrl: false
next: false
prev: false
title: "ToolIntegration"
---

Tool integration configuration. Mirrors the server-side
`tool_integrations` table shape. Secrets in `config` (e.g., api_key)
are masked to "***masked***" on read; updates that pass the mask value
back preserve the existing encrypted value.

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="config"></a> `config` | `Record`\<`string`, `string`\> | Provider-specific config. api_key is masked on read. |
| <a id="created_at"></a> `created_at` | `string` | - |
| <a id="created_by"></a> `created_by?` | `string` | - |
| <a id="enabled"></a> `enabled` | `boolean` | - |
| <a id="from_config"></a> `from_config?` | `boolean` | True when configured via env/YAML (read-only in the UI). |
| <a id="id"></a> `id` | `string` | - |
| <a id="integration_type"></a> `integration_type` | [`IntegrationType`](/api/sdk/type-aliases/integrationtype/) | - |
| <a id="is_default"></a> `is_default` | `boolean` | - |
| <a id="last_test_error"></a> `last_test_error?` | `string` | - |
| <a id="last_test_status"></a> `last_test_status?` | `"failed"` \| `"ok"` | - |
| <a id="last_tested_at"></a> `last_tested_at?` | `string` | Result of the most recent test-connection call, if any. |
| <a id="name"></a> `name` | `string` | - |
| <a id="provider"></a> `provider` | [`IntegrationProvider`](/api/sdk/type-aliases/integrationprovider/) | - |
| <a id="read_only"></a> `read_only?` | `boolean` | - |
| <a id="updated_at"></a> `updated_at` | `string` | - |

---
editUrl: false
next: false
prev: false
title: "AIUsageStats"
---

AI token usage statistics

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="cached_tokens"></a> `cached_tokens?` | `number` | Subset of prompt_tokens served from the provider's prompt cache (OpenAI automatic prefix caching, Anthropic prompt caching). 0 when caching didn't fire or the provider doesn't report it. (Ask 4) |
| <a id="completion_tokens"></a> `completion_tokens` | `number` | - |
| <a id="prompt_tokens"></a> `prompt_tokens` | `number` | - |
| <a id="total_tokens"></a> `total_tokens?` | `number` | - |

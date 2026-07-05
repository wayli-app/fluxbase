---
editUrl: false
next: false
prev: false
title: "AIDailyQuotaSnapshot"
---

Per-user daily quota snapshot returned in the done event (Ask 2) and by
fluxbase.ai.getUsage(). Counts are best-effort in-memory; they reset on
server restart and are per-instance in multi-replica deployments.

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="requests"></a> `requests` | [`AIQuota`](/api/sdk/interfaces/aiquota/) | - |
| <a id="resets_at"></a> `resets_at?` | `string` | RFC3339 timestamp of when the counters roll over to zero. |
| <a id="tokens"></a> `tokens` | [`AIQuota`](/api/sdk/interfaces/aiquota/) | - |

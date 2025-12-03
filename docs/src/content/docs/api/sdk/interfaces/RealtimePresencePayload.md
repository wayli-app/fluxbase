---
editUrl: false
next: false
prev: false
title: "RealtimePresencePayload"
---

Realtime presence payload structure

## Properties

| Property | Type |
| ------ | ------ |
| `currentPresences?` | `Record`\<`string`, [`PresenceState`](/api/sdk/interfaces/presencestate/)[]\> |
| `event` | `"sync"` \| `"join"` \| `"leave"` |
| `key?` | `string` |
| `leftPresences?` | [`PresenceState`](/api/sdk/interfaces/presencestate/)[] |
| `newPresences?` | [`PresenceState`](/api/sdk/interfaces/presencestate/)[] |

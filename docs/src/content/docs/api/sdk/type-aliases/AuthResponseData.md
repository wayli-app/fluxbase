---
editUrl: false
next: false
prev: false
title: "AuthResponseData"
---

> **AuthResponseData**: `object`

Auth response with user and session (Supabase-compatible)

## Type declaration

| Name | Type |
| ------ | ------ |
| `session` | [`AuthSession`](/api/sdk/interfaces/authsession/) \| `null` |
| `user` | [`User`](/api/sdk/interfaces/user/) |
| `weakPassword`? | [`WeakPassword`](/api/sdk/interfaces/weakpassword/) |

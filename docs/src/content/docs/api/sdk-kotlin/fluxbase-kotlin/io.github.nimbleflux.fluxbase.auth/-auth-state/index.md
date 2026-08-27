---
title: "AuthState"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[AuthState](./)

# AuthState

[jvm]\
data class [AuthState](./)(val event: [AuthChangeEvent](../-auth-change-event/), val session: [AuthSession](../-auth-session/)?)

A single auth state change event: the [event](event/) type plus the current session (null after SIGNED_OUT).

## Constructors

| | |
|---|---|
| [AuthState](-auth-state/) | [jvm]<br>constructor(event: [AuthChangeEvent](../-auth-change-event/), session: [AuthSession](../-auth-session/)?) |

## Properties

| Name | Summary |
|---|---|
| [event](event/) | [jvm]<br>val [event](event/): [AuthChangeEvent](../-auth-change-event/) |
| [session](session/) | [jvm]<br>val [session](session/): [AuthSession](../-auth-session/)? |

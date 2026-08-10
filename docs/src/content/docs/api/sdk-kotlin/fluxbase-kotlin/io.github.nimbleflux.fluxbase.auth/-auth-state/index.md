//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[AuthState](index.md)

# AuthState

[jvm]\
data class [AuthState](index.md)(val event: [AuthChangeEvent](../-auth-change-event/index.md), val session: [AuthSession](../-auth-session/index.md)?)

A single auth state change event: the [event](event.md) type plus the current session (null after SIGNED_OUT).

## Constructors

| | |
|---|---|
| [AuthState](-auth-state.md) | [jvm]<br>constructor(event: [AuthChangeEvent](../-auth-change-event/index.md), session: [AuthSession](../-auth-session/index.md)?) |

## Properties

| Name | Summary |
|---|---|
| [event](event.md) | [jvm]<br>val [event](event.md): [AuthChangeEvent](../-auth-change-event/index.md) |
| [session](session.md) | [jvm]<br>val [session](session.md): [AuthSession](../-auth-session/index.md)? |

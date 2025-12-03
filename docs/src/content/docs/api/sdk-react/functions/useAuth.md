---
editUrl: false
next: false
prev: false
title: "useAuth"
---

> **useAuth**(): `object`

Combined auth hook with all auth state and methods

## Returns

`object`

| Name | Type | Default value |
| ------ | ------ | ------ |
| `isAuthenticated` | `boolean` | !!session |
| `isLoading` | `boolean` | - |
| `isSigningIn` | `boolean` | signIn.isPending |
| `isSigningOut` | `boolean` | signOut.isPending |
| `isSigningUp` | `boolean` | signUp.isPending |
| `isUpdating` | `boolean` | updateUser.isPending |
| `session` | `undefined` \| `null` \| [`AuthSession`](/api/sdk-react/interfaces/authsession/) | - |
| `signIn` | `UseMutateAsyncFunction`\<`FluxbaseResponse`\<`AuthResponseData` \| `SignInWith2FAResponse`\>, `Error`, [`SignInCredentials`](/api/sdk-react/interfaces/signincredentials/), `unknown`\> | signIn.mutateAsync |
| `signOut` | `UseMutateAsyncFunction`\<`void`, `Error`, `void`, `unknown`\> | signOut.mutateAsync |
| `signUp` | `UseMutateAsyncFunction`\<`FluxbaseAuthResponse`, `Error`, [`SignUpCredentials`](/api/sdk-react/interfaces/signupcredentials/), `unknown`\> | signUp.mutateAsync |
| `updateUser` | `UseMutateAsyncFunction`\<`UserResponse`, `Error`, `Partial`\<`Pick`\<[`User`](/api/sdk-react/interfaces/user/), `"email"` \| `"metadata"`\>\>, `unknown`\> | updateUser.mutateAsync |
| `user` | `undefined` \| `null` \| [`User`](/api/sdk-react/interfaces/user/) | - |

---
editUrl: false
next: false
prev: false
title: "useSignInWithSAML"
---

> **useSignInWithSAML**(): `UseMutationResult`\<`DataResponse`\<`object`\>, `Error`, `object`, `unknown`\>

Hook to initiate SAML login (redirects to IdP)

This hook returns a mutation that when called, redirects the user to the
SAML Identity Provider for authentication.

## Returns

`UseMutationResult`\<`DataResponse`\<`object`\>, `Error`, `object`, `unknown`\>

## Example

```tsx
function SAMLLoginButton() {
  const signInWithSAML = useSignInWithSAML()

  return (
    <button
      onClick={() => signInWithSAML.mutate({ provider: 'okta' })}
      disabled={signInWithSAML.isPending}
    >
      {signInWithSAML.isPending ? 'Redirecting...' : 'Sign in with Okta'}
    </button>
  )
}
```

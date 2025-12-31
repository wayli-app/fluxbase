---
editUrl: false
next: false
prev: false
title: "useHandleSAMLCallback"
---

> **useHandleSAMLCallback**(): `UseMutationResult`\<`FluxbaseAuthResponse`, `Error`, `object`, `unknown`\>

Hook to handle SAML callback after IdP authentication

Use this in your SAML callback page to complete the authentication flow.

## Returns

`UseMutationResult`\<`FluxbaseAuthResponse`, `Error`, `object`, `unknown`\>

## Example

```tsx
function SAMLCallbackPage() {
  const handleCallback = useHandleSAMLCallback()
  const navigate = useNavigate()

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const samlResponse = params.get('SAMLResponse')

    if (samlResponse) {
      handleCallback.mutate(
        { samlResponse },
        {
          onSuccess: () => navigate('/dashboard'),
          onError: (error) => console.error('SAML login failed:', error)
        }
      )
    }
  }, [])

  if (handleCallback.isPending) {
    return <div>Completing sign in...</div>
  }

  if (handleCallback.isError) {
    return <div>Authentication failed: {handleCallback.error.message}</div>
  }

  return null
}
```

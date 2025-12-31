---
editUrl: false
next: false
prev: false
title: "useGetSAMLLoginUrl"
---

> **useGetSAMLLoginUrl**(): `UseMutationResult`\<`DataResponse`\<[`SAMLLoginResponse`](/api/sdk-react/interfaces/samlloginresponse/)\>, `Error`, `object`, `unknown`\>

Hook to get SAML login URL for a provider

This hook returns a function to get the login URL for a specific provider.
Use this when you need more control over the redirect behavior.

## Returns

`UseMutationResult`\<`DataResponse`\<[`SAMLLoginResponse`](/api/sdk-react/interfaces/samlloginresponse/)\>, `Error`, `object`, `unknown`\>

## Example

```tsx
function SAMLLoginButton({ provider }: { provider: string }) {
  const getSAMLLoginUrl = useGetSAMLLoginUrl()

  const handleClick = async () => {
    const { data, error } = await getSAMLLoginUrl.mutateAsync({
      provider,
      options: { redirectUrl: window.location.href }
    })
    if (!error) {
      window.location.href = data.url
    }
  }

  return <button onClick={handleClick}>Login with {provider}</button>
}
```

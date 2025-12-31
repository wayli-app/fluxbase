---
editUrl: false
next: false
prev: false
title: "useSAMLProviders"
---

> **useSAMLProviders**(): `UseQueryResult`\<[`SAMLProvider`](/api/sdk-react/interfaces/samlprovider/)[], `Error`\>

Hook to get available SAML SSO providers

## Returns

`UseQueryResult`\<[`SAMLProvider`](/api/sdk-react/interfaces/samlprovider/)[], `Error`\>

## Example

```tsx
function SAMLProviderList() {
  const { data: providers, isLoading } = useSAMLProviders()

  if (isLoading) return <div>Loading...</div>

  return (
    <div>
      {providers?.map(provider => (
        <button key={provider.id} onClick={() => signInWithSAML(provider.name)}>
          Sign in with {provider.name}
        </button>
      ))}
    </div>
  )
}
```

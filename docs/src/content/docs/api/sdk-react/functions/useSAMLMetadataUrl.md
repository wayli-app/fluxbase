---
editUrl: false
next: false
prev: false
title: "useSAMLMetadataUrl"
---

> **useSAMLMetadataUrl**(): (`provider`) => `string`

Hook to get SAML Service Provider metadata URL

Returns a function that generates the SP metadata URL for a given provider.
Use this URL when configuring your SAML IdP.

## Returns

`Function`

### Parameters

| Parameter | Type |
| ------ | ------ |
| `provider` | `string` |

### Returns

`string`

## Example

```tsx
function SAMLSetupInfo({ provider }: { provider: string }) {
  const getSAMLMetadataUrl = useSAMLMetadataUrl()
  const metadataUrl = getSAMLMetadataUrl(provider)

  return (
    <div>
      <p>SP Metadata URL:</p>
      <code>{metadataUrl}</code>
    </div>
  )
}
```

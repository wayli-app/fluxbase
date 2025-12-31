---
editUrl: false
next: false
prev: false
title: "useCaptchaConfig"
---

> **useCaptchaConfig**(): `UseQueryResult`\<[`CaptchaConfig`](/api/sdk-react/interfaces/captchaconfig/), `Error`\>

Hook to get the CAPTCHA configuration from the server
Use this to determine which CAPTCHA provider to load

## Returns

`UseQueryResult`\<[`CaptchaConfig`](/api/sdk-react/interfaces/captchaconfig/), `Error`\>

## Example

```tsx
function AuthPage() {
  const { data: captchaConfig, isLoading } = useCaptchaConfig();

  if (isLoading) return <Loading />;

  return captchaConfig?.enabled ? (
    <CaptchaWidget provider={captchaConfig.provider} siteKey={captchaConfig.site_key} />
  ) : null;
}
```

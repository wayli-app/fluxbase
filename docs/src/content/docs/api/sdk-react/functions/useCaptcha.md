---
editUrl: false
next: false
prev: false
title: "useCaptcha"
---

> **useCaptcha**(`provider`?): [`CaptchaState`](/api/sdk-react/interfaces/captchastate/)

Hook to manage CAPTCHA widget state

This hook provides a standardized interface for managing CAPTCHA tokens
across different providers (hCaptcha, reCAPTCHA v3, Turnstile, Cap).

Supported providers:
- hcaptcha: Privacy-focused visual challenge
- recaptcha_v3: Google's invisible risk-based CAPTCHA
- turnstile: Cloudflare's invisible CAPTCHA
- cap: Self-hosted proof-of-work CAPTCHA (https://capjs.js.org/)

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider`? | [`CaptchaProvider`](/api/sdk-react/type-aliases/captchaprovider/) | The CAPTCHA provider type |

## Returns

[`CaptchaState`](/api/sdk-react/interfaces/captchastate/)

CAPTCHA state and callbacks

## Examples

```tsx
function LoginForm() {
  const captcha = useCaptcha('hcaptcha');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    // Get CAPTCHA token
    const captchaToken = captcha.token || await captcha.execute();

    // Sign in with CAPTCHA token
    await signIn({
      email,
      password,
      captchaToken
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      <input name="email" />
      <input name="password" type="password" />

      <HCaptcha
        sitekey={siteKey}
        onVerify={captcha.onVerify}
        onExpire={captcha.onExpire}
        onError={captcha.onError}
      />

      <button type="submit" disabled={!captcha.isReady}>
        Sign In
      </button>
    </form>
  );
}
```

```tsx
function LoginForm() {
  const { data: config } = useCaptchaConfig();
  const captcha = useCaptcha(config?.provider);

  // For Cap, load the widget from cap_server_url
  // <script src={`${config.cap_server_url}/widget.js`} />
  // <cap-widget data-cap-url={config.cap_server_url} />
}
```

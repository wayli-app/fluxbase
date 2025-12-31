---
editUrl: false
next: false
prev: false
title: "CaptchaState"
---

CAPTCHA widget state for managing token generation

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `error` | `null` \| `Error` | Any error that occurred |
| `execute` | () => `Promise`\<`string`\> | Execute/trigger the CAPTCHA (for invisible CAPTCHA like reCAPTCHA v3) |
| `isLoading` | `boolean` | Whether a token is being generated |
| `isReady` | `boolean` | Whether the CAPTCHA widget is ready |
| `onError` | (`error`: `Error`) => `void` | Callback to be called when CAPTCHA errors |
| `onExpire` | () => `void` | Callback to be called when CAPTCHA expires |
| `onVerify` | (`token`: `string`) => `void` | Callback to be called when CAPTCHA is verified |
| `reset` | () => `void` | Reset the CAPTCHA widget |
| `token` | `null` \| `string` | Current CAPTCHA token (null until solved) |

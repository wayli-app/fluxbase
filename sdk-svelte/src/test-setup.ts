/**
 * Vitest setup for the Svelte SDK.
 *
 * Provides a minimal DOM-compatible environment so tests that exercise
 * Svelte components / stores behave like a browser.
 */

// jsdom (set in vitest.config.ts) supplies `window`, `document`, etc.
// Nothing else is needed for the unit tests in this package.
export {};

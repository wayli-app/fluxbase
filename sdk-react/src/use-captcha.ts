/**
 * CAPTCHA hooks for Fluxbase SDK
 *
 * Provides hooks to:
 * - Fetch CAPTCHA configuration from the server
 * - Manage CAPTCHA widget state
 */

import { useQuery } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { CaptchaConfig, CaptchaProvider } from "@fluxbase/sdk";
import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Hook to get the CAPTCHA configuration from the server
 * Use this to determine which CAPTCHA provider to load
 *
 * @example
 * ```tsx
 * function AuthPage() {
 *   const { data: captchaConfig, isLoading } = useCaptchaConfig();
 *
 *   if (isLoading) return <Loading />;
 *
 *   return captchaConfig?.enabled ? (
 *     <CaptchaWidget provider={captchaConfig.provider} siteKey={captchaConfig.site_key} />
 *   ) : null;
 * }
 * ```
 */
export function useCaptchaConfig() {
  const client = useFluxbaseClient();

  return useQuery<CaptchaConfig>({
    queryKey: ["fluxbase", "auth", "captcha", "config"],
    queryFn: async () => {
      const { data, error } = await client.auth.getCaptchaConfig();
      if (error) {
        throw error;
      }
      return data!;
    },
    staleTime: 1000 * 60 * 60, // Cache for 1 hour (config rarely changes)
    gcTime: 1000 * 60 * 60 * 24, // Keep in cache for 24 hours
  });
}

/**
 * CAPTCHA widget state for managing token generation
 */
export interface CaptchaState {
  /** Current CAPTCHA token (null until solved) */
  token: string | null;
  /** Whether the CAPTCHA widget is ready */
  isReady: boolean;
  /** Whether a token is being generated */
  isLoading: boolean;
  /** Any error that occurred */
  error: Error | null;
  /** Reset the CAPTCHA widget */
  reset: () => void;
  /** Execute/trigger the CAPTCHA (for invisible CAPTCHA like reCAPTCHA v3) */
  execute: () => Promise<string>;
  /** Callback to be called when CAPTCHA is verified */
  onVerify: (token: string) => void;
  /** Callback to be called when CAPTCHA expires */
  onExpire: () => void;
  /** Callback to be called when CAPTCHA errors */
  onError: (error: Error) => void;
}

/**
 * Hook to manage CAPTCHA widget state
 *
 * This hook provides a standardized interface for managing CAPTCHA tokens
 * across different providers (hCaptcha, reCAPTCHA v3, Turnstile, Cap).
 *
 * Supported providers:
 * - hcaptcha: Privacy-focused visual challenge
 * - recaptcha_v3: Google's invisible risk-based CAPTCHA
 * - turnstile: Cloudflare's invisible CAPTCHA
 * - cap: Self-hosted proof-of-work CAPTCHA (https://capjs.js.org/)
 *
 * @param provider - The CAPTCHA provider type
 * @returns CAPTCHA state and callbacks
 *
 * @example
 * ```tsx
 * function LoginForm() {
 *   const captcha = useCaptcha('hcaptcha');
 *
 *   const handleSubmit = async (e: FormEvent) => {
 *     e.preventDefault();
 *
 *     // Get CAPTCHA token
 *     const captchaToken = captcha.token || await captcha.execute();
 *
 *     // Sign in with CAPTCHA token
 *     await signIn({
 *       email,
 *       password,
 *       captchaToken
 *     });
 *   };
 *
 *   return (
 *     <form onSubmit={handleSubmit}>
 *       <input name="email" />
 *       <input name="password" type="password" />
 *
 *       <HCaptcha
 *         sitekey={siteKey}
 *         onVerify={captcha.onVerify}
 *         onExpire={captcha.onExpire}
 *         onError={captcha.onError}
 *       />
 *
 *       <button type="submit" disabled={!captcha.isReady}>
 *         Sign In
 *       </button>
 *     </form>
 *   );
 * }
 * ```
 *
 * @example Cap provider
 * ```tsx
 * function LoginForm() {
 *   const { data: config } = useCaptchaConfig();
 *   const captcha = useCaptcha(config?.provider);
 *
 *   // For Cap, load the widget from cap_server_url
 *   // <script src={`${config.cap_server_url}/widget.js`} />
 *   // <cap-widget data-cap-url={config.cap_server_url} />
 * }
 * ```
 */
export function useCaptcha(provider?: CaptchaProvider): CaptchaState {
  const [token, setToken] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Promise resolver for execute() method
  const executeResolverRef = useRef<((token: string) => void) | null>(null);
  const executeRejecterRef = useRef<((error: Error) => void) | null>(null);

  // Callback when CAPTCHA is verified
  const onVerify = useCallback((newToken: string) => {
    setToken(newToken);
    setIsLoading(false);
    setError(null);
    setIsReady(true);

    // Resolve the execute() promise if waiting
    if (executeResolverRef.current) {
      executeResolverRef.current(newToken);
      executeResolverRef.current = null;
      executeRejecterRef.current = null;
    }
  }, []);

  // Callback when CAPTCHA expires
  const onExpire = useCallback(() => {
    setToken(null);
    setIsReady(true);
  }, []);

  // Callback when CAPTCHA errors
  const onError = useCallback((err: Error) => {
    setError(err);
    setIsLoading(false);
    setToken(null);

    // Reject the execute() promise if waiting
    if (executeRejecterRef.current) {
      executeRejecterRef.current(err);
      executeResolverRef.current = null;
      executeRejecterRef.current = null;
    }
  }, []);

  // Reset the CAPTCHA
  const reset = useCallback(() => {
    setToken(null);
    setError(null);
    setIsLoading(false);
  }, []);

  // Execute/trigger the CAPTCHA (for invisible CAPTCHA)
  const execute = useCallback(async (): Promise<string> => {
    // If we already have a token, return it
    if (token) {
      return token;
    }

    // If CAPTCHA is not configured, return empty string
    if (!provider) {
      return "";
    }

    setIsLoading(true);
    setError(null);

    // Return a promise that will be resolved by onVerify
    return new Promise<string>((resolve, reject) => {
      executeResolverRef.current = resolve;
      executeRejecterRef.current = reject;

      // For invisible CAPTCHAs, the widget should call onVerify when done
      // The actual execution is handled by the CAPTCHA widget component
    });
  }, [token, provider]);

  // Mark as ready when provider is set
  useEffect(() => {
    if (provider) {
      setIsReady(true);
    }
  }, [provider]);

  return {
    token,
    isReady,
    isLoading,
    error,
    reset,
    execute,
    onVerify,
    onExpire,
    onError,
  };
}

/**
 * Check if CAPTCHA is required for a specific endpoint
 *
 * @param config - CAPTCHA configuration from useCaptchaConfig
 * @param endpoint - The endpoint to check (e.g., 'signup', 'login', 'password_reset')
 * @returns Whether CAPTCHA is required for this endpoint
 */
export function isCaptchaRequiredForEndpoint(
  config: CaptchaConfig | undefined,
  endpoint: string
): boolean {
  if (!config?.enabled) {
    return false;
  }
  return config.endpoints?.includes(endpoint) ?? false;
}

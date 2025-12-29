/**
 * SAML SSO hooks for Fluxbase React SDK
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type { SAMLLoginOptions, SAMLProvider } from "@fluxbase/sdk";

/**
 * Hook to get available SAML SSO providers
 *
 * @example
 * ```tsx
 * function SAMLProviderList() {
 *   const { data: providers, isLoading } = useSAMLProviders()
 *
 *   if (isLoading) return <div>Loading...</div>
 *
 *   return (
 *     <div>
 *       {providers?.map(provider => (
 *         <button key={provider.id} onClick={() => signInWithSAML(provider.name)}>
 *           Sign in with {provider.name}
 *         </button>
 *       ))}
 *     </div>
 *   )
 * }
 * ```
 */
export function useSAMLProviders() {
  const client = useFluxbaseClient();

  return useQuery<SAMLProvider[]>({
    queryKey: ["fluxbase", "auth", "saml", "providers"],
    queryFn: async () => {
      const { data, error } = await client.auth.getSAMLProviders();
      if (error) throw error;
      return data.providers;
    },
    staleTime: 1000 * 60 * 5, // 5 minutes - providers don't change often
  });
}

/**
 * Hook to get SAML login URL for a provider
 *
 * This hook returns a function to get the login URL for a specific provider.
 * Use this when you need more control over the redirect behavior.
 *
 * @example
 * ```tsx
 * function SAMLLoginButton({ provider }: { provider: string }) {
 *   const getSAMLLoginUrl = useGetSAMLLoginUrl()
 *
 *   const handleClick = async () => {
 *     const { data, error } = await getSAMLLoginUrl.mutateAsync({
 *       provider,
 *       options: { redirectUrl: window.location.href }
 *     })
 *     if (!error) {
 *       window.location.href = data.url
 *     }
 *   }
 *
 *   return <button onClick={handleClick}>Login with {provider}</button>
 * }
 * ```
 */
export function useGetSAMLLoginUrl() {
  const client = useFluxbaseClient();

  return useMutation({
    mutationFn: async ({
      provider,
      options,
    }: {
      provider: string;
      options?: SAMLLoginOptions;
    }) => {
      return await client.auth.getSAMLLoginUrl(provider, options);
    },
  });
}

/**
 * Hook to initiate SAML login (redirects to IdP)
 *
 * This hook returns a mutation that when called, redirects the user to the
 * SAML Identity Provider for authentication.
 *
 * @example
 * ```tsx
 * function SAMLLoginButton() {
 *   const signInWithSAML = useSignInWithSAML()
 *
 *   return (
 *     <button
 *       onClick={() => signInWithSAML.mutate({ provider: 'okta' })}
 *       disabled={signInWithSAML.isPending}
 *     >
 *       {signInWithSAML.isPending ? 'Redirecting...' : 'Sign in with Okta'}
 *     </button>
 *   )
 * }
 * ```
 */
export function useSignInWithSAML() {
  const client = useFluxbaseClient();

  return useMutation({
    mutationFn: async ({
      provider,
      options,
    }: {
      provider: string;
      options?: SAMLLoginOptions;
    }) => {
      return await client.auth.signInWithSAML(provider, options);
    },
  });
}

/**
 * Hook to handle SAML callback after IdP authentication
 *
 * Use this in your SAML callback page to complete the authentication flow.
 *
 * @example
 * ```tsx
 * function SAMLCallbackPage() {
 *   const handleCallback = useHandleSAMLCallback()
 *   const navigate = useNavigate()
 *
 *   useEffect(() => {
 *     const params = new URLSearchParams(window.location.search)
 *     const samlResponse = params.get('SAMLResponse')
 *
 *     if (samlResponse) {
 *       handleCallback.mutate(
 *         { samlResponse },
 *         {
 *           onSuccess: () => navigate('/dashboard'),
 *           onError: (error) => console.error('SAML login failed:', error)
 *         }
 *       )
 *     }
 *   }, [])
 *
 *   if (handleCallback.isPending) {
 *     return <div>Completing sign in...</div>
 *   }
 *
 *   if (handleCallback.isError) {
 *     return <div>Authentication failed: {handleCallback.error.message}</div>
 *   }
 *
 *   return null
 * }
 * ```
 */
export function useHandleSAMLCallback() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      samlResponse,
      provider,
    }: {
      samlResponse: string;
      provider?: string;
    }) => {
      return await client.auth.handleSAMLCallback(samlResponse, provider);
    },
    onSuccess: (result) => {
      if (result.data) {
        // Update auth state in React Query cache
        queryClient.setQueryData(
          ["fluxbase", "auth", "session"],
          result.data.session,
        );
        queryClient.setQueryData(
          ["fluxbase", "auth", "user"],
          result.data.user,
        );
        // Invalidate any dependent queries
        queryClient.invalidateQueries({ queryKey: ["fluxbase"] });
      }
    },
  });
}

/**
 * Hook to get SAML Service Provider metadata URL
 *
 * Returns a function that generates the SP metadata URL for a given provider.
 * Use this URL when configuring your SAML IdP.
 *
 * @example
 * ```tsx
 * function SAMLSetupInfo({ provider }: { provider: string }) {
 *   const getSAMLMetadataUrl = useSAMLMetadataUrl()
 *   const metadataUrl = getSAMLMetadataUrl(provider)
 *
 *   return (
 *     <div>
 *       <p>SP Metadata URL:</p>
 *       <code>{metadataUrl}</code>
 *     </div>
 *   )
 * }
 * ```
 */
export function useSAMLMetadataUrl() {
  const client = useFluxbaseClient();

  return (provider: string): string => {
    return client.auth.getSAMLMetadataUrl(provider);
  };
}

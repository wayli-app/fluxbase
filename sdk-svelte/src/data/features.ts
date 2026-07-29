/**
 * Feature stores for the Fluxbase Svelte SDK.
 *
 * Covers functions, jobs, branching, RPC, vector, secrets, GraphQL, and the
 * admin/management surfaces. Each is a thin TanStack Svelte Query wrapper over
 * the corresponding core-SDK module. Read results with `$store`.
 */

import { createQuery, createMutation } from "@tanstack/svelte-query";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import type { StoreDeps } from "../auth/store";

const key = (...parts: unknown[]) => ["fluxbase", ...parts];

// ---------------------------------------------------------------------------
// Edge functions
// ---------------------------------------------------------------------------

/** Invoke an edge function. */
export function createInvokeFunction({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (params: {
        name: string;
        options?: Parameters<FluxbaseClient["functions"]["invoke"]>[1];
      }) => {
        const { data, error } = await client.functions.invoke(
          params.name,
          params.options,
        );
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** List edge functions reactively. */
export function createFunctions({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("functions"),
      queryFn: async () => {
        const { data, error } = await client.functions.list();
        if (error) throw error;
        return data ?? [];
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Jobs
// ---------------------------------------------------------------------------

/** List jobs reactively. */
export function createJobs({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("jobs"),
      queryFn: async () => {
        const { data, error } = await client.jobs.list();
        if (error) throw error;
        return data ?? [];
      },
    },
    queryClient,
  );
}

/** Get a single job reactively. */
export function createJobStatus({ client, queryClient }: StoreDeps, jobId: string) {
  return createQuery(
    {
      queryKey: key("jobs", jobId),
      queryFn: async () => {
        const { data, error } = await client.jobs.get(jobId);
        if (error) throw error;
        return data;
      },
      enabled: !!jobId,
    },
    queryClient,
  );
}

/** Submit a job, then refresh the jobs list. */
export function createSubmitJob({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        params: Parameters<FluxbaseClient["jobs"]["submit"]>[0],
      ) => {
        const { data, error } = await client.jobs.submit(params);
        if (error) throw error;
        return data;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("jobs") });
      },
    },
    queryClient,
  );
}

/** Cancel a job, then refresh. */
export function createCancelJob({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (jobId: string) => {
        const { error } = await client.jobs.cancel(jobId);
        if (error) throw error;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("jobs") });
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Branching
// ---------------------------------------------------------------------------

/** List branches reactively. */
export function createBranches({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("branches"),
      queryFn: async () => {
        const { data, error } = await client.branching.list();
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** Create a branch, then refresh. */
export function createCreateBranch({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        params: Parameters<FluxbaseClient["branching"]["create"]>[0],
      ) => {
        const { data, error } = await client.branching.create(params);
        if (error) throw error;
        return data;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("branches") });
      },
    },
    queryClient,
  );
}

/** Delete a branch, then refresh. */
export function createDeleteBranch({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (branchId: string) => {
        const { error } = await client.branching.delete(branchId);
        if (error) throw error;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("branches") });
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// RPC
// ---------------------------------------------------------------------------

/** List RPC functions reactively. */
export function createRPCList({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("rpc"),
      queryFn: async () => {
        const { data, error } = await client.rpc.list();
        if (error) throw error;
        return data ?? [];
      },
    },
    queryClient,
  );
}

/** Invoke an RPC function. */
export function createInvokeRPC({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (params: {
        name: string;
        params?: Record<string, unknown>;
        options?: Parameters<FluxbaseClient["rpc"]["invoke"]>[2];
      }) => {
        const { data, error } = await client.rpc.invoke(
          params.name,
          params.params,
          params.options,
        );
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Vector
// ---------------------------------------------------------------------------

/** Embed text (or a structured embedding request). */
export function createVectorEmbed({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        request: Parameters<FluxbaseClient["vector"]["embed"]>[0],
      ) => {
        const { data, error } = await client.vector.embed(request);
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** Semantic search. */
export function createVectorSearch({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        params: Parameters<FluxbaseClient["vector"]["search"]>[0],
      ) => {
        const { data, error } = await client.vector.search(params);
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Secrets
// ---------------------------------------------------------------------------

/** List secrets reactively. */
export function createSecrets({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("secrets"),
      queryFn: async () => {
        // secrets.list() returns SecretSummary[] directly (not {data,error})
        const result = await client.secrets.list();
        return Array.isArray(result) ? result : [];
      },
    },
    queryClient,
  );
}

/** Create a secret, then refresh. */
export function createCreateSecret({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        params: Parameters<FluxbaseClient["secrets"]["create"]>[0],
      ) => {
        return await client.secrets.create(params);
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("secrets") });
      },
    },
    queryClient,
  );
}

/** Update a secret, then refresh. */
export function createUpdateSecret({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (params: {
        name: string;
        request: Parameters<FluxbaseClient["secrets"]["update"]>[1];
        options?: Parameters<FluxbaseClient["secrets"]["update"]>[2];
      }) => {
        return await client.secrets.update(
          params.name,
          params.request,
          params.options,
        );
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("secrets") });
      },
    },
    queryClient,
  );
}

/** Delete a secret, then refresh. */
export function createDeleteSecret({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (name: string) => {
        await client.secrets.delete(name);
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: key("secrets") });
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// GraphQL
// ---------------------------------------------------------------------------

/** Run a GraphQL query reactively. */
export function createGraphQLQuery(
  { client, queryClient }: StoreDeps,
  query: string,
  variables?: Record<string, unknown>,
) {
  return createQuery(
    {
      queryKey: key("graphql", query, variables ?? {}),
      queryFn: async () => {
        const { data, errors } = await client.graphql.query(query, variables);
        if (errors?.length) throw errors[0];
        return data;
      },
    },
    queryClient,
  );
}

/** Run a GraphQL mutation. */
export function createGraphQLMutation({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (params: {
        query: string;
        variables?: Record<string, unknown>;
      }) => {
        const { data, errors } = await client.graphql.mutation(
          params.query,
          params.variables,
        );
        if (errors?.length) throw errors[0];
        return data;
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// SAML
// ---------------------------------------------------------------------------

/** List SAML providers reactively. */
export function createSAMLProviders({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("saml", "providers"),
      queryFn: async () => {
        const { data, error } = await client.auth.getSAMLProviders();
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** Initiate SAML sign-in (returns a URL to redirect to). */
export function createSignInWithSAML({ client, queryClient }: StoreDeps) {
  return createMutation(
    {
      mutationFn: async (
        params: Parameters<FluxbaseClient["auth"]["signInWithSAML"]>[0],
      ) => {
        const { data, error } = await client.auth.signInWithSAML(params);
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Captcha & auth config
// ---------------------------------------------------------------------------

/** Read captcha configuration reactively. */
export function createCaptchaConfig({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("captcha", "config"),
      queryFn: async () => {
        const { data, error } = await client.auth.getCaptchaConfig();
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** Read auth configuration reactively. */
export function createAuthConfig({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("auth", "config"),
      queryFn: async () => {
        const { data, error } = await client.auth.getAuthConfig();
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

/** List users (admin). */
export function createUsers({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("admin", "users"),
      queryFn: async () => {
        const { data, error } = await client.admin.listUsers();
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** List webhooks (admin, via management). */
export function createWebhooks({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("admin", "webhooks"),
      queryFn: async () => {
        // The admin surface exposes webhooks under management.
        const result = await (client.admin as any).management.webhooks.list();
        return result?.webhooks ?? [];
      },
    },
    queryClient,
  );
}

/** Read app settings (admin). */
export function createAppSettings({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("admin", "settings", "app"),
      queryFn: async () => {
        return await (client.admin as any).settings.app.get();
      },
    },
    queryClient,
  );
}

/** Read system settings (admin). */
export function createSystemSettings({ client, queryClient }: StoreDeps) {
  return createQuery(
    {
      queryKey: key("admin", "settings", "system"),
      queryFn: async () => {
        const result = await (client.admin as any).settings.system.list();
        return result?.settings ?? [];
      },
    },
    queryClient,
  );
}

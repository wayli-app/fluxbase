/**
 * GraphQL hooks for Fluxbase React SDK
 *
 * Provides React Query integration for GraphQL queries and mutations.
 *
 * @example
 * ```tsx
 * import { useGraphQLQuery, useGraphQLMutation } from '@fluxbase/sdk-react'
 *
 * function UsersList() {
 *   const { data, isLoading, error } = useGraphQLQuery<UsersQuery>(
 *     'users-list',
 *     `query { users { id email } }`
 *   )
 *
 *   if (isLoading) return <div>Loading...</div>
 *   if (error) return <div>Error: {error.message}</div>
 *
 *   return (
 *     <ul>
 *       {data?.users.map(user => (
 *         <li key={user.id}>{user.email}</li>
 *       ))}
 *     </ul>
 *   )
 * }
 * ```
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  GraphQLResponse,
  GraphQLError,
  GraphQLRequestOptions,
} from "@fluxbase/sdk";

/**
 * Options for useGraphQLQuery hook
 */
export interface UseGraphQLQueryOptions<T> {
  /**
   * Variables to pass to the GraphQL query
   */
  variables?: Record<string, unknown>;

  /**
   * Operation name when the document contains multiple operations
   */
  operationName?: string;

  /**
   * Additional request options
   */
  requestOptions?: GraphQLRequestOptions;

  /**
   * Whether the query is enabled
   * @default true
   */
  enabled?: boolean;

  /**
   * Time in milliseconds after which the query is considered stale
   * @default 0 (considered stale immediately)
   */
  staleTime?: number;

  /**
   * Time in milliseconds after which inactive query data is garbage collected
   * @default 5 minutes
   */
  gcTime?: number;

  /**
   * Whether to refetch on window focus
   * @default true
   */
  refetchOnWindowFocus?: boolean;

  /**
   * Transform function to process the response data
   */
  select?: (data: T | undefined) => T | undefined;
}

/**
 * Options for useGraphQLMutation hook
 */
export interface UseGraphQLMutationOptions<T, V> {
  /**
   * Operation name when the document contains multiple operations
   */
  operationName?: string;

  /**
   * Additional request options
   */
  requestOptions?: GraphQLRequestOptions;

  /**
   * Callback when mutation succeeds
   */
  onSuccess?: (data: T, variables: V) => void;

  /**
   * Callback when mutation fails
   */
  onError?: (error: GraphQLError, variables: V) => void;

  /**
   * Query keys to invalidate on success
   */
  invalidateQueries?: string[];
}

/**
 * Hook to execute GraphQL queries with React Query caching
 *
 * @typeParam T - The expected response data type
 * @param queryKey - Unique key for caching (string or array)
 * @param query - The GraphQL query string
 * @param options - Query options including variables
 * @returns React Query result object
 *
 * @example
 * ```tsx
 * interface UsersQuery {
 *   users: Array<{ id: string; email: string }>
 * }
 *
 * function UsersList() {
 *   const { data, isLoading } = useGraphQLQuery<UsersQuery>(
 *     'users',
 *     `query { users { id email } }`
 *   )
 *
 *   return <div>{data?.users.length} users</div>
 * }
 * ```
 *
 * @example
 * ```tsx
 * // With variables
 * const { data } = useGraphQLQuery<UserQuery>(
 *   ['user', userId],
 *   `query GetUser($id: ID!) { user(id: $id) { id email } }`,
 *   { variables: { id: userId } }
 * )
 * ```
 */
export function useGraphQLQuery<T = unknown>(
  queryKey: string | readonly unknown[],
  query: string,
  options?: UseGraphQLQueryOptions<T>
) {
  const client = useFluxbaseClient();
  const normalizedKey = Array.isArray(queryKey)
    ? ["fluxbase", "graphql", ...queryKey]
    : ["fluxbase", "graphql", queryKey];

  return useQuery<T | undefined, GraphQLError>({
    queryKey: normalizedKey,
    queryFn: async () => {
      const response: GraphQLResponse<T> = await client.graphql.execute(
        query,
        options?.variables,
        options?.operationName,
        options?.requestOptions
      );

      if (response.errors && response.errors.length > 0) {
        throw response.errors[0];
      }

      return response.data;
    },
    enabled: options?.enabled ?? true,
    staleTime: options?.staleTime ?? 0,
    gcTime: options?.gcTime,
    refetchOnWindowFocus: options?.refetchOnWindowFocus ?? true,
    select: options?.select,
  });
}

/**
 * Hook to execute GraphQL mutations
 *
 * @typeParam T - The expected response data type
 * @typeParam V - The variables type
 * @param mutation - The GraphQL mutation string
 * @param options - Mutation options
 * @returns React Query mutation result object
 *
 * @example
 * ```tsx
 * interface CreateUserMutation {
 *   insertUser: { id: string; email: string }
 * }
 *
 * interface CreateUserVariables {
 *   data: { email: string }
 * }
 *
 * function CreateUserForm() {
 *   const mutation = useGraphQLMutation<CreateUserMutation, CreateUserVariables>(
 *     `mutation CreateUser($data: UserInput!) {
 *       insertUser(data: $data) { id email }
 *     }`,
 *     {
 *       onSuccess: (data) => console.log('Created:', data.insertUser),
 *       invalidateQueries: ['users']
 *     }
 *   )
 *
 *   const handleSubmit = (email: string) => {
 *     mutation.mutate({ data: { email } })
 *   }
 *
 *   return (
 *     <button
 *       onClick={() => handleSubmit('new@example.com')}
 *       disabled={mutation.isPending}
 *     >
 *       Create User
 *     </button>
 *   )
 * }
 * ```
 */
export function useGraphQLMutation<
  T = unknown,
  V extends Record<string, unknown> = Record<string, unknown>,
>(mutation: string, options?: UseGraphQLMutationOptions<T, V>) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation<T | undefined, GraphQLError, V>({
    mutationFn: async (variables: V) => {
      const response: GraphQLResponse<T> = await client.graphql.execute(
        mutation,
        variables,
        options?.operationName,
        options?.requestOptions
      );

      if (response.errors && response.errors.length > 0) {
        throw response.errors[0];
      }

      return response.data;
    },
    onSuccess: (data, variables) => {
      // Invalidate specified queries
      if (options?.invalidateQueries) {
        for (const key of options.invalidateQueries) {
          queryClient.invalidateQueries({
            queryKey: ["fluxbase", "graphql", key],
          });
        }
      }

      // Call user's onSuccess callback
      if (options?.onSuccess && data !== undefined) {
        options.onSuccess(data, variables);
      }
    },
    onError: (error, variables) => {
      if (options?.onError) {
        options.onError(error, variables);
      }
    },
  });
}

/**
 * Hook to fetch the GraphQL schema via introspection
 *
 * @param options - Query options
 * @returns React Query result with schema introspection data
 *
 * @example
 * ```tsx
 * function SchemaExplorer() {
 *   const { data, isLoading } = useGraphQLIntrospection()
 *
 *   if (isLoading) return <div>Loading schema...</div>
 *
 *   return (
 *     <div>
 *       <p>Query type: {data?.__schema.queryType.name}</p>
 *       <p>Types: {data?.__schema.types.length}</p>
 *     </div>
 *   )
 * }
 * ```
 */
export function useGraphQLIntrospection(options?: {
  enabled?: boolean;
  staleTime?: number;
  requestOptions?: GraphQLRequestOptions;
}) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "graphql", "__introspection"],
    queryFn: async () => {
      const response = await client.graphql.introspect(options?.requestOptions);

      if (response.errors && response.errors.length > 0) {
        throw response.errors[0];
      }

      return response.data;
    },
    enabled: options?.enabled ?? true,
    staleTime: options?.staleTime ?? 1000 * 60 * 5, // 5 minutes - schema doesn't change often
  });
}

/**
 * Hook to execute raw GraphQL operations (query or mutation)
 *
 * This is a lower-level hook that doesn't use React Query caching.
 * Useful for one-off operations or when you need full control.
 *
 * @returns Functions to execute queries and mutations
 *
 * @example
 * ```tsx
 * function AdminPanel() {
 *   const { executeQuery, executeMutation } = useGraphQL()
 *
 *   const handleExport = async () => {
 *     const result = await executeQuery<ExportData>(
 *       `query { exportAllData { url } }`
 *     )
 *     if (result.data) {
 *       window.open(result.data.exportAllData.url)
 *     }
 *   }
 *
 *   return <button onClick={handleExport}>Export Data</button>
 * }
 * ```
 */
export function useGraphQL() {
  const client = useFluxbaseClient();

  return {
    /**
     * Execute a GraphQL query
     */
    executeQuery: <T = unknown>(
      query: string,
      variables?: Record<string, unknown>,
      options?: GraphQLRequestOptions
    ): Promise<GraphQLResponse<T>> => {
      return client.graphql.query<T>(query, variables, options);
    },

    /**
     * Execute a GraphQL mutation
     */
    executeMutation: <T = unknown>(
      mutation: string,
      variables?: Record<string, unknown>,
      options?: GraphQLRequestOptions
    ): Promise<GraphQLResponse<T>> => {
      return client.graphql.mutation<T>(mutation, variables, options);
    },

    /**
     * Execute a GraphQL operation with an explicit operation name
     */
    execute: <T = unknown>(
      document: string,
      variables?: Record<string, unknown>,
      operationName?: string,
      options?: GraphQLRequestOptions
    ): Promise<GraphQLResponse<T>> => {
      return client.graphql.execute<T>(document, variables, operationName, options);
    },

    /**
     * Fetch the GraphQL schema via introspection
     */
    introspect: (options?: GraphQLRequestOptions) => {
      return client.graphql.introspect(options);
    },
  };
}

/**
 * GraphQL client for Fluxbase
 *
 * Provides a type-safe interface for executing GraphQL queries and mutations
 * against the auto-generated GraphQL schema from your database tables.
 *
 * @example
 * ```typescript
 * import { createClient } from '@fluxbase/sdk'
 *
 * const client = createClient({ url: 'http://localhost:8080' })
 *
 * // Execute a query
 * const { data, errors } = await client.graphql.query(`
 *   query GetUsers($limit: Int) {
 *     users(limit: $limit) {
 *       id
 *       email
 *       posts {
 *         title
 *       }
 *     }
 *   }
 * `, { limit: 10 })
 *
 * // Execute a mutation
 * const { data, errors } = await client.graphql.mutation(`
 *   mutation CreateUser($data: UserInput!) {
 *     insertUser(data: $data) {
 *       id
 *       email
 *     }
 *   }
 * `, { data: { email: 'user@example.com' } })
 * ```
 *
 * @category GraphQL
 */

import { FluxbaseFetch } from "./fetch";

/**
 * GraphQL error location in the query
 */
export interface GraphQLErrorLocation {
  line: number;
  column: number;
}

/**
 * GraphQL error returned from the server
 */
export interface GraphQLError {
  message: string;
  locations?: GraphQLErrorLocation[];
  path?: (string | number)[];
  extensions?: Record<string, unknown>;
}

/**
 * GraphQL response from the server
 */
export interface GraphQLResponse<T = unknown> {
  data?: T;
  errors?: GraphQLError[];
}

/**
 * Options for GraphQL requests
 */
export interface GraphQLRequestOptions {
  /**
   * Custom headers to include in the request
   */
  headers?: Record<string, string>;

  /**
   * Request timeout in milliseconds
   */
  timeout?: number;
}

/**
 * GraphQL client class for executing queries and mutations
 *
 * @category GraphQL
 */
export class FluxbaseGraphQL {
  private fetch: FluxbaseFetch;

  /**
   * Create a new GraphQL client
   * @param fetch - The HTTP client to use for requests
   */
  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Execute a GraphQL query
   *
   * @typeParam T - The expected response data type
   * @param query - The GraphQL query string
   * @param variables - Variables to pass to the query
   * @param options - Additional request options
   * @returns Promise resolving to the GraphQL response
   *
   * @example
   * ```typescript
   * interface UsersQuery {
   *   users: Array<{ id: string; email: string }>
   * }
   *
   * const { data, errors } = await client.graphql.query<UsersQuery>(`
   *   query {
   *     users { id email }
   *   }
   * `)
   *
   * if (data) {
   *   console.log(data.users)
   * }
   * ```
   */
  async query<T = unknown>(
    query: string,
    variables?: Record<string, unknown>,
    options?: GraphQLRequestOptions
  ): Promise<GraphQLResponse<T>> {
    return this.execute<T>(query, variables, undefined, options);
  }

  /**
   * Execute a GraphQL mutation
   *
   * @typeParam T - The expected response data type
   * @param mutation - The GraphQL mutation string
   * @param variables - Variables to pass to the mutation
   * @param options - Additional request options
   * @returns Promise resolving to the GraphQL response
   *
   * @example
   * ```typescript
   * interface CreateUserMutation {
   *   insertUser: { id: string; email: string }
   * }
   *
   * const { data, errors } = await client.graphql.mutation<CreateUserMutation>(`
   *   mutation CreateUser($data: UserInput!) {
   *     insertUser(data: $data) {
   *       id
   *       email
   *     }
   *   }
   * `, { data: { email: 'user@example.com' } })
   * ```
   */
  async mutation<T = unknown>(
    mutation: string,
    variables?: Record<string, unknown>,
    options?: GraphQLRequestOptions
  ): Promise<GraphQLResponse<T>> {
    return this.execute<T>(mutation, variables, undefined, options);
  }

  /**
   * Execute a GraphQL request with an operation name
   *
   * Use this when your query document contains multiple operations
   * and you need to specify which one to execute.
   *
   * @typeParam T - The expected response data type
   * @param query - The GraphQL document containing one or more operations
   * @param variables - Variables to pass to the operation
   * @param operationName - The name of the operation to execute
   * @param options - Additional request options
   * @returns Promise resolving to the GraphQL response
   *
   * @example
   * ```typescript
   * const { data } = await client.graphql.execute(`
   *   query GetUser($id: ID!) {
   *     user(id: $id) { id email }
   *   }
   *   query ListUsers {
   *     users { id email }
   *   }
   * `, { id: '123' }, 'GetUser')
   * ```
   */
  async execute<T = unknown>(
    query: string,
    variables?: Record<string, unknown>,
    operationName?: string,
    options?: GraphQLRequestOptions
  ): Promise<GraphQLResponse<T>> {
    const body: Record<string, unknown> = { query };

    if (variables && Object.keys(variables).length > 0) {
      body.variables = variables;
    }

    if (operationName) {
      body.operationName = operationName;
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...options?.headers,
    };

    try {
      const response = await this.fetch.post<GraphQLResponse<T>>(
        "/api/v1/graphql",
        body,
        { headers }
      );

      // Return the GraphQL response (which may contain both data and errors)
      return response || { errors: [{ message: "No response received" }] };
    } catch (err) {
      // Network or HTTP error
      const message = err instanceof Error ? err.message : "GraphQL request failed";
      return {
        errors: [{ message }],
      };
    }
  }

  /**
   * Fetch the GraphQL schema via introspection
   *
   * Returns the full schema information including types, fields, and directives.
   * Useful for tooling and documentation.
   *
   * @param options - Additional request options
   * @returns Promise resolving to the introspection result
   *
   * @example
   * ```typescript
   * const { data, errors } = await client.graphql.introspect()
   *
   * if (data) {
   *   console.log('Types:', data.__schema.types.length)
   * }
   * ```
   */
  async introspect(
    options?: GraphQLRequestOptions
  ): Promise<GraphQLResponse<{ __schema: IntrospectionSchema }>> {
    return this.query(INTROSPECTION_QUERY, undefined, options);
  }
}

/**
 * GraphQL introspection schema type
 */
export interface IntrospectionSchema {
  queryType: { name: string };
  mutationType?: { name: string };
  subscriptionType?: { name: string };
  types: IntrospectionType[];
  directives: IntrospectionDirective[];
}

/**
 * GraphQL introspection type
 */
export interface IntrospectionType {
  kind: string;
  name: string;
  description?: string;
  fields?: IntrospectionField[];
  inputFields?: IntrospectionInputValue[];
  interfaces?: IntrospectionTypeRef[];
  enumValues?: IntrospectionEnumValue[];
  possibleTypes?: IntrospectionTypeRef[];
}

/**
 * GraphQL introspection field
 */
export interface IntrospectionField {
  name: string;
  description?: string;
  args: IntrospectionInputValue[];
  type: IntrospectionTypeRef;
  isDeprecated: boolean;
  deprecationReason?: string;
}

/**
 * GraphQL introspection input value
 */
export interface IntrospectionInputValue {
  name: string;
  description?: string;
  type: IntrospectionTypeRef;
  defaultValue?: string;
}

/**
 * GraphQL introspection type reference
 */
export interface IntrospectionTypeRef {
  kind: string;
  name?: string;
  ofType?: IntrospectionTypeRef;
}

/**
 * GraphQL introspection enum value
 */
export interface IntrospectionEnumValue {
  name: string;
  description?: string;
  isDeprecated: boolean;
  deprecationReason?: string;
}

/**
 * GraphQL introspection directive
 */
export interface IntrospectionDirective {
  name: string;
  description?: string;
  locations: string[];
  args: IntrospectionInputValue[];
}

/**
 * Standard GraphQL introspection query
 */
const INTROSPECTION_QUERY = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`;

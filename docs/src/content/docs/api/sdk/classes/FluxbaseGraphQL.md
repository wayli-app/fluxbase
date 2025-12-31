---
editUrl: false
next: false
prev: false
title: "FluxbaseGraphQL"
---

GraphQL client class for executing queries and mutations

## Constructors

### new FluxbaseGraphQL()

> **new FluxbaseGraphQL**(`fetch`): [`FluxbaseGraphQL`](/api/sdk/classes/fluxbasegraphql/)

Create a new GraphQL client

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) | The HTTP client to use for requests |

#### Returns

[`FluxbaseGraphQL`](/api/sdk/classes/fluxbasegraphql/)

## Methods

### execute()

> **execute**\<`T`\>(`query`, `variables`?, `operationName`?, `options`?): `Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Execute a GraphQL request with an operation name

Use this when your query document contains multiple operations
and you need to specify which one to execute.

#### Type Parameters

| Type Parameter | Default type | Description |
| ------ | ------ | ------ |
| `T` | `unknown` | The expected response data type |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `query` | `string` | The GraphQL document containing one or more operations |
| `variables`? | `Record`\<`string`, `unknown`\> | Variables to pass to the operation |
| `operationName`? | `string` | The name of the operation to execute |
| `options`? | [`GraphQLRequestOptions`](/api/sdk/interfaces/graphqlrequestoptions/) | Additional request options |

#### Returns

`Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Promise resolving to the GraphQL response

#### Example

```typescript
const { data } = await client.graphql.execute(`
  query GetUser($id: ID!) {
    user(id: $id) { id email }
  }
  query ListUsers {
    users { id email }
  }
`, { id: '123' }, 'GetUser')
```

***

### introspect()

> **introspect**(`options`?): `Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`object`\>\>

Fetch the GraphQL schema via introspection

Returns the full schema information including types, fields, and directives.
Useful for tooling and documentation.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | [`GraphQLRequestOptions`](/api/sdk/interfaces/graphqlrequestoptions/) | Additional request options |

#### Returns

`Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`object`\>\>

Promise resolving to the introspection result

#### Example

```typescript
const { data, errors } = await client.graphql.introspect()

if (data) {
  console.log('Types:', data.__schema.types.length)
}
```

***

### mutation()

> **mutation**\<`T`\>(`mutation`, `variables`?, `options`?): `Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Execute a GraphQL mutation

#### Type Parameters

| Type Parameter | Default type | Description |
| ------ | ------ | ------ |
| `T` | `unknown` | The expected response data type |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `mutation` | `string` | The GraphQL mutation string |
| `variables`? | `Record`\<`string`, `unknown`\> | Variables to pass to the mutation |
| `options`? | [`GraphQLRequestOptions`](/api/sdk/interfaces/graphqlrequestoptions/) | Additional request options |

#### Returns

`Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Promise resolving to the GraphQL response

#### Example

```typescript
interface CreateUserMutation {
  insertUser: { id: string; email: string }
}

const { data, errors } = await client.graphql.mutation<CreateUserMutation>(`
  mutation CreateUser($data: UserInput!) {
    insertUser(data: $data) {
      id
      email
    }
  }
`, { data: { email: 'user@example.com' } })
```

***

### query()

> **query**\<`T`\>(`query`, `variables`?, `options`?): `Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Execute a GraphQL query

#### Type Parameters

| Type Parameter | Default type | Description |
| ------ | ------ | ------ |
| `T` | `unknown` | The expected response data type |

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `query` | `string` | The GraphQL query string |
| `variables`? | `Record`\<`string`, `unknown`\> | Variables to pass to the query |
| `options`? | [`GraphQLRequestOptions`](/api/sdk/interfaces/graphqlrequestoptions/) | Additional request options |

#### Returns

`Promise`\<[`GraphQLResponse`](/api/sdk/interfaces/graphqlresponse/)\<`T`\>\>

Promise resolving to the GraphQL response

#### Example

```typescript
interface UsersQuery {
  users: Array<{ id: string; email: string }>
}

const { data, errors } = await client.graphql.query<UsersQuery>(`
  query {
    users { id email }
  }
`)

if (data) {
  console.log(data.users)
}
```

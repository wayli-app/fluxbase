---
editUrl: false
next: false
prev: false
title: "UseGraphQLQueryOptions"
---

Options for useGraphQLQuery hook

## Type Parameters

| Type Parameter |
| ------ |
| `T` |

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `enabled?` | `boolean` | Whether the query is enabled **Default** `true` |
| `gcTime?` | `number` | Time in milliseconds after which inactive query data is garbage collected **Default** `5 minutes` |
| `operationName?` | `string` | Operation name when the document contains multiple operations |
| `refetchOnWindowFocus?` | `boolean` | Whether to refetch on window focus **Default** `true` |
| `requestOptions?` | [`GraphQLRequestOptions`](/api/sdk-react/interfaces/graphqlrequestoptions/) | Additional request options |
| `select?` | (`data`: `undefined` \| `T`) => `undefined` \| `T` | Transform function to process the response data |
| `staleTime?` | `number` | Time in milliseconds after which the query is considered stale **Default** `0 (considered stale immediately)` |
| `variables?` | `Record`\<`string`, `unknown`\> | Variables to pass to the GraphQL query |

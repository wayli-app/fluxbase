---
editUrl: false
next: false
prev: false
title: "UseGraphQLMutationOptions"
---

Options for useGraphQLMutation hook

## Type Parameters

| Type Parameter |
| ------ |
| `T` |
| `V` |

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `invalidateQueries?` | `string`[] | Query keys to invalidate on success |
| `onError?` | (`error`: [`GraphQLError`](/api/sdk-react/interfaces/graphqlerror/), `variables`: `V`) => `void` | Callback when mutation fails |
| `onSuccess?` | (`data`: `T`, `variables`: `V`) => `void` | Callback when mutation succeeds |
| `operationName?` | `string` | Operation name when the document contains multiple operations |
| `requestOptions?` | [`GraphQLRequestOptions`](/api/sdk-react/interfaces/graphqlrequestoptions/) | Additional request options |

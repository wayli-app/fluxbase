---
editUrl: false
next: false
prev: false
title: "useGraphQLMutation"
---

> **useGraphQLMutation**\<`T`, `V`\>(`mutation`, `options`?): `UseMutationResult`\<`undefined` \| `T`, [`GraphQLError`](/api/sdk-react/interfaces/graphqlerror/), `V`, `unknown`\>

Hook to execute GraphQL mutations

## Type Parameters

| Type Parameter | Default type | Description |
| ------ | ------ | ------ |
| `T` | `unknown` | The expected response data type |
| `V` *extends* `Record`\<`string`, `unknown`\> | `Record`\<`string`, `unknown`\> | The variables type |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `mutation` | `string` | The GraphQL mutation string |
| `options`? | [`UseGraphQLMutationOptions`](/api/sdk-react/interfaces/usegraphqlmutationoptions/)\<`T`, `V`\> | Mutation options |

## Returns

`UseMutationResult`\<`undefined` \| `T`, [`GraphQLError`](/api/sdk-react/interfaces/graphqlerror/), `V`, `unknown`\>

React Query mutation result object

## Example

```tsx
interface CreateUserMutation {
  insertUser: { id: string; email: string }
}

interface CreateUserVariables {
  data: { email: string }
}

function CreateUserForm() {
  const mutation = useGraphQLMutation<CreateUserMutation, CreateUserVariables>(
    `mutation CreateUser($data: UserInput!) {
      insertUser(data: $data) { id email }
    }`,
    {
      onSuccess: (data) => console.log('Created:', data.insertUser),
      invalidateQueries: ['users']
    }
  )

  const handleSubmit = (email: string) => {
    mutation.mutate({ data: { email } })
  }

  return (
    <button
      onClick={() => handleSubmit('new@example.com')}
      disabled={mutation.isPending}
    >
      Create User
    </button>
  )
}
```

---
editUrl: false
next: false
prev: false
title: "FluxbaseQueryOptions"
---

## Extends

- `Omit`\<`CreateQueryOptions`, `"queryKey"` \| `"queryFn"`\>

## Type Parameters

| Type Parameter |
| ------ |
| `T` |

## Properties

| Property | Type | Description | Inherited from |
| ------ | ------ | ------ | ------ |
| <a id="_defaulted"></a> `_defaulted?` | `boolean` | - | `Omit._defaulted` |
| <a id="_optimisticresults"></a> `_optimisticResults?` | `"optimistic"` \| `"isRestoring"` | - | `Omit._optimisticResults` |
| <a id="behavior"></a> `behavior?` | `QueryBehavior`\<`T`[], `Error`, `T`[], readonly `unknown`[]\> | - | `Omit.behavior` |
| <a id="enabled"></a> `enabled?` | `Enabled`\<`T`[], `Error`, `T`[], readonly `unknown`[]\> | Set this to `false` or a function that returns `false` to disable automatic refetching when the query mounts or changes query keys. To refetch the query, use the `refetch` method returned from the `useQuery` instance. Accepts a boolean or function that returns a boolean. Defaults to `true`. | `Omit.enabled` |
| <a id="experimental_prefetchinrender"></a> `experimental_prefetchInRender?` | `boolean` | Enable prefetching during rendering | `Omit.experimental_prefetchInRender` |
| <a id="gctime"></a> `gcTime?` | `number` | The time in milliseconds that unused/inactive cache data remains in memory. When a query's cache becomes unused or inactive, that cache data will be garbage collected after this duration. When different garbage collection times are specified, the longest one will be used. Setting it to `Infinity` will disable garbage collection. | `Omit.gcTime` |
| <a id="initialdata"></a> `initialData?` | `T`[] \| `InitialDataFunction`\<`T`[]\> | - | `Omit.initialData` |
| <a id="initialdataupdatedat"></a> `initialDataUpdatedAt?` | `number` \| (() => `number` \| `undefined`) | - | `Omit.initialDataUpdatedAt` |
| <a id="maxpages"></a> `maxPages?` | `number` | Maximum number of pages to store in the data of an infinite query. | `Omit.maxPages` |
| <a id="meta"></a> `meta?` | `Record`\<`string`, `unknown`\> | Additional payload to be stored on each query. Use this property to pass information that can be used in other places. | `Omit.meta` |
| <a id="networkmode"></a> `networkMode?` | `NetworkMode` | - | `Omit.networkMode` |
| <a id="notifyonchangeprops"></a> `notifyOnChangeProps?` | `NotifyOnChangeProps` | If set, the component will only re-render if any of the listed properties change. When set to `['data', 'error']`, the component will only re-render when the `data` or `error` properties change. When set to `'all'`, the component will re-render whenever a query is updated. When set to a function, the function will be executed to compute the list of properties. By default, access to properties will be tracked, and the component will only re-render when one of the tracked properties change. | `Omit.notifyOnChangeProps` |
| <a id="persister"></a> `persister?` | (`queryFn`, `context`, `query`) => `T`[] \| `Promise`\<`T`[]\> | - | `Omit.persister` |
| <a id="placeholderdata"></a> `placeholderData?` | `T`[] \| `PlaceholderDataFunction`\<`T`[], `Error`, `T`[], readonly `unknown`[]\> | If set, this value will be used as the placeholder data for this particular query observer while the query is still in the `loading` data and no initialData has been provided. | `Omit.placeholderData` |
| <a id="queryhash"></a> `queryHash?` | `string` | - | `Omit.queryHash` |
| <a id="querykey"></a> `queryKey?` | `unknown`[] | Stable query key. Strongly recommended for reliable caching. | - |
| <a id="querykeyhashfn"></a> `queryKeyHashFn?` | `QueryKeyHashFunction`\<readonly `unknown`[]\> | - | `Omit.queryKeyHashFn` |
| <a id="refetchinterval"></a> `refetchInterval?` | `number` \| `false` \| ((`query`) => `number` \| `false` \| `undefined`) | If set to a number, the query will continuously refetch at this frequency in milliseconds. If set to a function, the function will be executed with the latest data and query to compute a frequency Defaults to `false`. | `Omit.refetchInterval` |
| <a id="refetchintervalinbackground"></a> `refetchIntervalInBackground?` | `boolean` | If set to `true`, the query will continue to refetch while their tab/window is in the background. Defaults to `false`. | `Omit.refetchIntervalInBackground` |
| <a id="refetchonmount"></a> `refetchOnMount?` | `boolean` \| `"always"` \| ((`query`) => `boolean` \| `"always"`) | If set to `true`, the query will refetch on mount if the data is stale. If set to `false`, will disable additional instances of a query to trigger background refetch. If set to `'always'`, the query will always refetch on mount. If set to a function, the function will be executed with the latest data and query to compute the value Defaults to `true`. | `Omit.refetchOnMount` |
| <a id="refetchonreconnect"></a> `refetchOnReconnect?` | `boolean` \| `"always"` \| ((`query`) => `boolean` \| `"always"`) | If set to `true`, the query will refetch on reconnect if the data is stale. If set to `false`, the query will not refetch on reconnect. If set to `'always'`, the query will always refetch on reconnect. If set to a function, the function will be executed with the latest data and query to compute the value. Defaults to the value of `networkOnline` (`true`) | `Omit.refetchOnReconnect` |
| <a id="refetchonwindowfocus"></a> `refetchOnWindowFocus?` | `boolean` \| `"always"` \| ((`query`) => `boolean` \| `"always"`) | If set to `true`, the query will refetch on window focus if the data is stale. If set to `false`, the query will not refetch on window focus. If set to `'always'`, the query will always refetch on window focus. If set to a function, the function will be executed with the latest data and query to compute the value. Defaults to `true`. | `Omit.refetchOnWindowFocus` |
| <a id="retry"></a> `retry?` | `RetryValue`\<`Error`\> | If `false`, failed queries will not retry by default. If `true`, failed queries will retry infinitely., failureCount: num If set to an integer number, e.g. 3, failed queries will retry until the failed query count meets that number. If set to a function `(failureCount, error) => boolean` failed queries will retry until the function returns false. | `Omit.retry` |
| <a id="retrydelay"></a> `retryDelay?` | `RetryDelayValue`\<`Error`\> | - | `Omit.retryDelay` |
| <a id="retryonmount"></a> `retryOnMount?` | `boolean` | If set to `false`, the query will not be retried on mount if it contains an error. Defaults to `true`. | `Omit.retryOnMount` |
| <a id="select"></a> `select?` | (`data`) => `T`[] | This option can be used to transform or select a part of the data returned by the query function. | `Omit.select` |
| <a id="staletime"></a> `staleTime?` | `StaleTimeFunction`\<`T`[], `Error`, `T`[], readonly `unknown`[]\> | The time in milliseconds after data is considered stale. If set to `Infinity`, the data will never be considered stale. If set to a function, the function will be executed with the query to compute a `staleTime`. Defaults to `0`. | `Omit.staleTime` |
| <a id="structuralsharing"></a> `structuralSharing?` | `boolean` \| ((`oldData`, `newData`) => `unknown`) | Set this to `false` to disable structural sharing between query results. Set this to a function which accepts the old and new data and returns resolved data of the same type to implement custom structural sharing logic. Defaults to `true`. | `Omit.structuralSharing` |
| <a id="suspense"></a> `suspense?` | `boolean` | If set to `true`, the query will suspend when `status === 'pending'` and throw errors when `status === 'error'`. Defaults to `false`. | `Omit.suspense` |
| <a id="throwonerror"></a> `throwOnError?` | `ThrowOnError`\<`T`[], `Error`, `T`[], readonly `unknown`[]\> | Whether errors should be thrown instead of setting the `error` property. If set to `true` or `suspense` is `true`, all errors will be thrown to the error boundary. If set to `false` and `suspense` is `false`, errors are returned as state. If set to a function, it will be passed the error and the query, and it should return a boolean indicating whether to show the error in an error boundary (`true`) or return the error as state (`false`). Defaults to `false`. | `Omit.throwOnError` |

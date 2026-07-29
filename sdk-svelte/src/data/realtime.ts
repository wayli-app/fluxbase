/**
 * Realtime stores for the Fluxbase Svelte SDK.
 *
 * These wrap the core SDK's realtime channels in Svelte `writable` stores and
 * manage subscription lifecycle via `onMount`/`onDestroy`. Components read the
 * latest payload with `$store` and automatically (un)subscribe.
 */

import { writable, type Readable } from "svelte/store";
import { onMount, onDestroy } from "svelte";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import type { StoreDeps } from "../auth/store";

/**
 * Subscribe to a realtime channel event. Returns a readable store of payloads.
 * Must be called during component initialization (uses onMount/onDestroy).
 *
 * @example
 * ```svelte
 * <script lang="ts">
 *   import { realtimeChannel } from '@nimbleflux/fluxbase-sdk-svelte'
 *   const inserts = realtimeChannel('table:public.products', 'INSERT')
 * </script>
 * New rows: {JSON.stringify($inserts)}
 * ```
 */
export function createRealtimeChannel<T = any>(
  { client }: StoreDeps,
  channelName: string,
  event: "INSERT" | "UPDATE" | "DELETE" | "*",
): Readable<T | null> {
  const store = writable<T | null>(null);
  let channel: ReturnType<FluxbaseClient["channel"]> | null = null;

  onMount(() => {
    channel = client
      .channel(channelName)
      .on(event, (payload: any) => {
        store.set(payload as T);
      })
      .subscribe();
  });

  onDestroy(() => {
    channel?.unsubscribe();
  });

  return { subscribe: store.subscribe };
}

/**
 * Convenience: subscribe to inserts on a table channel.
 */
export function createTableInserts<T = any>(
  deps: StoreDeps,
  tableName: string,
): Readable<T | null> {
  return createRealtimeChannel<T>(deps, `table:${tableName}`, "INSERT");
}

/**
 * Convenience: subscribe to updates on a table channel.
 */
export function createTableUpdates<T = any>(
  deps: StoreDeps,
  tableName: string,
): Readable<T | null> {
  return createRealtimeChannel<T>(deps, `table:${tableName}`, "UPDATE");
}

/**
 * Convenience: subscribe to deletes on a table channel.
 */
export function createTableDeletes<T = any>(
  deps: StoreDeps,
  tableName: string,
): Readable<T | null> {
  return createRealtimeChannel<T>(deps, `table:${tableName}`, "DELETE");
}

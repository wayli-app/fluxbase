<!--
  Test harness for exercising the Svelte SDK's store factories.

  `@tanstack/svelte-query` v6 is built on Svelte 5 runes (`$effect.pre`,
  `$derived`, `$state`), so `createQuery` / `createMutation` must run inside a
  component initialization context — calling them as bare functions throws
  `effect_orphan`. This component runs a factory during init and hands the
  result back via `capture`, so tests can drive it with @testing-library/svelte.
-->
<script lang="ts">
  // `capture` is captured (not declared) so it stays out of props typing
  // while still being accessible from the test.
  export let capture: (value: any) => void;
  export let factory: () => any;
  const result = factory();
  capture(result);
</script>

import { defineConfig } from 'tsup'

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['esm', 'cjs'],
  dts: true,
  splitting: false,
  sourcemap: true,
  clean: true,
  external: ['svelte', '@sveltejs/kit', '@tanstack/svelte-query', '@nimbleflux/fluxbase-sdk'],
})

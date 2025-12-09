/**
 * Shared bundling module for edge functions and jobs
 * Provides client-side bundling using esbuild for Deno runtime compatibility
 */

// Optional esbuild import - will be dynamically loaded if available
let esbuild: typeof import("esbuild") | null = null;

// Optional fs import for reading deno.json
let fs: typeof import("fs") | null = null;

/**
 * Try to load esbuild for client-side bundling
 * Returns true if esbuild is available, false otherwise
 */
export async function loadEsbuild(): Promise<boolean> {
  if (esbuild) return true;
  try {
    esbuild = await import("esbuild");
    return true;
  } catch {
    return false;
  }
}

/**
 * Get the loaded esbuild instance
 * @internal
 */
export function getEsbuild(): typeof import("esbuild") | null {
  return esbuild;
}

/**
 * Try to load fs module
 */
async function loadFs(): Promise<boolean> {
  if (fs) return true;
  try {
    fs = await import("fs");
    return true;
  } catch {
    return false;
  }
}

/**
 * esbuild plugin that marks Deno-specific imports as external
 * Use this when bundling functions/jobs with esbuild to handle npm:, https://, and jsr: imports
 *
 * @example
 * ```typescript
 * import { denoExternalPlugin } from '@fluxbase/sdk'
 * import * as esbuild from 'esbuild'
 *
 * const result = await esbuild.build({
 *   entryPoints: ['./my-function.ts'],
 *   bundle: true,
 *   plugins: [denoExternalPlugin],
 *   // ... other options
 * })
 * ```
 */
export const denoExternalPlugin = {
  name: "deno-external",
  setup(build: {
    onResolve: (
      opts: { filter: RegExp },
      cb: (args: { path: string }) => { path: string; external: boolean },
    ) => void;
  }) {
    // Mark npm: imports as external - Deno will resolve them at runtime
    build.onResolve({ filter: /^npm:/ }, (args) => ({
      path: args.path,
      external: true,
    }));

    // Mark https:// and http:// imports as external
    build.onResolve({ filter: /^https?:\/\// }, (args) => ({
      path: args.path,
      external: true,
    }));

    // Mark jsr: imports as external (Deno's JSR registry)
    build.onResolve({ filter: /^jsr:/ }, (args) => ({
      path: args.path,
      external: true,
    }));
  },
};

/**
 * Load import map from a deno.json file
 *
 * @param denoJsonPath - Path to deno.json file
 * @returns Import map object or null if not found
 *
 * @example
 * ```typescript
 * const importMap = await loadImportMap('./deno.json')
 * const bundled = await bundleCode({
 *   code: myCode,
 *   importMap,
 * })
 * ```
 */
export async function loadImportMap(
  denoJsonPath: string,
): Promise<Record<string, string> | null> {
  const hasFs = await loadFs();
  if (!hasFs || !fs) {
    console.warn("fs module not available, cannot load import map");
    return null;
  }

  try {
    const content = fs.readFileSync(denoJsonPath, "utf-8");
    const config = JSON.parse(content);
    return config.imports || null;
  } catch (error) {
    console.warn(`Failed to load import map from ${denoJsonPath}:`, error);
    return null;
  }
}

/**
 * Options for bundling code
 */
export interface BundleOptions {
  /** Entry point code */
  code: string;
  /** External modules to exclude from bundle */
  external?: string[];
  /** Source map generation */
  sourcemap?: boolean;
  /** Minify output */
  minify?: boolean;
  /** Import map from deno.json (maps aliases to npm: or file paths) */
  importMap?: Record<string, string>;
  /** Base directory for resolving relative imports (resolveDir in esbuild) */
  baseDir?: string;
  /** Additional paths to search for node_modules (useful when importing from parent directories) */
  nodePaths?: string[];
  /** Custom define values for esbuild (e.g., { 'process.env.NODE_ENV': '"production"' }) */
  define?: Record<string, string>;
}

/**
 * Result of bundling code
 */
export interface BundleResult {
  /** Bundled code */
  code: string;
  /** Source map (if enabled) */
  sourceMap?: string;
}

/**
 * Bundle code using esbuild (client-side)
 *
 * Transforms and bundles TypeScript/JavaScript code into a single file
 * that can be executed by the Fluxbase Deno runtime.
 *
 * Requires esbuild as a peer dependency.
 *
 * @param options - Bundle options including source code
 * @returns Promise resolving to bundled code
 * @throws Error if esbuild is not available
 *
 * @example
 * ```typescript
 * import { bundleCode } from '@fluxbase/sdk'
 *
 * const bundled = await bundleCode({
 *   code: `
 *     import { helper } from './utils'
 *     export default async function handler(req) {
 *       return helper(req.payload)
 *     }
 *   `,
 *   minify: true,
 * })
 *
 * // Use bundled code in sync
 * await client.admin.functions.sync({
 *   namespace: 'default',
 *   functions: [{
 *     name: 'my-function',
 *     code: bundled.code,
 *     is_pre_bundled: true,
 *   }]
 * })
 * ```
 */
export async function bundleCode(options: BundleOptions): Promise<BundleResult> {
  const hasEsbuild = await loadEsbuild();
  if (!hasEsbuild || !esbuild) {
    throw new Error(
      "esbuild is required for bundling. Install it with: npm install esbuild",
    );
  }

  // Process import map to extract externals and aliases
  const externals = [...(options.external ?? [])];
  const alias: Record<string, string> = {};

  if (options.importMap) {
    for (const [key, value] of Object.entries(options.importMap)) {
      // npm: imports should be marked as external - Deno will resolve them at runtime
      if (value.startsWith("npm:")) {
        // Add the import key as external (e.g., "@streamparser/json")
        externals.push(key);
      } else if (
        value.startsWith("https://") ||
        value.startsWith("http://")
      ) {
        // URL imports should also be external - Deno will fetch them at runtime
        externals.push(key);
      } else if (
        value.startsWith("/") ||
        value.startsWith("./") ||
        value.startsWith("../")
      ) {
        // Local file paths - create alias for esbuild
        alias[key] = value;
      } else {
        // Other imports (bare specifiers) - mark as external
        externals.push(key);
      }
    }
  }

  // Create a plugin to handle Deno-specific imports (npm:, https://, http://)
  const denoPlugin: import("esbuild").Plugin = {
    name: "deno-external",
    setup(build) {
      // Mark npm: imports as external
      build.onResolve({ filter: /^npm:/ }, (args) => ({
        path: args.path,
        external: true,
      }));

      // Mark https:// and http:// imports as external
      build.onResolve({ filter: /^https?:\/\// }, (args) => ({
        path: args.path,
        external: true,
      }));

      // Mark jsr: imports as external (Deno's JSR registry)
      build.onResolve({ filter: /^jsr:/ }, (args) => ({
        path: args.path,
        external: true,
      }));
    },
  };

  const resolveDir = options.baseDir || process.cwd?.() || "/";

  const buildOptions: import("esbuild").BuildOptions = {
    stdin: {
      contents: options.code,
      loader: "ts",
      resolveDir,
    },
    // Set absWorkingDir for consistent path resolution
    absWorkingDir: resolveDir,
    bundle: true,
    write: false,
    format: "esm",
    // Use 'node' platform for better node_modules resolution (Deno supports Node APIs)
    platform: "node",
    target: "esnext",
    minify: options.minify ?? false,
    sourcemap: options.sourcemap ? "inline" : false,
    external: externals,
    plugins: [denoPlugin],
    // Preserve handler export
    treeShaking: true,
    // Resolve .ts, .js, .mjs extensions
    resolveExtensions: [".ts", ".tsx", ".js", ".mjs", ".json"],
    // ESM conditions for better module resolution
    conditions: ["import", "module"],
  };

  // Add alias if we have any
  if (Object.keys(alias).length > 0) {
    buildOptions.alias = alias;
  }

  // Add nodePaths for resolving modules from additional directories
  if (options.nodePaths && options.nodePaths.length > 0) {
    buildOptions.nodePaths = options.nodePaths;
  }

  // Add custom define values
  if (options.define) {
    buildOptions.define = options.define;
  }

  const result = await esbuild.build(buildOptions);

  const output = result.outputFiles?.[0];
  if (!output) {
    throw new Error("Bundling failed: no output generated");
  }

  return {
    code: output.text,
    sourceMap: options.sourcemap ? output.text : undefined,
  };
}

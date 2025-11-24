# Edge Functions Examples

This directory contains example edge functions demonstrating various features of Fluxbase.

## Examples

### 1. Simple Function with Shared Modules

**File:** `api-with-shared-modules.ts`

Demonstrates:
- Importing shared modules from `_shared/cors.ts` and `_shared/validation.ts`
- CORS preflight handling
- Input validation using shared utilities
- Error handling with CORS headers

### 2. Multi-File Function

**Directory:** `multi-file-function/`

Demonstrates:
- Directory-based function structure
- Multiple TypeScript files in one function
- Local imports (`./types.ts`, `./helpers.ts`)
- Shared module imports (`_shared/cors.ts`)
- Type-safe webhook processing

**Files:**
- `index.ts` - Main handler function (entry point)
- `types.ts` - TypeScript type definitions
- `helpers.ts` - Processing logic and utilities

## Shared Modules

**Directory:** `_shared/`

Common utilities shared across all functions:

- `_shared/cors.ts` - CORS utilities for handling cross-origin requests
- `_shared/validation.ts` - Input validation utilities

## File-Based Deployment

To use these examples with file-based deployment:

1. Mount the functions directory in docker-compose.yml
2. Start Fluxbase - functions are automatically loaded
3. Reload after changes using the admin API endpoint

See the main documentation for detailed deployment instructions.

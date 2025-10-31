import { z } from 'zod'

// Provider types based on how users authenticate
const providerSchema = z.enum([
  'email',
  'invite_pending',
  'magic_link',
  'google',
  'github',
  'microsoft',
  'apple',
  'facebook',
  'twitter',
  'linkedin',
  'gitlab',
  'bitbucket',
])
export type Provider = z.infer<typeof providerSchema>

// User schema matching the backend EnrichedUser struct
const userSchema = z.object({
  id: z.string(),
  email: z.string(),
  email_verified: z.boolean(),
  role: z.string(),
  provider: providerSchema,
  active_sessions: z.number(),
  last_sign_in: z.coerce.date().nullable(),
  metadata: z.record(z.string(), z.unknown()).nullable(),
  created_at: z.coerce.date(),
  updated_at: z.coerce.date(),
})
export type User = z.infer<typeof userSchema>

export const userListSchema = z.array(userSchema)

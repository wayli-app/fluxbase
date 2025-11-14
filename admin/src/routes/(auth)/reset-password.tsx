import z from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { ResetPassword } from '@/features/auth/reset-password'

const resetPasswordSearchSchema = z.object({
  token: z.string(),
})

export const Route = createFileRoute('/(auth)/reset-password')({
  validateSearch: resetPasswordSearchSchema,
  component: ResetPassword,
})

import { Shield, UserCheck } from 'lucide-react'
import { type Provider } from './schema'

// Provider badge colors
export const providerColors = new Map<Provider, string>([
  ['email', 'bg-teal-100/30 text-teal-900 dark:text-teal-200 border-teal-200'],
  ['invite_pending', 'bg-amber-200/40 text-amber-900 dark:text-amber-100 border-amber-300'],
  ['magic_link', 'bg-purple-200/40 text-purple-900 dark:text-purple-100 border-purple-300'],
  ['google', 'bg-red-100/30 text-red-900 dark:text-red-200 border-red-200'],
  ['github', 'bg-gray-100/30 text-gray-900 dark:text-gray-200 border-gray-200'],
  ['microsoft', 'bg-blue-100/30 text-blue-900 dark:text-blue-200 border-blue-200'],
  ['apple', 'bg-slate-100/30 text-slate-900 dark:text-slate-200 border-slate-200'],
  ['facebook', 'bg-blue-100/30 text-blue-900 dark:text-blue-200 border-blue-200'],
  ['twitter', 'bg-sky-100/30 text-sky-900 dark:text-sky-200 border-sky-200'],
  ['linkedin', 'bg-blue-100/30 text-blue-900 dark:text-blue-200 border-blue-200'],
  ['gitlab', 'bg-orange-100/30 text-orange-900 dark:text-orange-200 border-orange-200'],
  ['bitbucket', 'bg-blue-100/30 text-blue-900 dark:text-blue-200 border-blue-200'],
])

// Common role configurations
export const roles = [
  {
    label: 'Admin',
    value: 'admin',
    icon: Shield,
  },
  {
    label: 'User',
    value: 'user',
    icon: UserCheck,
  },
] as const

// Provider display names
export const providerLabels: Record<Provider, string> = {
  email: 'Email',
  invite_pending: 'Invite Pending',
  magic_link: 'Magic Link',
  google: 'Google',
  github: 'GitHub',
  microsoft: 'Microsoft',
  apple: 'Apple',
  facebook: 'Facebook',
  twitter: 'Twitter',
  linkedin: 'LinkedIn',
  gitlab: 'GitLab',
  bitbucket: 'Bitbucket',
}

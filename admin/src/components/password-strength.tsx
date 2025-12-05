import { useMemo } from 'react'
import { Check, X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface PasswordRequirement {
  label: string
  test: (password: string) => boolean
}

const requirements: PasswordRequirement[] = [
  { label: 'At least 8 characters', test: (p) => p.length >= 8 },
  { label: 'Contains lowercase letter', test: (p) => /[a-z]/.test(p) },
  { label: 'Contains uppercase letter', test: (p) => /[A-Z]/.test(p) },
  { label: 'Contains number', test: (p) => /\d/.test(p) },
]

function getStrength(password: string): number {
  if (!password) return 0
  return requirements.filter((req) => req.test(password)).length
}

function getStrengthLabel(strength: number): string {
  if (strength === 0) return ''
  if (strength === 1) return 'Weak'
  if (strength === 2) return 'Fair'
  if (strength === 3) return 'Good'
  return 'Strong'
}

function getStrengthColor(strength: number): string {
  if (strength === 0) return 'bg-muted'
  if (strength === 1) return 'bg-red-500'
  if (strength === 2) return 'bg-orange-500'
  if (strength === 3) return 'bg-yellow-500'
  return 'bg-green-500'
}

interface PasswordStrengthProps {
  password: string
  showRequirements?: boolean
  className?: string
}

export function PasswordStrength({
  password,
  showRequirements = true,
  className,
}: PasswordStrengthProps) {
  const strength = useMemo(() => getStrength(password), [password])
  const strengthLabel = useMemo(() => getStrengthLabel(strength), [strength])

  return (
    <div className={cn('space-y-2', className)}>
      {/* Strength bar */}
      <div className='space-y-1'>
        <div className='flex gap-1'>
          {[1, 2, 3, 4].map((level) => (
            <div
              key={level}
              className={cn(
                'h-1.5 flex-1 rounded-full transition-colors',
                strength >= level ? getStrengthColor(strength) : 'bg-muted'
              )}
            />
          ))}
        </div>
        {password && strengthLabel && (
          <p
            className={cn(
              'text-xs font-medium',
              strength <= 1 && 'text-red-500',
              strength === 2 && 'text-orange-500',
              strength === 3 && 'text-yellow-600 dark:text-yellow-500',
              strength === 4 && 'text-green-500'
            )}
          >
            {strengthLabel}
          </p>
        )}
      </div>

      {/* Requirements checklist */}
      {showRequirements && password && (
        <ul className='space-y-1'>
          {requirements.map((req) => {
            const met = req.test(password)
            return (
              <li
                key={req.label}
                className={cn(
                  'flex items-center gap-2 text-xs transition-colors',
                  met ? 'text-green-600 dark:text-green-500' : 'text-muted-foreground'
                )}
              >
                {met ? (
                  <Check className='h-3 w-3' />
                ) : (
                  <X className='h-3 w-3' />
                )}
                {req.label}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { SettingOverride } from './overridable-switch'
import { OverrideBadge } from './override-badge'

interface OverridableInputProps {
  id: string
  label: string
  description?: string
  type?: string
  value: string | number
  onChange: (value: string) => void
  override?: SettingOverride
  disabled?: boolean
  placeholder?: string
}

export function OverridableInput({
  id,
  label,
  description,
  type = 'text',
  value,
  onChange,
  override,
  disabled,
  placeholder,
}: OverridableInputProps) {
  const isOverridden = override?.is_overridden || false
  const isDisabled = disabled || isOverridden

  return (
    <div className='space-y-2'>
      <div className='flex items-center gap-2'>
        <Label htmlFor={id}>{label}</Label>
        {isOverridden && <OverrideBadge envVar={override?.env_var} />}
      </div>
      {description && (
        <p className='text-muted-foreground text-sm'>{description}</p>
      )}
      <Input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={isDisabled}
        placeholder={placeholder}
      />
      {isOverridden && (
        <p className='text-muted-foreground text-xs'>
          Controlled by{' '}
          <code className='bg-muted rounded px-1 py-0.5'>
            {override?.env_var}
          </code>
        </p>
      )}
    </div>
  )
}

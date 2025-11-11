import type { LucideIcon } from 'lucide-react'

type ContentSectionProps = {
  title: string
  desc: string
  children: React.JSX.Element
  icon?: LucideIcon
}

export function ContentSection({
  title,
  desc,
  children,
  icon: Icon,
}: ContentSectionProps) {
  return (
    <div className='flex flex-col gap-6 p-6'>
      <div>
        <h1 className='flex items-center gap-2 text-3xl font-bold tracking-tight'>
          {Icon && <Icon className='h-8 w-8' />}
          {title}
        </h1>
        <p className='text-muted-foreground mt-2'>{desc}</p>
      </div>
      <div className='lg:max-w-xl'>{children}</div>
    </div>
  )
}

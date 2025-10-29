import { LucideIcon } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface ComingSoonProps {
  icon: LucideIcon
  title: string
  description: string
  features?: string[]
}

export function ComingSoon({ icon: Icon, title, description, features }: ComingSoonProps) {
  return (
    <div className='flex h-full items-center justify-center p-6'>
      <Card className='w-full max-w-2xl'>
        <CardHeader className='text-center'>
          <div className='mx-auto mb-4 flex size-16 items-center justify-center rounded-full bg-primary/10'>
            <Icon className='text-primary size-8' />
          </div>
          <CardTitle className='text-2xl'>{title}</CardTitle>
          <CardDescription className='text-base'>{description}</CardDescription>
        </CardHeader>
        {features && features.length > 0 && (
          <CardContent>
            <div className='space-y-2'>
              <p className='text-muted-foreground text-sm font-medium'>Planned Features:</p>
              <ul className='text-muted-foreground space-y-1 text-sm'>
                {features.map((feature, index) => (
                  <li key={index} className='flex items-start gap-2'>
                    <span className='mt-1.5 size-1.5 rounded-full bg-current' />
                    {feature}
                  </li>
                ))}
              </ul>
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  )
}

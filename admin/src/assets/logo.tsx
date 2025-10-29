import { type SVGProps } from 'react'
import { cn } from '@/lib/utils'

export function Logo({ className, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      id='fluxbase-logo'
      viewBox='0 0 24 24'
      xmlns='http://www.w3.org/2000/svg'
      height='24'
      width='24'
      fill='none'
      stroke='currentColor'
      strokeWidth='2'
      strokeLinecap='round'
      strokeLinejoin='round'
      className={cn('size-6', className)}
      {...props}
    >
      <title>Fluxbase</title>
      {/* Database/server icon with flow lines */}
      <ellipse cx='12' cy='5' rx='9' ry='3' />
      <path d='M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5' />
      <path d='M3 12c0 1.66 4 3 9 3s9-1.34 9-3' />
    </svg>
  )
}

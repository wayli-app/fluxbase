import { LayoutDashboard } from 'lucide-react'
import { Main } from '@/components/layout/main'
import { HealthIndicators } from './components/health-indicators'
import { MetricsCards } from './components/metrics-cards'
import { FluxbaseStats } from './components/fluxbase-stats'
import { SecuritySummary } from './components/security-summary'

export function Dashboard() {
  return (
    <Main>
      <div className='bg-background flex items-center justify-between border-b px-6 py-4'>
        <div className='flex items-center gap-3'>
          <div className='bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg'>
            <LayoutDashboard className='text-primary h-5 w-5' />
          </div>
          <div>
            <h1 className='text-xl font-semibold'>Dashboard</h1>
            <p className='text-muted-foreground text-sm'>
              Monitor your Backend as a Service
            </p>
          </div>
        </div>
      </div>

      <div className='space-y-6 p-6'>
        {/* Health Indicators */}
        <HealthIndicators />

        {/* Metrics Cards with Trends */}
        <MetricsCards />

        {/* Legacy Stats - keeping for reference */}
        <FluxbaseStats />

        {/* Security Summary */}
        <SecuritySummary />
      </div>
    </Main>
  )
}

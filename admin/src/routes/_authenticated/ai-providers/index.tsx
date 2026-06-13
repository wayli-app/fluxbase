import { createFileRoute } from '@tanstack/react-router'
import { Bot } from 'lucide-react'
import { AIProvidersTab } from '@/components/ai-providers/ai-providers-tab'
import { PageHeader } from '@/components/layout/page-header'

const AIProvidersPage = () => {
  return (
    <div className='flex h-full flex-col'>
      <PageHeader
        icon={<Bot />}
        title="AI Providers"
        description="Configure AI providers for chatbots and intelligent features"
      />

      <div className='flex-1 overflow-auto p-6'>
        <AIProvidersTab />
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/ai-providers/')({
  component: AIProvidersPage,
})

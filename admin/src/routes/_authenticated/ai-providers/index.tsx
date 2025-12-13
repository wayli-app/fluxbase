import { createFileRoute } from '@tanstack/react-router'
import { Bot } from 'lucide-react'
import { AIProvidersTab } from '@/components/ai-providers/ai-providers-tab'

export const Route = createFileRoute('/_authenticated/ai-providers/')({
  component: AIProvidersPage,
})

function AIProvidersPage() {
  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight flex items-center gap-2'>
          <Bot className='h-8 w-8' />
          AI Providers
        </h1>
        <p className='text-sm text-muted-foreground mt-2'>Configure AI providers for chatbots and intelligent features</p>
      </div>

      <AIProvidersTab />
    </div>
  )
}

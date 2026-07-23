import { createFileRoute } from '@tanstack/react-router'
import { AIProvidersTab } from '@/components/ai-providers/ai-providers-tab'

const AIProvidersPage = () => {
  return (
    <div className='flex h-full flex-col'>
      <div className='flex-1 overflow-auto p-6'>
        <AIProvidersTab />
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/ai-providers/')({
  component: AIProvidersPage,
})

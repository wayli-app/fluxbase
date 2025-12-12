import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  chatbotsApi,
  type AIChatbot,
  type AIChatbotSummary,
  type AIProvider,
} from '@/lib/api'
import { getAccessToken } from '@/lib/auth'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

interface ChatbotSettingsFormProps {
  chatbot: AIChatbot
  providers: AIProvider[]
  onSuccess: () => void
  onCancel: () => void
}

function ChatbotSettingsForm({
  chatbot,
  providers,
  onSuccess,
  onCancel,
}: ChatbotSettingsFormProps) {
  const queryClient = useQueryClient()

  // Form state - initialized directly from props
  const [description, setDescription] = useState(chatbot.description || '')
  const [enabled, setEnabled] = useState(chatbot.enabled)
  const [maxTokens, setMaxTokens] = useState(chatbot.max_tokens)
  const [temperature, setTemperature] = useState(chatbot.temperature)
  const [providerId, setProviderId] = useState<string>(
    chatbot.provider_id || '__default__'
  )
  const [persistConversations, setPersistConversations] = useState(
    chatbot.persist_conversations
  )
  const [conversationTTLHours, setConversationTTLHours] = useState(
    chatbot.conversation_ttl_hours
  )
  const [maxConversationTurns, setMaxConversationTurns] = useState(
    chatbot.max_conversation_turns
  )
  const [rateLimitPerMinute, setRateLimitPerMinute] = useState(
    chatbot.rate_limit_per_minute
  )
  const [dailyRequestLimit, setDailyRequestLimit] = useState(
    chatbot.daily_request_limit
  )
  const [dailyTokenBudget, setDailyTokenBudget] = useState(
    chatbot.daily_token_budget
  )
  const [allowUnauthenticated, setAllowUnauthenticated] = useState(
    chatbot.allow_unauthenticated
  )
  const [isPublic, setIsPublic] = useState(chatbot.is_public)

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: (data: Partial<AIChatbot>) =>
      chatbotsApi.update(chatbot.id, data),
    onSuccess: () => {
      toast.success('Chatbot settings updated successfully')
      queryClient.invalidateQueries({ queryKey: ['chatbot', chatbot.id] })
      queryClient.invalidateQueries({ queryKey: ['chatbots'] })
      onSuccess()
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || 'Failed to update chatbot'
          : 'Failed to update chatbot'
      toast.error(errorMessage)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Validate inputs
    if (temperature < 0 || temperature > 2) {
      toast.error('Temperature must be between 0 and 2')
      return
    }
    if (maxTokens <= 0) {
      toast.error('Max tokens must be positive')
      return
    }

    updateMutation.mutate({
      description,
      enabled,
      max_tokens: maxTokens,
      temperature,
      provider_id: providerId === '__default__' ? '' : providerId,
      persist_conversations: persistConversations,
      conversation_ttl_hours: conversationTTLHours,
      max_conversation_turns: maxConversationTurns,
      rate_limit_per_minute: rateLimitPerMinute,
      daily_request_limit: dailyRequestLimit,
      daily_token_budget: dailyTokenBudget,
      allow_unauthenticated: allowUnauthenticated,
      is_public: isPublic,
    })
  }

  return (
    <form onSubmit={handleSubmit} className='space-y-4'>
      {/* Description */}
      <div className='space-y-2'>
        <Label htmlFor='description'>Description</Label>
        <Input
          id='description'
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder='Brief description of this chatbot'
        />
      </div>

      {/* Enabled */}
      <div className='flex items-center justify-between'>
        <Label htmlFor='enabled'>Enabled</Label>
        <Switch id='enabled' checked={enabled} onCheckedChange={setEnabled} />
      </div>

      {/* AI Model Settings */}
      <div className='space-y-4 border-t pt-4'>
        <h3 className='font-medium'>AI Model Settings</h3>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='maxTokens'>Max Tokens</Label>
            <Input
              id='maxTokens'
              type='number'
              min='1'
              value={maxTokens}
              onChange={(e) => setMaxTokens(parseInt(e.target.value))}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='temperature'>Temperature (0-2)</Label>
            <Input
              id='temperature'
              type='number'
              min='0'
              max='2'
              step='0.1'
              value={temperature}
              onChange={(e) => setTemperature(parseFloat(e.target.value))}
            />
          </div>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='provider'>AI Provider</Label>
          <Select value={providerId} onValueChange={setProviderId}>
            <SelectTrigger id='provider'>
              <SelectValue placeholder='Use default provider' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='__default__'>Default Provider</SelectItem>
              {providers.map((provider) => (
                <SelectItem key={provider.id} value={provider.id}>
                  <div className='flex items-center gap-2'>
                    <span>{provider.display_name}</span>
                    {provider.from_config && (
                      <Badge variant='secondary' className='text-xs'>
                        Config
                      </Badge>
                    )}
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Conversation Settings */}
      <div className='space-y-4 border-t pt-4'>
        <h3 className='font-medium'>Conversation Settings</h3>

        <div className='flex items-center justify-between'>
          <Label htmlFor='persistConversations'>Persist Conversations</Label>
          <Switch
            id='persistConversations'
            checked={persistConversations}
            onCheckedChange={setPersistConversations}
          />
        </div>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='conversationTTLHours'>
              Conversation TTL (hours)
            </Label>
            <Input
              id='conversationTTLHours'
              type='number'
              min='1'
              value={conversationTTLHours}
              onChange={(e) =>
                setConversationTTLHours(parseInt(e.target.value))
              }
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='maxConversationTurns'>Max Conversation Turns</Label>
            <Input
              id='maxConversationTurns'
              type='number'
              min='1'
              value={maxConversationTurns}
              onChange={(e) =>
                setMaxConversationTurns(parseInt(e.target.value))
              }
            />
          </div>
        </div>
      </div>

      {/* Rate Limiting */}
      <div className='space-y-4 border-t pt-4'>
        <h3 className='font-medium'>Rate Limiting</h3>

        <div className='space-y-4'>
          <div className='space-y-2'>
            <Label htmlFor='rateLimitPerMinute'>Rate Limit (per minute)</Label>
            <Input
              id='rateLimitPerMinute'
              type='number'
              min='1'
              value={rateLimitPerMinute}
              onChange={(e) => setRateLimitPerMinute(parseInt(e.target.value))}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='dailyRequestLimit'>Daily Request Limit</Label>
            <Input
              id='dailyRequestLimit'
              type='number'
              min='1'
              value={dailyRequestLimit}
              onChange={(e) => setDailyRequestLimit(parseInt(e.target.value))}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='dailyTokenBudget'>Daily Token Budget</Label>
            <Input
              id='dailyTokenBudget'
              type='number'
              min='1'
              value={dailyTokenBudget}
              onChange={(e) => setDailyTokenBudget(parseInt(e.target.value))}
            />
          </div>
        </div>
      </div>

      {/* Access Control */}
      <div className='space-y-4 border-t pt-4'>
        <h3 className='font-medium'>Access Control</h3>

        <div className='flex items-center justify-between'>
          <Label htmlFor='allowUnauthenticated'>Allow Unauthenticated</Label>
          <Switch
            id='allowUnauthenticated'
            checked={allowUnauthenticated}
            onCheckedChange={setAllowUnauthenticated}
          />
        </div>

        <div className='flex items-center justify-between'>
          <Label htmlFor='isPublic'>Public</Label>
          <Switch
            id='isPublic'
            checked={isPublic}
            onCheckedChange={setIsPublic}
          />
        </div>
      </div>

      <DialogFooter>
        <Button type='button' variant='outline' onClick={onCancel}>
          Cancel
        </Button>
        <Button type='submit' disabled={updateMutation.isPending}>
          {updateMutation.isPending ? (
            <>
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              Saving...
            </>
          ) : (
            'Save Changes'
          )}
        </Button>
      </DialogFooter>
    </form>
  )
}

interface ChatbotSettingsDialogProps {
  chatbot: AIChatbotSummary
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChatbotSettingsDialog({
  chatbot,
  open,
  onOpenChange,
}: ChatbotSettingsDialogProps) {
  // Fetch full chatbot details
  const { data: fullChatbot, isLoading } = useQuery({
    queryKey: ['chatbot', chatbot.id],
    queryFn: () => chatbotsApi.get(chatbot.id),
    enabled: open,
  })

  // Fetch available providers
  const { data: providersData } = useQuery<{ providers: AIProvider[] }>({
    queryKey: ['ai-providers'],
    queryFn: async () => {
      const response = await fetch('/api/v1/admin/ai/providers', {
        headers: {
          Authorization: `Bearer ${getAccessToken()}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch providers')
      return response.json()
    },
    enabled: open,
  })

  const providers: AIProvider[] = providersData?.providers || []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[90vh] max-w-2xl overflow-y-auto'>
        <DialogHeader>
          <DialogTitle>Chatbot Settings</DialogTitle>
          <DialogDescription>
            Configure settings for <strong>{chatbot.name}</strong>
          </DialogDescription>
        </DialogHeader>

        {isLoading || !fullChatbot ? (
          <div className='flex items-center justify-center p-8'>
            <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
          </div>
        ) : (
          <ChatbotSettingsForm
            key={fullChatbot.id}
            chatbot={fullChatbot}
            providers={providers}
            onSuccess={() => onOpenChange(false)}
            onCancel={() => onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

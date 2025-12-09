import { useState, useEffect, useRef, useCallback, memo } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Bot, Send, Loader2, AlertCircle, ChevronDown, ChevronUp, User, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'
import { FluxbaseAIChat } from '@fluxbase/sdk'
import type { AIChatbotSummary } from '@/lib/api'
import { getAccessToken } from '@/lib/auth'
import { UserSearch } from '@/features/impersonation/components/user-search'

// Helper to get active token (impersonation takes precedence, same pattern as api.ts)
function getActiveToken(): string | null {
  const impersonationToken = localStorage.getItem('fluxbase_impersonation_token')
  return impersonationToken || getAccessToken()
}

interface ChatbotTestDialogProps {
  chatbot: AIChatbotSummary
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface QueryResultMetadata {
  query: string
  summary: string
  rowCount: number
  data: Record<string, unknown>[]
}

interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
  metadata?: {
    isStreaming?: boolean
    queryResults?: QueryResultMetadata[]
    type?: 'info' | 'error'
  }
}

function getWebSocketUrl(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  return `${protocol}//${host}/ai/ws`
}

function formatTimestamp(date: Date): string {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const QueryResultDisplay = memo(function QueryResultDisplay({ result }: { result: QueryResultMetadata }) {
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(result.query)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('Failed to copy to clipboard')
    }
  }

  return (
    <div className='mt-3 space-y-2 border-t pt-2 min-w-0 w-full'>
      <div>
        <div className='flex items-center justify-between mb-1'>
          <span className='text-xs font-medium'>SQL Query</span>
          {result.data.length > 0 && (
            <Button
              variant='ghost'
              size='sm'
              className='h-6 px-2 text-xs'
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? (
                <>
                  <ChevronUp className='h-3 w-3 mr-1' />
                  Hide Data
                </>
              ) : (
                <>
                  <ChevronDown className='h-3 w-3 mr-1' />
                  Show Data
                </>
              )}
            </Button>
          )}
        </div>
        <div className='relative group'>
          <pre className='text-xs bg-black/10 dark:bg-white/10 p-2 pr-10 rounded overflow-x-auto'>
            <code>{result.query}</code>
          </pre>
          <Button
            variant='ghost'
            size='icon'
            className='absolute top-1 right-1 h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity'
            onClick={handleCopy}
          >
            <Copy className={copied ? 'h-3 w-3 hidden' : 'h-3 w-3'} />
            <Check className={copied ? 'h-3 w-3 text-green-500' : 'h-3 w-3 hidden'} />
          </Button>
        </div>
      </div>

      <div className='text-xs text-muted-foreground'>
        {result.summary} ({result.rowCount} rows)
      </div>

      {expanded && result.data.length > 0 && (
        <div className='max-h-60 overflow-auto rounded border w-full'>
          <Table>
            <TableHeader>
              <TableRow>
                {Object.keys(result.data[0]).map((key) => (
                  <TableHead key={key} className='text-xs py-1 px-2'>
                    {key}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.data.slice(0, 10).map((row, idx) => (
                <TableRow key={idx}>
                  {Object.values(row).map((value, vidx) => (
                    <TableCell key={vidx} className='text-xs py-1 px-2'>
                      {value === null ? (
                        <span className='text-muted-foreground italic'>null</span>
                      ) : (
                        String(value)
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {result.data.length > 10 && (
            <div className='text-xs text-center text-muted-foreground py-2 border-t'>
              Showing 10 of {result.data.length} rows
            </div>
          )}
        </div>
      )}
    </div>
  )
})

const MessageBubble = memo(function MessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user'
  const isSystem = message.role === 'system'
  const isError = message.metadata?.type === 'error'

  return (
    <div className={cn('flex w-full', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[85%] min-w-0 rounded-lg px-4 py-2 overflow-hidden',
          isUser && 'bg-primary text-primary-foreground',
          isSystem && !isError && 'bg-muted text-muted-foreground text-sm italic w-full text-center',
          isSystem && isError && 'bg-destructive/10 text-destructive text-sm w-full',
          !isUser && !isSystem && 'bg-muted'
        )}
      >
        <div className='whitespace-pre-wrap break-words'>
          {message.content}
          {message.metadata?.isStreaming && (
            <span className='inline-block w-1.5 h-4 bg-current animate-pulse ml-0.5' />
          )}
        </div>

        {message.metadata?.queryResults && message.metadata.queryResults.length > 0 && (
          <div className='space-y-3'>
            {message.metadata.queryResults.map((result, idx) => (
              <QueryResultDisplay key={idx} result={result} />
            ))}
          </div>
        )}

        {!isSystem && (
          <div className='mt-1 text-xs opacity-70'>
            {formatTimestamp(message.timestamp)}
          </div>
        )}
      </div>
    </div>
  )
})

function ConnectionLoadingState() {
  return (
    <div className='flex items-center justify-center py-12 flex-1'>
      <div className='text-center space-y-2'>
        <Loader2 className='h-8 w-8 animate-spin mx-auto text-muted-foreground' />
        <p className='text-sm text-muted-foreground'>Connecting to chatbot...</p>
      </div>
    </div>
  )
}

function ConnectionErrorState({ error }: { error: string }) {
  return (
    <div className='flex items-center justify-center py-12 flex-1'>
      <div className='text-center space-y-2'>
        <AlertCircle className='h-8 w-8 mx-auto text-destructive' />
        <p className='text-sm text-muted-foreground'>Failed to connect to chatbot</p>
        <p className='text-xs text-destructive'>{error}</p>
      </div>
    </div>
  )
}

export function ChatbotTestDialog({
  chatbot,
  open,
  onOpenChange,
}: ChatbotTestDialogProps) {
  const [conversationId, setConversationId] = useState<string | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [isThinking, setIsThinking] = useState(false)
  const [currentProgress, setCurrentProgress] = useState<string | null>(null)

  // Impersonation state
  const [impersonateUserId, setImpersonateUserId] = useState<string | null>(null)
  const [impersonateUserEmail, setImpersonateUserEmail] = useState<string | null>(null)

  const chatClientRef = useRef<FluxbaseAIChat | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = useCallback(() => {
    // Use requestAnimationFrame to ensure DOM has updated before scrolling
    requestAnimationFrame(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
    })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, scrollToBottom])

  const addSystemMessage = useCallback((content: string, type: 'info' | 'error' = 'info') => {
    const systemMsg: ChatMessage = {
      id: `system-${Date.now()}`,
      role: 'system',
      content,
      timestamp: new Date(),
      metadata: { type },
    }
    setMessages((prev) => [...prev, systemMsg])
  }, [])

  const handleContentChunk = useCallback((delta: string, _convId: string) => {
    setMessages((prev) => {
      const lastMsg = prev[prev.length - 1]

      if (lastMsg?.role === 'assistant' && lastMsg.metadata?.isStreaming) {
        return prev.map((msg, idx) =>
          idx === prev.length - 1
            ? { ...msg, content: msg.content + delta }
            : msg
        )
      }

      return [
        ...prev,
        {
          id: `msg-${Date.now()}`,
          role: 'assistant' as const,
          content: delta,
          timestamp: new Date(),
          metadata: { isStreaming: true },
        },
      ]
    })
  }, [])

  const handleProgress = useCallback((step: string, message: string, _convId: string) => {
    setCurrentProgress(`${step}: ${message}`)
    setIsThinking(true)
  }, [])

  const handleQueryResult = useCallback(
    (query: string, summary: string, rowCount: number, data: Record<string, unknown>[], _convId: string) => {
      const newResult: QueryResultMetadata = { query, summary, rowCount, data }

      setMessages((prev) => {
        const lastMsg = prev[prev.length - 1]

        // If the last message is an assistant message, append the query result to it
        if (lastMsg?.role === 'assistant') {
          return prev.map((msg, idx) =>
            idx === prev.length - 1
              ? {
                  ...msg,
                  metadata: {
                    ...msg.metadata,
                    queryResults: [...(msg.metadata?.queryResults || []), newResult],
                  },
                }
              : msg
          )
        }

        // If there's no assistant message yet, create one with the query result
        // This happens when the AI uses tool calls without streaming content first
        return [
          ...prev,
          {
            id: `msg-${Date.now()}`,
            role: 'assistant' as const,
            content: summary,
            timestamp: new Date(),
            metadata: {
              isStreaming: false,
              queryResults: [newResult],
            },
          },
        ]
      })
    },
    []
  )

  const handleDone = useCallback((_usage: unknown, _convId: string) => {
    setIsThinking(false)
    setCurrentProgress(null)
    setIsSending(false)

    setMessages((prev) =>
      prev.map((msg, idx) =>
        idx === prev.length - 1 && msg.role === 'assistant'
          ? { ...msg, metadata: { ...msg.metadata, isStreaming: false } }
          : msg
      )
    )
  }, [])

  const handleError = useCallback(
    (error: string, _code: string | undefined, _convId: string | undefined) => {
      setIsThinking(false)
      setCurrentProgress(null)
      setIsSending(false)

      addSystemMessage(`Error: ${error}`, 'error')
      toast.error(`Chatbot error: ${error}`)
    },
    [addSystemMessage]
  )

  useEffect(() => {
    if (!open) return

    let mounted = true

    const initializeConnection = async () => {
      setIsConnecting(true)
      setConnectionError(null)

      try {
        const wsUrl = getWebSocketUrl()
        const token = getActiveToken()

        const chatClient = new FluxbaseAIChat({
          wsUrl,
          token: token || undefined,
          onContent: handleContentChunk,
          onProgress: handleProgress,
          onQueryResult: handleQueryResult,
          onDone: handleDone,
          onError: handleError,
          reconnectAttempts: 0,
        })

        await chatClient.connect()

        if (!mounted) {
          chatClient.disconnect()
          return
        }

        chatClientRef.current = chatClient
        setIsConnected(true)

        const convId = await chatClient.startChat(
          chatbot.name,
          chatbot.namespace,
          undefined, // conversationId
          impersonateUserId || undefined
        )

        if (!mounted) {
          chatClient.disconnect()
          return
        }

        setConversationId(convId)
        addSystemMessage(
          impersonateUserEmail
            ? `Connected to ${chatbot.name} (testing as ${impersonateUserEmail})`
            : `Connected to ${chatbot.name}`
        )
      } catch (error) {
        if (mounted) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          setConnectionError(errorMessage)
          toast.error('Failed to connect to chatbot')
        }
      } finally {
        if (mounted) {
          setIsConnecting(false)
        }
      }
    }

    initializeConnection()

    return () => {
      mounted = false
      if (chatClientRef.current) {
        chatClientRef.current.disconnect()
        chatClientRef.current = null
      }
      setIsConnected(false)
      setConversationId(null)
      setMessages([])
      setIsThinking(false)
      setCurrentProgress(null)
      setIsSending(false)
      setConnectionError(null)
    }
  }, [
    open,
    chatbot.name,
    chatbot.namespace,
    impersonateUserId,
    impersonateUserEmail,
    addSystemMessage,
    handleContentChunk,
    handleProgress,
    handleQueryResult,
    handleDone,
    handleError,
  ])

  const handleSendMessage = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()

      if (!inputValue.trim() || !isConnected || !chatClientRef.current || !conversationId) {
        return
      }

      const userMessage = inputValue.trim()
      setInputValue('')
      setIsSending(true)
      setIsThinking(true)

      const userMsg: ChatMessage = {
        id: `msg-${Date.now()}`,
        role: 'user',
        content: userMessage,
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, userMsg])

      try {
        chatClientRef.current.sendMessage(conversationId, userMessage)
      } catch {
        toast.error('Failed to send message')
        setIsSending(false)
        setIsThinking(false)
      }
    },
    [inputValue, isConnected, conversationId]
  )

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side='right' className='w-full sm:max-w-2xl h-full flex flex-col p-0'>
        <SheetHeader className='px-6 py-4 border-b shrink-0'>
          <SheetTitle className='flex items-center gap-2'>
            <Bot className='h-5 w-5' />
            Test {chatbot.name}
          </SheetTitle>
          <SheetDescription>
            {chatbot.model ? (
              <>Model: <span className='font-medium'>{chatbot.model}</span></>
            ) : (
              'Chat with this bot to test its responses'
            )}
          </SheetDescription>
        </SheetHeader>

        {/* User impersonation selector */}
        <div className='px-6 py-3 border-b bg-muted/30 shrink-0'>
          <div className='flex items-center gap-3'>
            <User className='h-4 w-4 text-muted-foreground shrink-0' />
            <div className='flex-1'>
              {impersonateUserId ? (
                <div className='flex items-center gap-2'>
                  <span className='text-sm'>
                    Testing as: <strong>{impersonateUserEmail}</strong>
                  </span>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='h-6 px-2 text-xs'
                    onClick={() => {
                      setImpersonateUserId(null)
                      setImpersonateUserEmail(null)
                    }}
                  >
                    Clear
                  </Button>
                </div>
              ) : (
                <UserSearch
                  value={impersonateUserId || undefined}
                  onSelect={(userId, userEmail) => {
                    setImpersonateUserId(userId)
                    setImpersonateUserEmail(userEmail)
                  }}
                />
              )}
            </div>
          </div>
          {impersonateUserId && (
            <p className='text-xs text-muted-foreground mt-1.5 ml-7'>
              Queries will run with this user's RLS context
            </p>
          )}
        </div>

        {isConnecting && <ConnectionLoadingState />}

        {connectionError && !isConnecting && (
          <ConnectionErrorState error={connectionError} />
        )}

        {isConnected && !connectionError && (
          <>
            <div className='flex-1 min-h-0 overflow-y-auto overflow-x-hidden px-6 py-4'>
              <div className='space-y-4'>
                {messages.map((msg) => (
                  <MessageBubble key={msg.id} message={msg} />
                ))}

                {isThinking && (
                  <div className='flex items-center gap-2 text-sm text-muted-foreground'>
                    <Loader2 className='h-4 w-4 animate-spin' />
                    {currentProgress || 'Thinking...'}
                  </div>
                )}

                <div ref={messagesEndRef} />
              </div>
            </div>

            <div className='border-t px-6 py-4 shrink-0'>
              <form onSubmit={handleSendMessage} className='flex gap-2'>
                <Input
                  value={inputValue}
                  onChange={(e) => setInputValue(e.target.value)}
                  placeholder='Ask a question...'
                  disabled={!isConnected || isSending}
                  className='flex-1'
                />
                <Button
                  type='submit'
                  disabled={!isConnected || isSending || !inputValue.trim()}
                  size='icon'
                >
                  {isSending ? (
                    <Loader2 className='h-4 w-4 animate-spin' />
                  ) : (
                    <Send className='h-4 w-4' />
                  )}
                </Button>
              </form>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}

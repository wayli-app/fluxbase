import { useState, useEffect } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Bot, RefreshCw, HardDrive, Trash2, Settings, MessageSquare } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { chatbotsApi, type AIChatbotSummary } from '@/lib/api'
import { ChatbotSettingsDialog } from '@/components/chatbots/chatbot-settings-dialog'
import { ChatbotTestDialog } from '@/components/chatbots/chatbot-test-dialog'

export const Route = createFileRoute('/_authenticated/chatbots/')({
  component: ChatbotsPage,
})

function ChatbotsPage() {
  const [chatbots, setChatbots] = useState<AIChatbotSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [reloading, setReloading] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [settingsChatbot, setSettingsChatbot] = useState<AIChatbotSummary | null>(null)
  const [testChatbot, setTestChatbot] = useState<AIChatbotSummary | null>(null)

  const fetchChatbots = async () => {
    setLoading(true)
    try {
      const data = await chatbotsApi.list()
      setChatbots(data || [])
    } catch {
      toast.error('Failed to fetch chatbots')
    } finally {
      setLoading(false)
    }
  }

  const handleReloadClick = async () => {
    setReloading(true)
    try {
      const result = await chatbotsApi.sync()
      const { created, updated, deleted, errors } = result.summary

      if (created > 0 || updated > 0 || deleted > 0) {
        const messages = []
        if (created > 0) messages.push(`${created} created`)
        if (updated > 0) messages.push(`${updated} updated`)
        if (deleted > 0) messages.push(`${deleted} deleted`)

        toast.success(`Chatbots synced: ${messages.join(', ')}`)
      } else if (errors > 0) {
        toast.error(`Failed to sync chatbots: ${errors} errors`)
      } else {
        toast.info('No changes detected')
      }

      await fetchChatbots()
    } catch {
      toast.error('Failed to sync chatbots from filesystem')
    } finally {
      setReloading(false)
    }
  }

  const toggleChatbot = async (chatbot: AIChatbotSummary) => {
    const newEnabledState = !chatbot.enabled

    try {
      await chatbotsApi.toggle(chatbot.id, newEnabledState)
      toast.success(`Chatbot ${newEnabledState ? 'enabled' : 'disabled'}`)
      await fetchChatbots()
    } catch {
      toast.error('Failed to toggle chatbot')
    }
  }

  const deleteChatbot = async (id: string) => {
    try {
      await chatbotsApi.delete(id)
      toast.success('Chatbot deleted successfully')
      await fetchChatbots()
    } catch {
      toast.error('Failed to delete chatbot')
    } finally {
      setDeleteConfirm(null)
    }
  }

  useEffect(() => {
    fetchChatbots()
  }, [])

  if (loading) {
    return (
      <div className='flex h-96 items-center justify-center'>
        <RefreshCw className='text-muted-foreground h-8 w-8 animate-spin' />
      </div>
    )
  }

  return (
    <div className='flex flex-1 flex-col gap-6 p-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-3xl font-bold'>AI Chatbots</h1>
          <p className='text-muted-foreground'>
            Manage AI-powered chatbots for database interactions
          </p>
        </div>
      </div>

      <div className='flex items-center justify-between'>
        <div className='flex gap-4 text-sm'>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Total:</span>
            <Badge variant='secondary' className='h-5 px-2'>
              {chatbots.length}
            </Badge>
          </div>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Active:</span>
            <Badge variant='secondary' className='h-5 px-2 bg-green-500/10 text-green-600 dark:text-green-400'>
              {chatbots.filter((c) => c.enabled).length}
            </Badge>
          </div>
        </div>
        <div className='flex items-center gap-2'>
          <Button
            onClick={handleReloadClick}
            variant='outline'
            size='sm'
            disabled={reloading}
          >
            {reloading ? (
              <>
                <RefreshCw className='mr-2 h-4 w-4 animate-spin' />
                Syncing...
              </>
            ) : (
              <>
                <HardDrive className='mr-2 h-4 w-4' />
                Sync from Filesystem
              </>
            )}
          </Button>
          <Button
            onClick={() => fetchChatbots()}
            variant='outline'
            size='sm'
          >
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
        </div>
      </div>

      <ScrollArea className='h-[calc(100vh-16rem)]'>
        <div className='grid gap-1'>
          {chatbots.length === 0 ? (
            <Card>
              <CardContent className='p-12 text-center'>
                <Bot className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
                <p className='mb-2 text-lg font-medium'>
                  No chatbots yet
                </p>
                <p className='text-muted-foreground mb-4 text-sm'>
                  Create chatbot files in the ./chatbots directory and sync them to get started
                </p>
                <Button onClick={handleReloadClick}>
                  <HardDrive className='mr-2 h-4 w-4' />
                  Sync from Filesystem
                </Button>
              </CardContent>
            </Card>
          ) : (
            chatbots.map((chatbot) => (
              <div
                key={chatbot.id}
                className='flex items-center justify-between gap-2 px-3 py-1.5 rounded-md border hover:border-primary/50 transition-colors bg-card'
              >
                <div className='flex items-center gap-2 min-w-0 flex-1'>
                  <Bot className='h-4 w-4 shrink-0' />
                  <span className='text-sm font-medium truncate'>{chatbot.name}</span>
                  {chatbot.namespace !== 'default' && (
                    <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      {chatbot.namespace}
                    </Badge>
                  )}
                  {chatbot.version > 0 && (
                    <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      v{chatbot.version}
                    </Badge>
                  )}
                  {chatbot.source && (
                    <Badge variant='outline' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      {chatbot.source}
                    </Badge>
                  )}
                  {chatbot.model && (
                    <Badge variant='secondary' className='shrink-0 text-[10px] px-1 py-0 h-4'>
                      {chatbot.model}
                    </Badge>
                  )}
                  <Switch
                    checked={chatbot.enabled}
                    onCheckedChange={() => toggleChatbot(chatbot)}
                    className='scale-75'
                  />
                </div>
                <div className='flex items-center gap-0.5 shrink-0'>
                  {chatbot.description && (
                    <span className='text-[10px] text-muted-foreground mr-2 max-w-[200px] truncate'>
                      {chatbot.description}
                    </span>
                  )}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => setTestChatbot(chatbot)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0'
                      >
                        <MessageSquare className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Test chatbot</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => setSettingsChatbot(chatbot)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0'
                      >
                        <Settings className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Settings</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        onClick={() => setDeleteConfirm(chatbot.id)}
                        size='sm'
                        variant='ghost'
                        className='h-6 w-6 p-0 text-destructive hover:text-destructive hover:bg-destructive/10'
                      >
                        <Trash2 className='h-3 w-3' />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Delete chatbot</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            ))
          )}
        </div>
      </ScrollArea>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteConfirm !== null} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Chatbot</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this chatbot? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && deleteChatbot(deleteConfirm)}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Settings Dialog */}
      {settingsChatbot && (
        <ChatbotSettingsDialog
          chatbot={settingsChatbot}
          open={settingsChatbot !== null}
          onOpenChange={(open) => !open && setSettingsChatbot(null)}
        />
      )}

      {/* Test Dialog */}
      {testChatbot && (
        <ChatbotTestDialog
          chatbot={testChatbot}
          open={testChatbot !== null}
          onOpenChange={(open) => !open && setTestChatbot(null)}
        />
      )}
    </div>
  )
}

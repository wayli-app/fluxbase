import { useState, useEffect } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { BookOpen, Plus, RefreshCw, Trash2, Settings, Search, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  knowledgeBasesApi,
  type KnowledgeBaseSummary,
  type CreateKnowledgeBaseRequest,
} from '@/lib/api'

export const Route = createFileRoute('/_authenticated/knowledge-bases/')({
  component: KnowledgeBasesPage,
})

function KnowledgeBasesPage() {
  const navigate = useNavigate()
  const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBaseSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newKB, setNewKB] = useState<CreateKnowledgeBaseRequest>({
    name: '',
    description: '',
    chunk_size: 512,
    chunk_overlap: 50,
    chunk_strategy: 'recursive',
  })

  const fetchKnowledgeBases = async () => {
    setLoading(true)
    try {
      const data = await knowledgeBasesApi.list()
      setKnowledgeBases(data || [])
    } catch {
      toast.error('Failed to fetch knowledge bases')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    if (!newKB.name.trim()) {
      toast.error('Name is required')
      return
    }

    try {
      await knowledgeBasesApi.create(newKB)
      toast.success('Knowledge base created')
      setCreateDialogOpen(false)
      setNewKB({
        name: '',
        description: '',
        chunk_size: 512,
        chunk_overlap: 50,
        chunk_strategy: 'recursive',
      })
      await fetchKnowledgeBases()
    } catch {
      toast.error('Failed to create knowledge base')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await knowledgeBasesApi.delete(id)
      toast.success('Knowledge base deleted')
      await fetchKnowledgeBases()
    } catch {
      toast.error('Failed to delete knowledge base')
    } finally {
      setDeleteConfirm(null)
    }
  }

  const toggleEnabled = async (kb: KnowledgeBaseSummary) => {
    try {
      await knowledgeBasesApi.update(kb.id, { enabled: !kb.enabled })
      toast.success(`Knowledge base ${kb.enabled ? 'disabled' : 'enabled'}`)
      await fetchKnowledgeBases()
    } catch {
      toast.error('Failed to update knowledge base')
    }
  }

  useEffect(() => {
    fetchKnowledgeBases()
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
          <h1 className='text-3xl font-bold'>Knowledge Bases</h1>
          <p className='text-muted-foreground'>
            Manage knowledge bases for RAG-powered AI chatbots
          </p>
        </div>
      </div>

      <div className='flex items-center justify-between'>
        <div className='flex gap-4 text-sm'>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Total:</span>
            <Badge variant='secondary' className='h-5 px-2'>
              {knowledgeBases.length}
            </Badge>
          </div>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Active:</span>
            <Badge variant='secondary' className='h-5 px-2 bg-green-500/10 text-green-600 dark:text-green-400'>
              {knowledgeBases.filter((kb) => kb.enabled).length}
            </Badge>
          </div>
          <div className='flex items-center gap-1.5'>
            <span className='text-muted-foreground'>Documents:</span>
            <Badge variant='secondary' className='h-5 px-2'>
              {knowledgeBases.reduce((sum, kb) => sum + kb.document_count, 0)}
            </Badge>
          </div>
        </div>
        <div className='flex items-center gap-2'>
          <Button onClick={() => fetchKnowledgeBases()} variant='outline' size='sm'>
            <RefreshCw className='mr-2 h-4 w-4' />
            Refresh
          </Button>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button size='sm'>
                <Plus className='mr-2 h-4 w-4' />
                Create Knowledge Base
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Knowledge Base</DialogTitle>
                <DialogDescription>
                  Create a new knowledge base for storing documents and providing context to AI chatbots.
                </DialogDescription>
              </DialogHeader>
              <div className='grid gap-4 py-4'>
                <div className='grid gap-2'>
                  <Label htmlFor='name'>Name</Label>
                  <Input
                    id='name'
                    value={newKB.name}
                    onChange={(e) => setNewKB({ ...newKB, name: e.target.value })}
                    placeholder='e.g., product-docs'
                  />
                </div>
                <div className='grid gap-2'>
                  <Label htmlFor='description'>Description</Label>
                  <Textarea
                    id='description'
                    value={newKB.description || ''}
                    onChange={(e) => setNewKB({ ...newKB, description: e.target.value })}
                    placeholder='What kind of documents will this knowledge base contain?'
                  />
                </div>
                <div className='grid grid-cols-2 gap-4'>
                  <div className='grid gap-2'>
                    <Label htmlFor='chunk_size'>Chunk Size</Label>
                    <Input
                      id='chunk_size'
                      type='number'
                      value={newKB.chunk_size}
                      onChange={(e) => setNewKB({ ...newKB, chunk_size: parseInt(e.target.value) || 512 })}
                    />
                    <p className='text-xs text-muted-foreground'>Characters per chunk</p>
                  </div>
                  <div className='grid gap-2'>
                    <Label htmlFor='chunk_overlap'>Chunk Overlap</Label>
                    <Input
                      id='chunk_overlap'
                      type='number'
                      value={newKB.chunk_overlap}
                      onChange={(e) => setNewKB({ ...newKB, chunk_overlap: parseInt(e.target.value) || 50 })}
                    />
                    <p className='text-xs text-muted-foreground'>Overlap between chunks</p>
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button variant='outline' onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleCreate}>Create</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <ScrollArea className='h-[calc(100vh-16rem)]'>
        {knowledgeBases.length === 0 ? (
          <Card>
            <CardContent className='p-12 text-center'>
              <BookOpen className='text-muted-foreground mx-auto mb-4 h-12 w-12' />
              <p className='mb-2 text-lg font-medium'>No knowledge bases yet</p>
              <p className='text-muted-foreground mb-4 text-sm'>
                Create a knowledge base to store documents for RAG-powered AI chatbots
              </p>
              <Button onClick={() => setCreateDialogOpen(true)}>
                <Plus className='mr-2 h-4 w-4' />
                Create Knowledge Base
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
            {knowledgeBases.map((kb) => (
              <Card key={kb.id} className='relative'>
                <CardHeader className='pb-2'>
                  <div className='flex items-start justify-between'>
                    <div className='flex items-center gap-2'>
                      <BookOpen className='h-5 w-5' />
                      <CardTitle className='text-lg'>{kb.name}</CardTitle>
                    </div>
                    <div className='flex items-center gap-1'>
                      <Switch
                        checked={kb.enabled}
                        onCheckedChange={() => toggleEnabled(kb)}
                      />
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-8 w-8 p-0'
                            onClick={() => navigate({ to: `/knowledge-bases/$id`, params: { id: kb.id } })}
                          >
                            <FileText className='h-4 w-4' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>View Documents</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-8 w-8 p-0'
                            onClick={() => navigate({ to: `/knowledge-bases/$id/search`, params: { id: kb.id } })}
                          >
                            <Search className='h-4 w-4' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Search</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-8 w-8 p-0'
                            onClick={() => navigate({ to: `/knowledge-bases/$id/settings`, params: { id: kb.id } })}
                          >
                            <Settings className='h-4 w-4' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Settings</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-8 w-8 p-0 text-destructive hover:text-destructive'
                            onClick={() => setDeleteConfirm(kb.id)}
                          >
                            <Trash2 className='h-4 w-4' />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Delete</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                  {kb.namespace !== 'default' && (
                    <Badge variant='outline' className='w-fit text-[10px]'>
                      {kb.namespace}
                    </Badge>
                  )}
                </CardHeader>
                <CardContent>
                  {kb.description && (
                    <CardDescription className='mb-3 line-clamp-2'>
                      {kb.description}
                    </CardDescription>
                  )}
                  <div className='flex flex-wrap gap-2 text-xs'>
                    <Badge variant='secondary'>
                      {kb.document_count} {kb.document_count === 1 ? 'document' : 'documents'}
                    </Badge>
                    <Badge variant='secondary'>
                      {kb.total_chunks} chunks
                    </Badge>
                    {kb.embedding_model && (
                      <Badge variant='outline' className='text-[10px]'>
                        {kb.embedding_model}
                      </Badge>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </ScrollArea>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteConfirm !== null} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Knowledge Base</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this knowledge base? This will permanently delete all documents and chunks. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && handleDelete(deleteConfirm)}
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { userKnowledgeBasesApi } from '@/lib/api'

interface QuotaConfig {
  maxDocuments: number
  maxChunks: number
  maxStorageMB: number
}

interface CreateKnowledgeBaseRequest {
  name: string
  description: string
  visibility: 'private' | 'shared' | 'public'
  quota_max_documents?: number
  quota_max_chunks?: number
  quota_max_storage_bytes?: number
}

interface CreateKnowledgeBaseDialogProps {
  onClose: () => void
}

function CreateKnowledgeBaseDialog({ onClose }: CreateKnowledgeBaseDialogProps) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [visibility, setVisibility] = useState<'private' | 'shared' | 'public'>('private')
  const [useCustomQuotas, setUseCustomQuotas] = useState(false)
  const [quotaConfig, setQuotaConfig] = useState<QuotaConfig>({
    maxDocuments: 1000,
    maxChunks: 50000,
    maxStorageMB: 1024, // 1GB in MB
  })
  const queryClient = useQueryClient()

  const createMutation = useMutation({
    mutationFn: async () => {
      const body: CreateKnowledgeBaseRequest = { name, description, visibility }

      if (useCustomQuotas) {
        body.quota_max_documents = quotaConfig.maxDocuments
        body.quota_max_chunks = quotaConfig.maxChunks
        body.quota_max_storage_bytes = quotaConfig.maxStorageMB * 1024 * 1024
      }

      return userKnowledgeBasesApi.create(body)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-knowledge-bases'] })
      toast.success('Knowledge base created')
      onClose()
    },
    onError: (error: Error) => {
      toast.error('Failed to create knowledge base', {
        description: error.message,
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    createMutation.mutate()
  }

  const formatBytes = (bytes: number) => {
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
    }
    if (bytes >= 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    }
    return `${bytes} bytes`
  }

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Knowledge Base</DialogTitle>
          <DialogDescription>Create a new knowledge base for your AI assistant</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Information */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold">Basic Information</h3>

            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="My Knowledge Base"
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="A brief description of this knowledge base"
                rows={3}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="visibility">Visibility</Label>
              <Select value={visibility} onValueChange={(v: 'private' | 'shared' | 'public') => setVisibility(v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="private">Private - Only you</SelectItem>
                  <SelectItem value="shared">Shared - With specific users</SelectItem>
                  <SelectItem value="public">Public - Anyone with access</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Quota Configuration */}
          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="use-custom-quotas"
                checked={useCustomQuotas}
                onCheckedChange={(checked) => setUseCustomQuotas(checked === true)}
              />
              <Label htmlFor="use-custom-quotas" className="cursor-pointer">
                Configure custom quota limits
              </Label>
            </div>

            {useCustomQuotas && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 bg-muted/50 rounded-lg space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="max-documents">Max Documents</Label>
                  <Input
                    id="max-documents"
                    type="number"
                    min={1}
                    value={quotaConfig.maxDocuments}
                    onChange={(e) => setQuotaConfig({ ...quotaConfig, maxDocuments: parseInt(e.target.value) || 1000 })}
                    className="text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Default: 1,000 documents
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="max-chunks">Max Chunks</Label>
                  <Input
                    id="max-chunks"
                    type="number"
                    min={1}
                    value={quotaConfig.maxChunks}
                    onChange={(e) => setQuotaConfig({ ...quotaConfig, maxChunks: parseInt(e.target.value) || 50000 })}
                    className="text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Default: 50,000 chunks
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="max-storage">Max Storage (MB)</Label>
                  <Input
                    id="max-storage"
                    type="number"
                    min={1}
                    value={quotaConfig.maxStorageMB}
                    onChange={(e) => setQuotaConfig({ ...quotaConfig, maxStorageMB: parseInt(e.target.value) || 1024 })}
                    className="text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Default: 1,024 MB ({formatBytes(quotaConfig.maxStorageMB * 1024 * 1024)})
                  </p>
                </div>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending || !name.trim()}>
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export { CreateKnowledgeBaseDialog }

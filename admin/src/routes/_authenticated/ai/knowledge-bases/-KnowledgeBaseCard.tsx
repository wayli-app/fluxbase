import { useState } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { MoreVertical, Eye, Edit, Share2, Trash2 } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { ShareKnowledgeBaseDialog } from './-ShareKnowledgeBaseDialog'
import { userKnowledgeBasesApi, type KnowledgeBaseSummary } from '@/lib/api'

interface KnowledgeBaseCardProps {
  kb: KnowledgeBaseSummary
  isOwner: boolean
}

function KnowledgeBaseCard({ kb, isOwner }: KnowledgeBaseCardProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [shareDialogOpen, setShareDialogOpen] = useState(false)

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this knowledge base?')) {
      return
    }

    try {
      await userKnowledgeBasesApi.delete(id)
      toast.success('Knowledge base deleted')
      queryClient.invalidateQueries({ queryKey: ['user-knowledge-bases'] })
    } catch (error) {
      toast.error('Failed to delete knowledge base', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  const getVisibilityBadge = () => {
    if (!kb.visibility || kb.visibility === 'private') {
      return <Badge variant="outline">Private</Badge>
    }
    if (kb.visibility === 'shared') {
      return <Badge variant="secondary">Shared</Badge>
    }
    return <Badge>Public</Badge>
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-start">
          <div className="flex-1">
            <CardTitle className="text-lg">{kb.name}</CardTitle>
            <div className="flex gap-2 mt-2">
              {getVisibilityBadge()}
              {isOwner && <Badge variant="default">Owner</Badge>}
              {!isOwner && kb.user_permission && (
                <Badge variant="outline">{kb.user_permission}</Badge>
              )}
            </div>
          </div>
          {isOwner && (
            <>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm">
                    <MoreVertical className="w-4 h-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    onClick={() => navigate({ to: '/knowledge-bases/$id', params: { id: kb.id } })}
                  >
                    <Eye className="w-4 h-4 mr-2" />
                    View
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() =>
                      navigate({ to: '/knowledge-bases/$id/settings', params: { id: kb.id } })
                    }
                  >
                    <Edit className="w-4 h-4 mr-2" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setShareDialogOpen(true)}>
                    <Share2 className="w-4 h-4 mr-2" />
                    Share
                  </DropdownMenuItem>
                  <DropdownMenuItem className="text-destructive" onClick={() => handleDelete(kb.id)}>
                    <Trash2 className="w-4 h-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <ShareKnowledgeBaseDialog
                kbId={kb.id}
                kbName={kb.name}
                open={shareDialogOpen}
                onClose={() => setShareDialogOpen(false)}
              />
            </>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {kb.description && (
          <p className="text-sm text-muted-foreground mb-4">{kb.description}</p>
        )}
        <div className="flex gap-4 text-sm text-muted-foreground">
          <span>{kb.document_count || 0} documents</span>
          <span>{kb.total_chunks || 0} chunks</span>
        </div>
      </CardContent>
    </Card>
  )
}

export { KnowledgeBaseCard }

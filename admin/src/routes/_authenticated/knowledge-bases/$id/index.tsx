import { useState, useEffect, useRef, useCallback } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  Plus,
  RefreshCw,
  Trash2,
  FileText,
  Clock,
  CheckCircle,
  XCircle,
  Loader2,
  Upload,
  AlertTriangle,
  Pencil,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  knowledgeBasesApi,
  type KnowledgeBase,
  type KnowledgeBaseDocument,
} from '@/lib/api'

const SUPPORTED_FILE_TYPES = [
  '.pdf',
  '.txt',
  '.md',
  '.html',
  '.htm',
  '.csv',
  '.docx',
  '.xlsx',
  '.rtf',
  '.epub',
  '.json',
]

const MAX_FILE_SIZE = 50 * 1024 * 1024 // 50MB

interface KnowledgeBaseCapabilities {
  ocr_enabled: boolean
  ocr_available: boolean
  ocr_languages: string[]
  supported_file_types: string[]
}

export const Route = createFileRoute(
  '/_authenticated/knowledge-bases/$id/'
)({
  component: KnowledgeBaseDetailPage,
})

function KnowledgeBaseDetailPage() {
  const { id } = Route.useParams()
  const navigate = useNavigate()
  const [knowledgeBase, setKnowledgeBase] = useState<KnowledgeBase | null>(null)
  const [documents, setDocuments] = useState<KnowledgeBaseDocument[]>([])
  const [loading, setLoading] = useState(true)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [newDoc, setNewDoc] = useState({ title: '', content: '' })
  const [newDocTags, setNewDocTags] = useState('')
  const [newDocMetadata, setNewDocMetadata] = useState<{ key: string; value: string }[]>([])
  const [adding, setAdding] = useState(false)
  const [uploadMode, setUploadMode] = useState<'paste' | 'upload'>('paste')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [capabilities, setCapabilities] = useState<KnowledgeBaseCapabilities | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Edit document state
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingDoc, setEditingDoc] = useState<KnowledgeBaseDocument | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editTags, setEditTags] = useState('')
  const [editMetadata, setEditMetadata] = useState<{ key: string; value: string }[]>([])
  const [saving, setSaving] = useState(false)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [kb, docs] = await Promise.all([
        knowledgeBasesApi.get(id),
        knowledgeBasesApi.listDocuments(id),
      ])
      setKnowledgeBase(kb)
      setDocuments(docs || [])
    } catch {
      toast.error('Failed to fetch knowledge base')
    } finally {
      setLoading(false)
    }
  }, [id])

  const handleAddDocument = async () => {
    if (!newDoc.content.trim()) {
      toast.error('Content is required')
      return
    }

    // Parse tags from comma-separated string
    const tags = newDocTags
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0)

    // Convert metadata array to object
    const metadata: Record<string, string> = {}
    newDocMetadata.forEach(({ key, value }) => {
      if (key.trim() && value.trim()) {
        metadata[key.trim()] = value.trim()
      }
    })

    setAdding(true)
    try {
      await knowledgeBasesApi.addDocument(id, {
        title: newDoc.title || undefined,
        content: newDoc.content,
        tags: tags.length > 0 ? tags : undefined,
        metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      })
      toast.success('Document added - processing will begin shortly')
      setAddDialogOpen(false)
      setNewDoc({ title: '', content: '' })
      setNewDocTags('')
      setNewDocMetadata([])
      await fetchData()
    } catch {
      toast.error('Failed to add document')
    } finally {
      setAdding(false)
    }
  }

  const handleOpenEditDialog = (doc: KnowledgeBaseDocument) => {
    setEditingDoc(doc)
    setEditTitle(doc.title || '')
    setEditTags(doc.tags?.join(', ') || '')
    // Parse existing metadata
    const existingMetadata: { key: string; value: string }[] = []
    if (doc.metadata) {
      try {
        const parsed = typeof doc.metadata === 'string' ? JSON.parse(doc.metadata) : doc.metadata
        Object.entries(parsed).forEach(([key, value]) => {
          existingMetadata.push({ key, value: String(value) })
        })
      } catch {
        // Ignore parse errors
      }
    }
    setEditMetadata(existingMetadata)
    setEditDialogOpen(true)
  }

  const handleSaveDocument = async () => {
    if (!editingDoc) return

    // Parse tags from comma-separated string
    const tags = editTags
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0)

    // Convert metadata array to object
    const metadata: Record<string, string> = {}
    editMetadata.forEach(({ key, value }) => {
      if (key.trim() && value.trim()) {
        metadata[key.trim()] = value.trim()
      }
    })

    setSaving(true)
    try {
      await knowledgeBasesApi.updateDocument(id, editingDoc.id, {
        title: editTitle || undefined,
        tags,
        metadata,
      })
      toast.success('Document updated')
      setEditDialogOpen(false)
      setEditingDoc(null)
      await fetchData()
    } catch {
      toast.error('Failed to update document')
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteDocument = async (docId: string) => {
    try {
      await knowledgeBasesApi.deleteDocument(id, docId)
      toast.success('Document deleted')
      await fetchData()
    } catch {
      toast.error('Failed to delete document')
    } finally {
      setDeleteConfirm(null)
    }
  }

  const handleUploadDocument = async () => {
    if (!selectedFile) {
      toast.error('Please select a file')
      return
    }

    if (selectedFile.size > MAX_FILE_SIZE) {
      toast.error(`File too large. Maximum size is ${MAX_FILE_SIZE / (1024 * 1024)}MB`)
      return
    }

    setAdding(true)
    try {
      await knowledgeBasesApi.uploadDocument(id, selectedFile, newDoc.title || undefined)
      toast.success('Document uploaded - processing will begin shortly')
      setAddDialogOpen(false)
      setNewDoc({ title: '', content: '' })
      setSelectedFile(null)
      setUploadMode('paste')
      await fetchData()
    } catch {
      toast.error('Failed to upload document')
    } finally {
      setAdding(false)
    }
  }

  const handleFileSelect = useCallback((file: File) => {
    const ext = '.' + file.name.split('.').pop()?.toLowerCase()
    if (!SUPPORTED_FILE_TYPES.includes(ext)) {
      toast.error(`Unsupported file type: ${ext}. Supported: ${SUPPORTED_FILE_TYPES.join(', ')}`)
      return
    }
    if (file.size > MAX_FILE_SIZE) {
      toast.error(`File too large. Maximum size is ${MAX_FILE_SIZE / (1024 * 1024)}MB`)
      return
    }
    setSelectedFile(file)
    // Use filename without extension as default title
    setNewDoc((prev) => ({ ...prev, title: file.name.replace(/\.[^/.]+$/, '') }))
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setIsDragging(false)
      const file = e.dataTransfer.files[0]
      if (file) {
        handleFileSelect(file)
      }
    },
    [handleFileSelect]
  )

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'indexed':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'processing':
        return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />
      case 'pending':
        return <Clock className="h-4 w-4 text-yellow-500" />
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <Clock className="h-4 w-4 text-muted-foreground" />
    }
  }

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Poll for status updates when documents are processing
  useEffect(() => {
    const hasProcessing = documents.some(
      (doc) => doc.status === 'pending' || doc.status === 'processing'
    )

    if (!hasProcessing) return

    const interval = setInterval(() => {
      knowledgeBasesApi.listDocuments(id).then((docs) => {
        setDocuments(docs || [])
      }).catch(() => {})
    }, 3000) // Poll every 3 seconds

    return () => clearInterval(interval)
  }, [documents, id])

  if (loading) {
    return (
      <div className="flex h-96 items-center justify-center">
        <RefreshCw className="text-muted-foreground h-8 w-8 animate-spin" />
      </div>
    )
  }

  if (!knowledgeBase) {
    return (
      <div className="flex h-96 flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">Knowledge base not found</p>
        <Button
          variant="outline"
          onClick={() => navigate({ to: '/knowledge-bases' })}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Knowledge Bases
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6 p-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate({ to: '/knowledge-bases' })}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">{knowledgeBase.name}</h1>
          {knowledgeBase.description && (
            <p className="text-muted-foreground">{knowledgeBase.description}</p>
          )}
        </div>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex gap-4 text-sm">
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Documents:</span>
            <Badge variant="secondary" className="h-5 px-2">
              {knowledgeBase.document_count}
            </Badge>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Chunks:</span>
            <Badge variant="secondary" className="h-5 px-2">
              {knowledgeBase.total_chunks}
            </Badge>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Model:</span>
            <Badge variant="outline" className="h-5 px-2 text-[10px]">
              {knowledgeBase.embedding_model}
            </Badge>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={() => fetchData()} variant="outline" size="sm">
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Dialog open={addDialogOpen} onOpenChange={(open) => {
            setAddDialogOpen(open)
            if (open) {
              // Fetch capabilities when dialog opens
              knowledgeBasesApi.getCapabilities().then(setCapabilities).catch(() => {
                // Silently fail - we'll just not show the warning
              })
            } else {
              setNewDoc({ title: '', content: '' })
              setNewDocTags('')
              setNewDocMetadata([])
              setSelectedFile(null)
              setUploadMode('paste')
            }
          }}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="mr-2 h-4 w-4" />
                Add Document
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
              <DialogHeader>
                <DialogTitle>Add Document</DialogTitle>
                <DialogDescription>
                  Add a new document to this knowledge base. The document will
                  be chunked and embedded for search.
                </DialogDescription>
              </DialogHeader>
              <Tabs value={uploadMode} onValueChange={(v) => setUploadMode(v as 'paste' | 'upload')} className="w-full">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="paste">Paste Text</TabsTrigger>
                  <TabsTrigger value="upload">Upload File</TabsTrigger>
                </TabsList>
                <TabsContent value="paste" className="mt-4">
                  <div className="grid gap-4">
                    <div className="grid gap-2">
                      <Label htmlFor="title">Title (optional)</Label>
                      <Input
                        id="title"
                        value={newDoc.title}
                        onChange={(e) =>
                          setNewDoc({ ...newDoc, title: e.target.value })
                        }
                        placeholder="Document title"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="content">Content</Label>
                      <Textarea
                        id="content"
                        value={newDoc.content}
                        onChange={(e) =>
                          setNewDoc({ ...newDoc, content: e.target.value })
                        }
                        placeholder="Paste your document content here..."
                        className="min-h-[200px]"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="tags">Tags (optional)</Label>
                      <Input
                        id="tags"
                        value={newDocTags}
                        onChange={(e) => setNewDocTags(e.target.value)}
                        placeholder="food, japanese, tokyo (comma-separated)"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label>Metadata (optional)</Label>
                      <div className="space-y-2">
                        {newDocMetadata.map((field, i) => (
                          <div key={i} className="flex gap-2">
                            <Input
                              placeholder="Key"
                              value={field.key}
                              onChange={(e) => {
                                const updated = [...newDocMetadata]
                                updated[i].key = e.target.value
                                setNewDocMetadata(updated)
                              }}
                              className="flex-1"
                            />
                            <Input
                              placeholder="Value"
                              value={field.value}
                              onChange={(e) => {
                                const updated = [...newDocMetadata]
                                updated[i].value = e.target.value
                                setNewDocMetadata(updated)
                              }}
                              className="flex-1"
                            />
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-9 w-9 p-0"
                              onClick={() => {
                                setNewDocMetadata(newDocMetadata.filter((_, j) => j !== i))
                              }}
                            >
                              <X className="h-4 w-4" />
                            </Button>
                          </div>
                        ))}
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setNewDocMetadata([...newDocMetadata, { key: '', value: '' }])}
                        >
                          <Plus className="mr-2 h-4 w-4" />
                          Add Field
                        </Button>
                      </div>
                    </div>
                  </div>
                </TabsContent>
                <TabsContent value="upload" className="mt-4">
                  <div className="grid gap-4">
                    {capabilities && !capabilities.ocr_available && (
                      <Alert className="border-amber-500/50 bg-amber-500/10">
                        <AlertTriangle className="h-4 w-4 text-amber-500" />
                        <AlertDescription className="text-amber-700 dark:text-amber-400">
                          OCR is not available. Scanned or image-based PDFs may not be readable.
                          Text-based documents will still work normally.
                        </AlertDescription>
                      </Alert>
                    )}
                    <div className="grid gap-2">
                      <Label htmlFor="file-title">Title (optional)</Label>
                      <Input
                        id="file-title"
                        value={newDoc.title}
                        onChange={(e) =>
                          setNewDoc({ ...newDoc, title: e.target.value })
                        }
                        placeholder="Document title"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label>File</Label>
                      <div
                        className={`relative flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors ${
                          isDragging
                            ? 'border-primary bg-primary/5'
                            : 'border-muted-foreground/25 hover:border-muted-foreground/50'
                        }`}
                        onDragOver={handleDragOver}
                        onDragLeave={handleDragLeave}
                        onDrop={handleDrop}
                      >
                        <input
                          ref={fileInputRef}
                          type="file"
                          className="hidden"
                          accept={SUPPORTED_FILE_TYPES.join(',')}
                          onChange={(e) => {
                            const file = e.target.files?.[0]
                            if (file) handleFileSelect(file)
                          }}
                        />
                        {selectedFile ? (
                          <div className="flex flex-col items-center gap-2">
                            <FileText className="h-10 w-10 text-primary" />
                            <p className="font-medium">{selectedFile.name}</p>
                            <p className="text-muted-foreground text-sm">
                              {(selectedFile.size / 1024).toFixed(1)} KB
                            </p>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => {
                                setSelectedFile(null)
                                if (fileInputRef.current) {
                                  fileInputRef.current.value = ''
                                }
                              }}
                            >
                              Remove
                            </Button>
                          </div>
                        ) : (
                          <div className="flex flex-col items-center gap-2">
                            <Upload className="text-muted-foreground h-10 w-10" />
                            <p className="text-muted-foreground text-sm">
                              Drag and drop a file here, or
                            </p>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => fileInputRef.current?.click()}
                            >
                              Browse Files
                            </Button>
                            <p className="text-muted-foreground mt-2 text-xs">
                              Supported: PDF, TXT, MD, HTML, CSV, DOCX, XLSX, RTF, EPUB, JSON (max 50MB)
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setAddDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={uploadMode === 'paste' ? handleAddDocument : handleUploadDocument}
                  disabled={adding || (uploadMode === 'paste' ? !newDoc.content.trim() : !selectedFile)}
                >
                  {adding && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {uploadMode === 'paste' ? 'Add Document' : 'Upload Document'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Documents</CardTitle>
          <CardDescription>
            Documents in this knowledge base that can be searched.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {documents.length === 0 ? (
            <div className="py-12 text-center">
              <FileText className="text-muted-foreground mx-auto mb-4 h-12 w-12" />
              <p className="mb-2 text-lg font-medium">No documents yet</p>
              <p className="text-muted-foreground mb-4 text-sm">
                Add documents to enable search in this knowledge base
              </p>
              <Button onClick={() => setAddDialogOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                Add Document
              </Button>
            </div>
          ) : (
            <ScrollArea className="h-[400px]">
              <div className="min-w-[600px]">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Title</TableHead>
                      <TableHead>Tags</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Chunks</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="w-[80px]"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {documents.map((doc) => (
                      <TableRow key={doc.id}>
                        <TableCell className="font-medium">
                          {doc.title || 'Untitled'}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1 flex-wrap max-w-[200px]">
                            {doc.tags?.slice(0, 3).map((tag) => (
                              <Badge key={tag} variant="secondary" className="text-xs">
                                {tag}
                              </Badge>
                            ))}
                            {doc.tags && doc.tags.length > 3 && (
                              <Badge variant="outline" className="text-xs">
                                +{doc.tags.length - 3}
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(doc.status)}
                            <span className="capitalize">{doc.status}</span>
                          </div>
                        </TableCell>
                        <TableCell>{doc.chunk_count}</TableCell>
                        <TableCell>
                          {new Date(doc.created_at).toLocaleDateString()}
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0"
                              onClick={() => handleOpenEditDialog(doc)}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                              onClick={() => setDeleteConfirm(doc.id)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
              <ScrollBar orientation="horizontal" />
            </ScrollArea>
          )}
        </CardContent>
      </Card>

      <AlertDialog
        open={deleteConfirm !== null}
        onOpenChange={(open) => !open && setDeleteConfirm(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Document</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this document? This will also
              delete all associated chunks. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteConfirm && handleDeleteDocument(deleteConfirm)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Edit Document Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={(open) => {
        setEditDialogOpen(open)
        if (!open) {
          setEditingDoc(null)
          setEditTitle('')
          setEditTags('')
          setEditMetadata([])
        }
      }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Edit Document</DialogTitle>
            <DialogDescription>
              Update the document's title, tags, and metadata.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-title">Title</Label>
              <Input
                id="edit-title"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                placeholder="Document title"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-tags">Tags</Label>
              <Input
                id="edit-tags"
                value={editTags}
                onChange={(e) => setEditTags(e.target.value)}
                placeholder="food, japanese, tokyo (comma-separated)"
              />
            </div>
            <div className="grid gap-2">
              <Label>Metadata</Label>
              <div className="space-y-2">
                {editMetadata.map((field, i) => (
                  <div key={i} className="flex gap-2">
                    <Input
                      placeholder="Key"
                      value={field.key}
                      onChange={(e) => {
                        const updated = [...editMetadata]
                        updated[i].key = e.target.value
                        setEditMetadata(updated)
                      }}
                      className="flex-1"
                    />
                    <Input
                      placeholder="Value"
                      value={field.value}
                      onChange={(e) => {
                        const updated = [...editMetadata]
                        updated[i].value = e.target.value
                        setEditMetadata(updated)
                      }}
                      className="flex-1"
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-9 w-9 p-0"
                      onClick={() => {
                        setEditMetadata(editMetadata.filter((_, j) => j !== i))
                      }}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setEditMetadata([...editMetadata, { key: '', value: '' }])}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Field
                </Button>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleSaveDocument} disabled={saving}>
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save Changes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

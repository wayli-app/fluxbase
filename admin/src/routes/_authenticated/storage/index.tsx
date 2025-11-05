import { createFileRoute } from '@tanstack/react-router'
import { useState, useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Checkbox } from '@/components/ui/checkbox'
import {
  FolderOpen,
  FolderPlus,
  File,
  Upload,
  Download,
  Trash2,
  Plus,
  Image as ImageIcon,
  FileText,
  FileJson,
  FileCode,
  FileCog,
  RefreshCw,
  Eye,
  Copy,
  Clock,
  HardDrive,
  ChevronRight,
  Home,
  Info,
  Link,
  Calendar,
  FileType,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'

export const Route = createFileRoute('/_authenticated/storage/')({
  component: StorageBrowser,
})

// Types
interface StorageObject {
  key: string
  size: number
  last_modified: string
  content_type?: string
  etag?: string
  metadata?: Record<string, string>
}

interface BucketListResponse {
  buckets: string[]
}

interface ObjectListResponse {
  bucket: string
  objects: StorageObject[] | null
  prefixes: string[]
  truncated: boolean
}

function StorageBrowser() {
  // State
  const [buckets, setBuckets] = useState<string[]>([])
  const [selectedBucket, setSelectedBucket] = useState<string>('')
  const [currentPrefix, setCurrentPrefix] = useState<string>('')
  const [objects, setObjects] = useState<StorageObject[]>([])
  const [prefixes, setPrefixes] = useState<string[]>([])
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({})
  const [searchQuery, setSearchQuery] = useState('')
  const [sortBy, setSortBy] = useState<'name' | 'size' | 'date'>('name')
  const [fileTypeFilter, setFileTypeFilter] = useState<string>('all')
  const [showCreateBucket, setShowCreateBucket] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [showFilePreview, setShowFilePreview] = useState(false)
  const [showCreateFolder, setShowCreateFolder] = useState(false)
  const [previewFile, setPreviewFile] = useState<StorageObject | null>(null)
  const [previewUrl, setPreviewUrl] = useState<string>('')
  const [newBucketName, setNewBucketName] = useState('')
  const [newFolderName, setNewFolderName] = useState('')
  const [dragActive, setDragActive] = useState(false)
  const [showMetadata, setShowMetadata] = useState(false)
  const [metadataFile, setMetadataFile] = useState<StorageObject | null>(null)
  const [signedUrl, setSignedUrl] = useState<string>('')
  const [signedUrlExpiry, setSignedUrlExpiry] = useState<number>(3600)
  const [generatingUrl, setGeneratingUrl] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Breadcrumb navigation
  const breadcrumbs = currentPrefix ? currentPrefix.split('/').filter(Boolean) : []

  // Load buckets on mount
  useEffect(() => {
    loadBuckets()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Load objects when bucket or prefix changes
  useEffect(() => {
    if (selectedBucket) {
      loadObjects()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedBucket, currentPrefix])

  const loadBuckets = async () => {
    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch('/api/v1/storage/buckets', {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const data: BucketListResponse = await res.json()
        setBuckets(data.buckets || [])
        if (data.buckets && data.buckets.length > 0 && !selectedBucket) {
          setSelectedBucket(data.buckets[0])
        }
      } else {
        toast.error('Failed to load buckets')
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to load buckets: ${errorMessage}`)
    } finally {
      setLoading(false)
    }
  }

  const loadObjects = async () => {
    if (!selectedBucket) return

    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const params = new URLSearchParams()
      if (currentPrefix) params.append('prefix', currentPrefix)
      params.append('delimiter', '/')

      const url = `/api/v1/storage/${selectedBucket}?${params.toString()}`
      const res = await fetch(url, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const data: ObjectListResponse = await res.json()
        setObjects(data.objects || [])
        setPrefixes(data.prefixes || [])
      } else {
        toast.error('Failed to load files')
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to load files: ${errorMessage}`)
    } finally {
      setLoading(false)
    }
  }

  const createBucket = async () => {
    if (!newBucketName.trim()) {
      toast.error('Bucket name is required')
      return
    }

    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/storage/buckets/${newBucketName}`, {
        method: 'POST',
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        toast.success(`Bucket "${newBucketName}" created`)
        setShowCreateBucket(false)
        setNewBucketName('')
        await loadBuckets()
        setSelectedBucket(newBucketName)
      } else {
        const error = await res.text()
        toast.error(`Failed to create bucket: ${error}`)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to create bucket: ${errorMessage}`)
    } finally {
      setLoading(false)
    }
  }

  const deleteBucket = async (bucketName: string) => {
    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/storage/buckets/${bucketName}`, {
        method: 'DELETE',
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        toast.success(`Bucket "${bucketName}" deleted`)
        await loadBuckets()
        if (selectedBucket === bucketName) {
          setSelectedBucket(buckets[0] || '')
        }
      } else {
        const error = await res.text()
        toast.error(`Failed to delete bucket: ${error}`)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to delete bucket: ${errorMessage}`)
    } finally {
      setLoading(false)
    }
  }

  const uploadFiles = async (files: FileList | File[]) => {
    if (!selectedBucket) {
      toast.error('Please select a bucket first')
      return
    }

    setUploading(true)
    const filesArray = Array.from(files)
    let successCount = 0

    try {
      // Upload files sequentially
      for (const file of filesArray) {
        const key = currentPrefix ? `${currentPrefix}${file.name}` : file.name
        const token = localStorage.getItem('fluxbase-auth-token')

        // Set initial progress
        setUploadProgress((prev) => ({ ...prev, [file.name]: 0 }))

        try {
          const formData = new FormData()
          formData.append('file', file)

          // Use XMLHttpRequest for better progress tracking and large file support
          const uploadUrl = `/api/v1/storage/${selectedBucket}/${encodeURIComponent(key)}`

          await new Promise<void>((resolve, reject) => {
            const xhr = new XMLHttpRequest()

            // Track upload progress
            xhr.upload.addEventListener('progress', (e) => {
              if (e.lengthComputable) {
                const percentComplete = Math.round((e.loaded / e.total) * 100)
                setUploadProgress((prev) => ({ ...prev, [file.name]: percentComplete }))
              }
            })

            // Handle completion
            xhr.addEventListener('load', () => {
              if (xhr.status >= 200 && xhr.status < 300) {
                setUploadProgress((prev) => ({ ...prev, [file.name]: 100 }))
                setTimeout(() => {
                  setUploadProgress((prev) => {
                    const updated = { ...prev }
                    delete updated[file.name]
                    return updated
                  })
                }, 500)
                successCount++
                resolve()
              } else {
                // eslint-disable-next-line no-console
                console.error(`Failed to upload ${file.name}: ${xhr.status} ${xhr.statusText}`)
                // eslint-disable-next-line no-console
                console.error('Response:', xhr.responseText)
                setUploadProgress((prev) => {
                  const updated = { ...prev }
                  delete updated[file.name]
                  return updated
                })
                reject(new Error(`Upload failed with status ${xhr.status}`))
              }
            })

            // Handle errors
            xhr.addEventListener('error', () => {
              // eslint-disable-next-line no-console
              console.error('Upload error')
              setUploadProgress((prev) => {
                const updated = { ...prev }
                delete updated[file.name]
                return updated
              })
              reject(new Error('Network error during upload'))
            })

            xhr.addEventListener('abort', () => {
              // eslint-disable-next-line no-console
              console.error('Upload aborted')
              setUploadProgress((prev) => {
                const updated = { ...prev }
                delete updated[file.name]
                return updated
              })
              reject(new Error('Upload aborted'))
            })

            // Open and send request
            xhr.open('POST', uploadUrl, true)
            if (token) {
              xhr.setRequestHeader('Authorization', `Bearer ${token}`)
            }
            xhr.send(formData)
          })
        } catch (error: unknown) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error'
          // eslint-disable-next-line no-console
          console.error(`Error uploading ${file.name}:`, errorMessage)
          setUploadProgress((prev) => {
            const updated = { ...prev }
            delete updated[file.name]
            return updated
          })
        }
      }

      if (successCount > 0) {
        toast.success(`Uploaded ${successCount} file(s)`)
        await loadObjects()
      }

      if (successCount < filesArray.length) {
        toast.error(`Failed to upload ${filesArray.length - successCount} file(s)`)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to upload files: ${errorMessage}`)
    } finally {
      setUploading(false)
      setUploadProgress({})
    }
  }

  const downloadFile = async (key: string) => {
    if (!selectedBucket) return

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/storage/${selectedBucket}/${key}`, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        const blob = await res.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = key.split('/').pop() || key
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
        toast.success('File downloaded')
      } else {
        toast.error('Failed to download file')
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to download file: ${errorMessage}`)
    }
  }

  const deleteFile = async (key: string) => {
    if (!selectedBucket) return

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/storage/${selectedBucket}/${key}`, {
        method: 'DELETE',
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        toast.success('File deleted')
        await loadObjects()
        setSelectedFiles(prev => {
          const next = new Set(prev)
          next.delete(key)
          return next
        })
      } else {
        toast.error('Failed to delete file')
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to delete file: ${errorMessage}`)
    }
  }

  const deleteSelected = async () => {
    const files = Array.from(selectedFiles)
    if (files.length === 0) return

    setLoading(true)
    let successCount = 0

    for (const key of files) {
      try {
        const token = localStorage.getItem('fluxbase-auth-token')
        const res = await fetch(`/api/v1/storage/${selectedBucket}/${key}`, {
          method: 'DELETE',
          headers: {
            Authorization: token ? `Bearer ${token}` : '',
          },
        })

        if (res.ok) {
          successCount++
        }
      } catch (error: unknown) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error'
        // eslint-disable-next-line no-console
        console.error(`Failed to delete ${key}:`, errorMessage)
      }
    }

    if (successCount > 0) {
      toast.success(`Deleted ${successCount} file(s)`)
      await loadObjects()
      setSelectedFiles(new Set())
    }

    if (successCount < files.length) {
      toast.error(`Failed to delete ${files.length - successCount} file(s)`)
    }

    setLoading(false)
    setShowDeleteConfirm(false)
  }

  const previewFileHandler = async (obj: StorageObject) => {
    if (!selectedBucket) return

    // Check if file is previewable
    const isImage = obj.content_type?.startsWith('image/')
    const isText = obj.content_type?.startsWith('text/') ||
      obj.content_type?.includes('json') ||
      obj.content_type?.includes('javascript')

    if (!isImage && !isText) {
      toast.error('Preview not available for this file type')
      return
    }

    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(`/api/v1/storage/${selectedBucket}/${obj.key}`, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })

      if (res.ok) {
        if (isImage) {
          const blob = await res.blob()
          const url = URL.createObjectURL(blob)
          setPreviewUrl(url)
        } else if (isText) {
          const text = await res.text()
          setPreviewUrl(text)
        }
        setPreviewFile(obj)
        setShowFilePreview(true)
      } else {
        toast.error('Failed to load file preview')
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to load file preview: ${errorMessage}`)
    }
  }

  const navigateToPrefix = (prefix: string) => {
    setCurrentPrefix(prefix)
    setSelectedFiles(new Set())
  }

  const createFolder = async () => {
    if (!selectedBucket || !newFolderName.trim()) {
      toast.error('Please enter a folder name')
      return
    }

    setLoading(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      // Create a folder by uploading a placeholder file with .keep extension
      const folderPath = currentPrefix
        ? `${currentPrefix}${newFolderName.trim()}/.keep`
        : `${newFolderName.trim()}/.keep`

      const formData = new FormData()
      // Create an empty blob as a placeholder file
      const emptyBlob = new Blob([''], { type: 'text/plain' })
      formData.append('file', emptyBlob, '.keep')

      // Encode each path segment separately, not the entire path
      const encodedPath = folderPath.split('/').map(segment => encodeURIComponent(segment)).join('/')

      const res = await fetch(`/api/v1/storage/${selectedBucket}/${encodedPath}`, {
        method: 'POST',
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
        body: formData,
      })

      if (res.ok) {
        toast.success(`Folder "${newFolderName}" created`)
        setShowCreateFolder(false)
        setNewFolderName('')
        await loadObjects()
      } else {
        const error = await res.text()
        toast.error(`Failed to create folder: ${error}`)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to create folder: ${errorMessage}`)
    } finally {
      setLoading(false)
    }
  }

  const openFileMetadata = async (file: StorageObject) => {
    setMetadataFile(file)
    setShowMetadata(true)
    setSignedUrl('') // Reset signed URL
  }

  const generateSignedURL = async () => {
    if (!selectedBucket || !metadataFile) {
      toast.error('No file selected')
      return
    }

    setGeneratingUrl(true)
    try {
      const token = localStorage.getItem('fluxbase-auth-token')
      const res = await fetch(
        `/api/v1/storage/${selectedBucket}/${encodeURIComponent(metadataFile.key)}/signed-url`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: token ? `Bearer ${token}` : '',
          },
          body: JSON.stringify({
            expires_in: signedUrlExpiry,
            method: 'GET',
          }),
        }
      )

      if (res.ok) {
        const data = await res.json()
        setSignedUrl(data.url)
        toast.success('Signed URL generated')
      } else {
        const error = await res.text()
        toast.error(`Failed to generate signed URL: ${error}`)
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      toast.error(`Failed to generate signed URL: ${errorMessage}`)
    } finally {
      setGeneratingUrl(false)
    }
  }

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`${label} copied to clipboard`)
  }

  const getPublicUrl = (key: string) => {
    return `${window.location.origin}/api/v1/storage/${selectedBucket}/${encodeURIComponent(key)}`
  }

  const formatJson = (text: string) => {
    try {
      const json = JSON.parse(text)
      return JSON.stringify(json, null, 2)
    } catch {
      return text
    }
  }

  const isJsonFile = (contentType?: string, fileName?: string) => {
    return contentType?.includes('json') ||
      fileName?.endsWith('.json') ||
      fileName?.endsWith('.jsonl')
  }

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      uploadFiles(e.dataTransfer.files)
    }
  }

  const toggleFileSelection = (key: string) => {
    setSelectedFiles(prev => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const getFileIcon = (contentType?: string) => {
    if (!contentType) return <File className="h-4 w-4" />
    if (contentType.startsWith('image/')) return <ImageIcon className="h-4 w-4" />
    if (contentType.includes('json')) return <FileJson className="h-4 w-4" />
    if (contentType.startsWith('text/')) return <FileText className="h-4 w-4" />
    if (contentType.includes('javascript') || contentType.includes('typescript')) {
      return <FileCode className="h-4 w-4" />
    }
    return <FileCog className="h-4 w-4" />
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // Filter and sort objects
  const filteredObjects = objects
    .filter(obj => {
      // Search filter
      if (!obj.key.toLowerCase().includes(searchQuery.toLowerCase())) {
        return false
      }

      // File type filter
      if (fileTypeFilter !== 'all') {
        const contentType = obj.content_type || ''
        if (fileTypeFilter === 'image' && !contentType.startsWith('image/')) return false
        if (fileTypeFilter === 'video' && !contentType.startsWith('video/')) return false
        if (fileTypeFilter === 'audio' && !contentType.startsWith('audio/')) return false
        if (fileTypeFilter === 'document' && !['application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument', 'text/plain'].some(t => contentType.includes(t))) return false
        if (fileTypeFilter === 'code' && !['text/javascript', 'text/typescript', 'application/json', 'text/html', 'text/css', 'text/x-python', 'text/x-go'].some(t => contentType.includes(t)) && !['.js', '.ts', '.json', '.html', '.css', '.py', '.go', '.tsx', '.jsx'].some(ext => obj.key.endsWith(ext))) return false
        if (fileTypeFilter === 'archive' && !['application/zip', 'application/x-tar', 'application/gzip'].some(t => contentType.includes(t))) return false
      }

      return true
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'name':
          return a.key.localeCompare(b.key)
        case 'size':
          return b.size - a.size
        case 'date':
          return new Date(b.last_modified).getTime() - new Date(a.last_modified).getTime()
        default:
          return 0
      }
    })

  const totalSize = objects.reduce((sum, obj) => sum + obj.size, 0)
  const selectedCount = selectedFiles.size

  return (
    <div className="flex h-full">
      {/* Sidebar - Buckets */}
      <div className="w-64 border-r bg-muted/10 p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold">Buckets</h3>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setShowCreateBucket(true)}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <ScrollArea className="h-[calc(100vh-200px)]">
          <div className="space-y-1">
            {buckets.map(bucket => (
              <div
                key={bucket}
                className={`group flex items-center justify-between p-2 rounded cursor-pointer hover:bg-muted/50 ${selectedBucket === bucket ? 'bg-muted' : ''
                  }`}
                onClick={() => {
                  setSelectedBucket(bucket)
                  setCurrentPrefix('')
                  setSelectedFiles(new Set())
                }}
              >
                <div className="flex items-center gap-2 flex-1 min-w-0">
                  <HardDrive className="h-4 w-4 flex-shrink-0" />
                  <span className="text-sm truncate">{bucket}</span>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 opacity-0 group-hover:opacity-100"
                  onClick={(e) => {
                    e.stopPropagation()
                    if (confirm(`Delete bucket "${bucket}"?`)) {
                      deleteBucket(bucket)
                    }
                  }}
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
            ))}
            {buckets.length === 0 && !loading && (
              <p className="text-sm text-muted-foreground">No buckets</p>
            )}
          </div>
        </ScrollArea>

        {selectedBucket && (
          <div className="pt-4 border-t space-y-2">
            <div className="text-xs text-muted-foreground">
              <div className="flex justify-between">
                <span>Files:</span>
                <span>{objects.length}</span>
              </div>
              <div className="flex justify-between">
                <span>Total Size:</span>
                <span>{formatBytes(totalSize)}</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {selectedBucket ? (
          <>
            {/* Toolbar */}
            <div className="border-b p-4 space-y-4">
              {/* Breadcrumb */}
              <div className="flex items-center gap-2 text-sm">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => navigateToPrefix('')}
                  className="h-7 px-2"
                >
                  <Home className="h-3 w-3" />
                </Button>
                {breadcrumbs.map((crumb, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const prefix = breadcrumbs.slice(0, i + 1).join('/') + '/'
                        navigateToPrefix(prefix)
                      }}
                      className="h-7 px-2"
                    >
                      {crumb}
                    </Button>
                  </div>
                ))}
              </div>

              {/* Actions */}
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <div className="flex-1 flex items-center gap-2">
                    <div className="relative flex-1 max-w-sm">
                      <Input
                        placeholder="Search files..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                      />
                    </div>
                    <Select value={sortBy} onValueChange={(v) => setSortBy(v as 'name' | 'size' | 'date')}>
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="name">Name</SelectItem>
                        <SelectItem value="size">Size</SelectItem>
                        <SelectItem value="date">Date</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    onClick={loadObjects}
                    disabled={loading}
                  >
                    <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                  </Button>

                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={(e) => {
                      if (e.target.files) {
                        uploadFiles(e.target.files)
                        e.target.value = ''
                      }
                    }}
                  />

                  <Button
                    variant="outline"
                    onClick={() => setShowCreateFolder(true)}
                    size="sm"
                  >
                    <FolderPlus className="h-4 w-4 mr-2" />
                    New Folder
                  </Button>

                  <Button
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading}
                    size="sm"
                  >
                    <Upload className="h-4 w-4 mr-2" />
                    {uploading ? 'Uploading...' : 'Upload'}
                  </Button>
                </div>

                {/* File Type Filter Chips */}
                <div className="flex items-center gap-2 flex-wrap">
                  <Badge
                    variant={fileTypeFilter === 'all' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('all')}
                  >
                    All Files
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'image' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('image')}
                  >
                    <ImageIcon className="h-3 w-3 mr-1" />
                    Images
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'video' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('video')}
                  >
                    <FileCode className="h-3 w-3 mr-1" />
                    Videos
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'audio' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('audio')}
                  >
                    <FileText className="h-3 w-3 mr-1" />
                    Audio
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'document' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('document')}
                  >
                    <FileText className="h-3 w-3 mr-1" />
                    Documents
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'code' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('code')}
                  >
                    <FileCode className="h-3 w-3 mr-1" />
                    Code
                  </Badge>
                  <Badge
                    variant={fileTypeFilter === 'archive' ? 'default' : 'outline'}
                    className="cursor-pointer"
                    onClick={() => setFileTypeFilter('archive')}
                  >
                    <FileCog className="h-3 w-3 mr-1" />
                    Archives
                  </Badge>
                </div>

                {filteredObjects.length > 0 && (
                  <div className="flex items-center gap-2">
                    <Checkbox
                      checked={selectedCount === filteredObjects.length && filteredObjects.length > 0}
                      onCheckedChange={(checked) => {
                        if (checked) {
                          // Select all filtered files
                          setSelectedFiles(new Set(filteredObjects.map(obj => obj.key)))
                        } else {
                          // Deselect all
                          setSelectedFiles(new Set())
                        }
                      }}
                    />
                    <span className="text-sm text-muted-foreground">
                      {selectedCount === 0 ? 'Select All' : `${selectedCount} selected`}
                    </span>
                  </div>
                )}

                {selectedCount > 0 && (
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => setShowDeleteConfirm(true)}
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete ({selectedCount})
                  </Button>
                )}
              </div>
            </div>

            {/* Upload Progress */}
            {Object.keys(uploadProgress).length > 0 && (
              <div className="border-b bg-muted/40 p-4 space-y-3">
                <div className="text-sm font-medium">Uploading files...</div>
                {Object.entries(uploadProgress).map(([filename, progress]) => (
                  <div key={filename} className="space-y-1.5">
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-muted-foreground truncate flex-1">{filename}</span>
                      <span className="ml-2 font-medium">{progress}%</span>
                    </div>
                    <div className="relative h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full bg-primary transition-all duration-300"
                        style={{ width: `${progress}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}

            {/* File List */}
            <div
              className="flex-1 p-4 overflow-auto"
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
            >
              {dragActive && (
                <div className="fixed inset-0 bg-primary/10 border-4 border-dashed border-primary flex items-center justify-center z-50">
                  <div className="text-center">
                    <Upload className="h-12 w-12 mx-auto mb-4 text-primary" />
                    <p className="text-lg font-semibold">Drop files to upload</p>
                  </div>
                </div>
              )}

              <div className="space-y-2">
                {/* Folders */}
                {prefixes.map(prefix => (
                  <Card
                    key={prefix}
                    className="p-3 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => navigateToPrefix(prefix)}
                  >
                    <div className="flex items-center gap-3">
                      <FolderOpen className="h-5 w-5 text-blue-500" />
                      <div className="flex-1 min-w-0">
                        <p className="font-medium truncate">
                          {prefix.replace(currentPrefix, '').replace('/', '')}
                        </p>
                      </div>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </div>
                  </Card>
                ))}

                {/* Files */}
                {filteredObjects.map(obj => (
                  <Card
                    key={obj.key}
                    className="p-3 hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <Checkbox
                        checked={selectedFiles.has(obj.key)}
                        onCheckedChange={() => toggleFileSelection(obj.key)}
                      />
                      {getFileIcon(obj.content_type)}
                      <div className="flex-1 min-w-0">
                        <p className="font-medium truncate">
                          {obj.key.replace(currentPrefix, '')}
                        </p>
                        <div className="flex items-center gap-3 text-xs text-muted-foreground">
                          <span>{formatBytes(obj.size)}</span>
                          <span className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {formatDistanceToNow(new Date(obj.last_modified), { addSuffix: true })}
                          </span>
                          {obj.content_type && (
                            <Badge variant="outline" className="text-xs">
                              {obj.content_type.split('/')[1]}
                            </Badge>
                          )}
                        </div>
                      </div>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => openFileMetadata(obj)}
                          title="File info"
                        >
                          <Info className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => previewFileHandler(obj)}
                          title="Preview"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => downloadFile(obj.key)}
                          title="Download"
                        >
                          <Download className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => {
                            if (confirm(`Delete "${obj.key}"?`)) {
                              deleteFile(obj.key)
                            }
                          }}
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </Card>
                ))}

                {filteredObjects.length === 0 && prefixes.length === 0 && !loading && (
                  <div className="text-center py-12 text-muted-foreground">
                    <FolderOpen className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>No files in this folder</p>
                    <p className="text-sm mt-2">
                      Drag and drop files here or click Upload
                    </p>
                  </div>
                )}
              </div>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <HardDrive className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>Select a bucket to browse files</p>
            </div>
          </div>
        )}
      </div>

      {/* Create Bucket Dialog */}
      <Dialog open={showCreateBucket} onOpenChange={setShowCreateBucket}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Bucket</DialogTitle>
            <DialogDescription>
              Enter a name for your new storage bucket
            </DialogDescription>
          </DialogHeader>
          <Input
            placeholder="my-bucket"
            value={newBucketName}
            onChange={(e) => setNewBucketName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') createBucket()
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateBucket(false)}>
              Cancel
            </Button>
            <Button onClick={createBucket} disabled={loading}>
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Create Folder Dialog */}
      <Dialog open={showCreateFolder} onOpenChange={setShowCreateFolder}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Folder</DialogTitle>
            <DialogDescription>
              Enter a name for your new folder
            </DialogDescription>
          </DialogHeader>
          <Input
            placeholder="my-folder"
            value={newFolderName}
            onChange={(e) => setNewFolderName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') createFolder()
            }}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreateFolder(false)}>
              Cancel
            </Button>
            <Button onClick={createFolder} disabled={loading}>
              Create Folder
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Files</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete {selectedCount} file(s)? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={deleteSelected} disabled={loading}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* File Preview Dialog */}
      <Dialog open={showFilePreview} onOpenChange={setShowFilePreview}>
        <DialogContent className="max-w-4xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle>{previewFile?.key}</DialogTitle>
            <DialogDescription>
              {previewFile && (
                <div className="flex items-center gap-4 text-sm">
                  <span>{formatBytes(previewFile.size)}</span>
                  <span>{previewFile.content_type}</span>
                  <span>{formatDistanceToNow(new Date(previewFile.last_modified), { addSuffix: true })}</span>
                </div>
              )}
            </DialogDescription>
          </DialogHeader>
          <ScrollArea className="max-h-[60vh]">
            {previewFile?.content_type?.startsWith('image/') ? (
              <img src={previewUrl} alt={previewFile.key} className="w-full" />
            ) : isJsonFile(previewFile?.content_type, previewFile?.key) ? (
              <div className="p-4 bg-slate-950 rounded-lg">
                <pre className="text-sm font-mono">
                  <code className="language-json text-slate-100"
                    dangerouslySetInnerHTML={{
                      __html: formatJson(previewUrl).replace(
                        /("(?:[^"\\]|\\.)*")\s*:/g,
                        '<span style="color: #94a3b8">$1</span>:'
                      ).replace(
                        /:\s*("(?:[^"\\]|\\.)*")/g,
                        ': <span style="color: #86efac">$1</span>'
                      ).replace(
                        /:\s*(\d+(?:\.\d+)?)/g,
                        ': <span style="color: #fbbf24">$1</span>'
                      ).replace(
                        /:\s*(true|false|null)/g,
                        ': <span style="color: #f472b6">$1</span>'
                      )
                    }}
                  />
                </pre>
              </div>
            ) : (
              <pre className="text-sm p-4 bg-muted/50 rounded font-mono">{previewUrl}</pre>
            )}
          </ScrollArea>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowFilePreview(false)}>
              Close
            </Button>
            {previewFile && !previewFile.content_type?.startsWith('image/') && (
              <Button
                variant="outline"
                onClick={() => {
                  const textToCopy = isJsonFile(previewFile.content_type, previewFile.key)
                    ? formatJson(previewUrl)
                    : previewUrl
                  navigator.clipboard.writeText(textToCopy)
                  toast.success('Copied to clipboard')
                }}
              >
                <Copy className="h-4 w-4 mr-2" />
                Copy
              </Button>
            )}
            {previewFile && (
              <Button onClick={() => downloadFile(previewFile.key)}>
                <Download className="h-4 w-4 mr-2" />
                Download
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* File Metadata Sheet */}
      <Sheet open={showMetadata} onOpenChange={setShowMetadata}>
        <SheetContent className="w-full sm:max-w-lg overflow-y-auto">
          <SheetHeader>
            <SheetTitle>File Details</SheetTitle>
            <SheetDescription>
              View and manage file metadata
            </SheetDescription>
          </SheetHeader>

          {metadataFile && (
            <div className="mt-6 space-y-6">
              {/* File Info */}
              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  {getFileIcon(metadataFile.content_type)}
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium truncate">
                      {metadataFile.key.replace(currentPrefix, '')}
                    </h3>
                    <p className="text-sm text-muted-foreground truncate">
                      {metadataFile.key}
                    </p>
                  </div>
                </div>

                <div className="grid gap-3">
                  <div className="flex items-center justify-between py-2 border-b">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <HardDrive className="h-4 w-4" />
                      <span>Size</span>
                    </div>
                    <span className="text-sm font-medium">{formatBytes(metadataFile.size)}</span>
                  </div>

                  <div className="flex items-center justify-between py-2 border-b">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <FileType className="h-4 w-4" />
                      <span>Type</span>
                    </div>
                    <Badge variant="outline" className="text-xs">
                      {metadataFile.content_type || 'Unknown'}
                    </Badge>
                  </div>

                  <div className="flex items-center justify-between py-2 border-b">
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Calendar className="h-4 w-4" />
                      <span>Modified</span>
                    </div>
                    <span className="text-sm font-medium">
                      {formatDistanceToNow(new Date(metadataFile.last_modified), { addSuffix: true })}
                    </span>
                  </div>

                  {metadataFile.etag && (
                    <div className="flex items-center justify-between py-2 border-b">
                      <div className="text-sm text-muted-foreground">ETag</div>
                      <span className="text-sm font-mono truncate max-w-[200px]">
                        {metadataFile.etag}
                      </span>
                    </div>
                  )}
                </div>
              </div>

              {/* Public URL */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Public URL</label>
                <div className="flex gap-2">
                  <Input
                    value={getPublicUrl(metadataFile.key)}
                    readOnly
                    className="flex-1 font-mono text-xs"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => copyToClipboard(getPublicUrl(metadataFile.key), 'URL')}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              {/* Signed URL Generator */}
              <div className="space-y-3 pt-4 border-t">
                <div className="flex items-center gap-2">
                  <Link className="h-4 w-4" />
                  <h4 className="font-medium">Generate Signed URL</h4>
                </div>
                <p className="text-sm text-muted-foreground">
                  Create a temporary URL with an expiration time for secure file sharing.
                </p>

                <div className="space-y-2">
                  <label className="text-sm font-medium">Expires In</label>
                  <Select
                    value={signedUrlExpiry.toString()}
                    onValueChange={(val) => setSignedUrlExpiry(parseInt(val))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="900">15 minutes</SelectItem>
                      <SelectItem value="1800">30 minutes</SelectItem>
                      <SelectItem value="3600">1 hour</SelectItem>
                      <SelectItem value="7200">2 hours</SelectItem>
                      <SelectItem value="21600">6 hours</SelectItem>
                      <SelectItem value="86400">24 hours</SelectItem>
                      <SelectItem value="604800">7 days</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <Button
                  onClick={generateSignedURL}
                  disabled={generatingUrl}
                  className="w-full"
                >
                  {generatingUrl ? (
                    <>
                      <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                      Generating...
                    </>
                  ) : (
                    <>
                      <Link className="h-4 w-4 mr-2" />
                      Generate Signed URL
                    </>
                  )}
                </Button>

                {signedUrl && (
                  <div className="space-y-2 p-3 bg-muted rounded-lg">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">Signed URL</span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(signedUrl, 'Signed URL')}
                      >
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </Button>
                    </div>
                    <p className="text-xs text-muted-foreground break-all font-mono">
                      {signedUrl}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      Expires in {signedUrlExpiry < 3600 ? `${signedUrlExpiry / 60} minutes` : `${signedUrlExpiry / 3600} hours`}
                    </p>
                  </div>
                )}
              </div>

              {/* Custom Metadata */}
              {metadataFile.metadata && Object.keys(metadataFile.metadata).length > 0 && (
                <div className="space-y-3 pt-4 border-t">
                  <h4 className="font-medium">Custom Metadata</h4>
                  <div className="space-y-2">
                    {Object.entries(metadataFile.metadata).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between py-2 border-b">
                        <span className="text-sm text-muted-foreground">{key}</span>
                        <span className="text-sm font-medium truncate max-w-[200px]">{value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Actions */}
              <div className="flex gap-2 pt-4 border-t">
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => downloadFile(metadataFile.key)}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Download
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => {
                    setShowMetadata(false)
                    previewFileHandler(metadataFile)
                  }}
                >
                  <Eye className="h-4 w-4 mr-2" />
                  Preview
                </Button>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}

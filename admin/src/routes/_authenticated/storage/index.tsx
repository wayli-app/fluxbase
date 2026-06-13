import { useState, useEffect, useRef, useCallback } from "react";
import { createFileRoute } from "@tanstack/react-router";
import type { AdminStorageObject, AdminBucket } from "@nimbleflux/fluxbase-sdk";
import { useFluxbaseClient } from "@nimbleflux/fluxbase-sdk-react";
import {
  HardDrive,
  File,
  Image as ImageIcon,
  FileText,
  FileJson,
  FileCode,
  FileCog,
} from "lucide-react";
import { toast } from "sonner";
import { useImpersonationStore } from "@/stores/impersonation-store";
import { useTenantStore } from "@/stores/tenant-store";
import {
  CreateBucketDialog,
  CreateFolderDialog,
  DeleteConfirmDialog,
  FilePreviewDialog,
  DeleteBucketDialog,
  DeleteFileDialog,
  FileMetadataSheet,
  BucketList,
  FileList,
  UploadProgress,
  Toolbar,
} from "@/components/storage";
import { PageHeader } from "@/components/layout/page-header";

export const Route = createFileRoute("/_authenticated/storage/")({
  component: StorageBrowser,
});

function StorageBrowser() {
  const client = useFluxbaseClient();
  const currentTenantId = useTenantStore((state) => state.currentTenant?.id);

  const [buckets, setBuckets] = useState<AdminBucket[]>([]);
  const [selectedBucket, setSelectedBucket] = useState<string>("");
  const [currentPrefix, setCurrentPrefix] = useState<string>("");
  const [objects, setObjects] = useState<AdminStorageObject[]>([]);
  const [prefixes, setPrefixes] = useState<string[]>([]);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>(
    {},
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<"name" | "size" | "date">("name");
  const [fileTypeFilter, setFileTypeFilter] = useState<string>("all");
  const [isCreateBucketOpen, setIsCreateBucketOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isFilePreviewOpen, setIsFilePreviewOpen] = useState(false);
  const [isCreateFolderOpen, setIsCreateFolderOpen] = useState(false);
  const [isDeleteBucketConfirmOpen, setIsDeleteBucketConfirmOpen] =
    useState(false);
  const [deletingBucketName, setDeletingBucketName] = useState<string | null>(
    null,
  );
  const [isDeleteFileConfirmOpen, setIsDeleteFileConfirmOpen] = useState(false);
  const [deletingFilePath, setDeletingFilePath] = useState<string | null>(null);
  const [previewFile, setPreviewFile] = useState<AdminStorageObject | null>(
    null,
  );
  const [previewUrl, setPreviewUrl] = useState<string>("");
  const [newBucketName, setNewBucketName] = useState("");
  const [newFolderName, setNewFolderName] = useState("");
  const [dragActive, setDragActive] = useState(false);
  const [isMetadataOpen, setIsMetadataOpen] = useState(false);
  const [metadataFile, setMetadataFile] = useState<AdminStorageObject | null>(
    null,
  );
  const [signedUrl, setSignedUrl] = useState<string>("");
  const [signedUrlExpiry, setSignedUrlExpiry] = useState<number>(3600);
  const [generatingUrl, setGeneratingUrl] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const breadcrumbs = currentPrefix
    ? currentPrefix.split("/").filter(Boolean)
    : [];

  const loadBuckets = useCallback(async () => {
    setLoading(true);
    try {
      const { data, error } = await client.admin.storage.listBuckets();
      if (error) throw error;
      setBuckets(data?.buckets || []);
      if (data?.buckets && data.buckets.length > 0) {
        setSelectedBucket((prev) => prev || data.buckets[0].name);
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to load buckets: ${errorMessage}`);
    } finally {
      setLoading(false);
    }
  }, [client]);

  const loadObjects = useCallback(async () => {
    if (!selectedBucket) return;
    setLoading(true);
    try {
      const { data, error } = await client.admin.storage.listObjects(
        selectedBucket,
        currentPrefix || undefined,
        "/",
      );
      if (error) throw error;
      setObjects(data?.objects || []);
      setPrefixes(data?.prefixes || []);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to load files: ${errorMessage}`);
    } finally {
      setLoading(false);
    }
  }, [selectedBucket, currentPrefix, client]);

  useEffect(() => {
    setSelectedBucket("");
    setObjects([]);
    setPrefixes([]);
    setCurrentPrefix("");
    setSelectedFiles(new Set());
    loadBuckets();
  }, [loadBuckets, currentTenantId]);

  useEffect(() => {
    if (selectedBucket) {
      loadObjects();
    }
  }, [selectedBucket, currentPrefix, loadObjects]);

  const createBucket = async () => {
    if (!newBucketName.trim()) {
      toast.error("Bucket name is required");
      return;
    }
    setLoading(true);
    try {
      const { error } = await client.admin.storage.createBucket(newBucketName);
      if (error) throw error;
      toast.success(`Bucket "${newBucketName}" created`);
      setIsCreateBucketOpen(false);
      setNewBucketName("");
      await loadBuckets();
      setSelectedBucket(newBucketName);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to create bucket: ${errorMessage}`);
    } finally {
      setLoading(false);
    }
  };

  const deleteBucket = async (bucketName: string) => {
    setLoading(true);
    try {
      const { error } = await client.admin.storage.deleteBucket(bucketName);
      if (error) throw error;
      toast.success(`Bucket "${bucketName}" deleted`);
      await loadBuckets();
      if (selectedBucket === bucketName) {
        setSelectedBucket(buckets[0]?.name || "");
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to delete bucket: ${errorMessage}`);
    } finally {
      setLoading(false);
    }
  };

  const uploadFiles = async (files: FileList | File[]) => {
    if (!selectedBucket) {
      toast.error("Please select a bucket first");
      return;
    }
    setUploading(true);
    const filesArray = Array.from(files);
    let successCount = 0;

    try {
      for (const file of filesArray) {
        const key = currentPrefix ? `${currentPrefix}${file.name}` : file.name;
        const {
          isImpersonating: isImpersonatingNow,
          impersonationToken: impersonationTokenNow,
        } = useImpersonationStore.getState();
        const token =
          isImpersonatingNow && impersonationTokenNow
            ? impersonationTokenNow
            : localStorage.getItem("fluxbase-auth-token");

        setUploadProgress((prev) => ({ ...prev, [file.name]: 0 }));

        try {
          const formData = new FormData();
          formData.append("file", file);
          const uploadUrl = `/api/v1/storage/${selectedBucket}/${encodeURIComponent(key)}`;

          await new Promise<void>((resolve, reject) => {
            const xhr = new XMLHttpRequest();

            xhr.upload.addEventListener("progress", (e) => {
              if (e.lengthComputable) {
                const percentComplete = Math.round((e.loaded / e.total) * 100);
                setUploadProgress((prev) => ({
                  ...prev,
                  [file.name]: percentComplete,
                }));
              }
            });

            xhr.addEventListener("load", () => {
              if (xhr.status >= 200 && xhr.status < 300) {
                setUploadProgress((prev) => ({ ...prev, [file.name]: 100 }));
                setTimeout(() => {
                  setUploadProgress((prev) => {
                    const updated = { ...prev };
                    delete updated[file.name];
                    return updated;
                  });
                }, 500);
                successCount++;
                resolve();
              } else {
                setUploadProgress((prev) => {
                  const updated = { ...prev };
                  delete updated[file.name];
                  return updated;
                });
                reject(new Error(`Upload failed with status ${xhr.status}`));
              }
            });

            xhr.addEventListener("error", () => {
              setUploadProgress((prev) => {
                const updated = { ...prev };
                delete updated[file.name];
                return updated;
              });
              reject(new Error("Network error during upload"));
            });

            xhr.addEventListener("abort", () => {
              setUploadProgress((prev) => {
                const updated = { ...prev };
                delete updated[file.name];
                return updated;
              });
              reject(new Error("Upload aborted"));
            });

            xhr.open("POST", uploadUrl, true);
            if (token) {
              xhr.setRequestHeader("Authorization", `Bearer ${token}`);
            }
            const currentTenant = useTenantStore.getState().currentTenant;
            if (currentTenant?.id) {
              xhr.setRequestHeader("X-FB-Tenant", currentTenant.id);
            }
            xhr.send(formData);
          });
        } catch {
          setUploadProgress((prev) => {
            const updated = { ...prev };
            delete updated[file.name];
            return updated;
          });
        }
      }

      if (successCount > 0) {
        toast.success(`Uploaded ${successCount} file(s)`);
        await loadObjects();
      }
      if (successCount < filesArray.length) {
        toast.error(
          `Failed to upload ${filesArray.length - successCount} file(s)`,
        );
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to upload files: ${errorMessage}`);
    } finally {
      setUploading(false);
      setUploadProgress({});
    }
  };

  const downloadFile = async (key: string) => {
    if (!selectedBucket) return;
    try {
      const { data: blob, error } = await client.admin.storage.downloadObject(
        selectedBucket,
        key,
      );
      if (error) throw error;
      if (!blob) throw new Error("No data received");
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = key.split("/").pop() || key;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success("File downloaded");
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to download file: ${errorMessage}`);
    }
  };

  const deleteFile = async (key: string) => {
    if (!selectedBucket) return;
    try {
      const { error } = await client.admin.storage.deleteObject(
        selectedBucket,
        key,
      );
      if (error) throw error;
      toast.success("File deleted");
      await loadObjects();
      setSelectedFiles((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to delete file: ${errorMessage}`);
    }
  };

  const deleteSelected = async () => {
    const files = Array.from(selectedFiles);
    if (files.length === 0) return;
    setLoading(true);
    let successCount = 0;

    for (const key of files) {
      try {
        const { error } = await client.admin.storage.deleteObject(
          selectedBucket,
          key,
        );
        if (error) throw error;
        successCount++;
      } catch {
        // Error already handled
      }
    }

    if (successCount > 0) {
      toast.success(`Deleted ${successCount} file(s)`);
      await loadObjects();
      setSelectedFiles(new Set());
    }
    if (successCount < files.length) {
      toast.error(`Failed to delete ${files.length - successCount} file(s)`);
    }
    setLoading(false);
    setIsDeleteConfirmOpen(false);
  };

  const previewFileHandler = async (obj: AdminStorageObject) => {
    if (!selectedBucket) return;

    const isImage = obj.mime_type?.startsWith("image/");
    const isText =
      obj.mime_type?.startsWith("text/") ||
      obj.mime_type?.includes("json") ||
      obj.mime_type?.includes("javascript");

    if (!isImage && !isText) {
      toast.error("Preview not available for this file type");
      return;
    }

    try {
      const { data: blob, error } = await client.admin.storage.downloadObject(
        selectedBucket,
        obj.path,
      );
      if (error) throw error;
      if (!blob) throw new Error("No data received");
      if (isImage) {
        const url = URL.createObjectURL(blob);
        setPreviewUrl(url);
      } else if (isText) {
        const text = await blob.text();
        setPreviewUrl(text);
      }
      setPreviewFile(obj);
      setIsFilePreviewOpen(true);
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to load file preview: ${errorMessage}`);
    }
  };

  const navigateToPrefix = (prefix: string) => {
    setCurrentPrefix(prefix);
    setSelectedFiles(new Set());
  };

  const createFolder = async () => {
    if (!selectedBucket || !newFolderName.trim()) {
      toast.error("Please enter a folder name");
      return;
    }
    setLoading(true);
    try {
      const folderPath = currentPrefix
        ? `${currentPrefix}${newFolderName.trim()}/.keep`
        : `${newFolderName.trim()}/.keep`;
      const { error } = await client.admin.storage.createFolder(
        selectedBucket,
        folderPath,
      );
      if (error) throw error;
      toast.success(`Folder "${newFolderName}" created`);
      setIsCreateFolderOpen(false);
      setNewFolderName("");
      await loadObjects();
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to create folder: ${errorMessage}`);
    } finally {
      setLoading(false);
    }
  };

  const openFileMetadata = async (file: AdminStorageObject) => {
    setMetadataFile(file);
    setIsMetadataOpen(true);
    setSignedUrl("");
  };

  const generateSignedURL = async () => {
    if (!selectedBucket || !metadataFile) {
      toast.error("No file selected");
      return;
    }
    setGeneratingUrl(true);
    try {
      const { data, error } = await client.admin.storage.generateSignedUrl(
        selectedBucket,
        metadataFile.path,
        signedUrlExpiry,
      );
      if (error) throw error;
      if (!data) throw new Error("No data received");
      setSignedUrl(data.url);
      toast.success("Signed URL generated");
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error";
      toast.error(`Failed to generate signed URL: ${errorMessage}`);
    } finally {
      setGeneratingUrl(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    toast.success(`${label} copied to clipboard`);
  };

  const getPublicUrl = (key: string) => {
    return `${window.location.origin}/api/v1/storage/${selectedBucket}/${encodeURIComponent(key)}`;
  };

  const escapeHtml = (text: string): string => {
    const htmlEntities: Record<string, string> = {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    };
    return text.replace(/[&<>"']/g, (char) => htmlEntities[char]);
  };

  const formatJson = (text: string) => {
    try {
      const json = JSON.parse(text);
      return JSON.stringify(json, null, 2);
    } catch {
      return text;
    }
  };

  const formatJsonWithHighlighting = (text: string): string => {
    const formatted = formatJson(text);
    const escaped = escapeHtml(formatted);
    return escaped
      .replace(
        /(&quot;(?:[^&]|&(?!quot;))*?&quot;)\s*:/g,
        '<span style="color: #94a3b8">$1</span>:',
      )
      .replace(
        /:\s*(&quot;(?:[^&]|&(?!quot;))*?&quot;)/g,
        ': <span style="color: #86efac">$1</span>',
      )
      .replace(
        /:\s*(\d+(?:\.\d+)?)/g,
        ': <span style="color: #fbbf24">$1</span>',
      )
      .replace(
        /:\s*(true|false|null)/g,
        ': <span style="color: #f472b6">$1</span>',
      );
  };

  const isJsonFile = (contentType?: string, fileName?: string): boolean => {
    return !!(
      contentType?.includes("json") ||
      fileName?.endsWith(".json") ||
      fileName?.endsWith(".jsonl")
    );
  };

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      uploadFiles(e.dataTransfer.files);
    }
  };

  const toggleFileSelection = (key: string) => {
    setSelectedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const getFileIcon = (contentType?: string) => {
    if (!contentType) return <File className="h-4 w-4" />;
    if (contentType.startsWith("image/"))
      return <ImageIcon className="h-4 w-4" />;
    if (contentType.includes("json")) return <FileJson className="h-4 w-4" />;
    if (contentType.startsWith("text/"))
      return <FileText className="h-4 w-4" />;
    if (
      contentType.includes("javascript") ||
      contentType.includes("typescript")
    )
      return <FileCode className="h-4 w-4" />;
    return <FileCog className="h-4 w-4" />;
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const filteredObjects = objects
    .filter((obj) => {
      if (!obj.path.toLowerCase().includes(searchQuery.toLowerCase()))
        return false;

      if (fileTypeFilter !== "all") {
        const contentType = obj.mime_type || "";
        if (fileTypeFilter === "image" && !contentType.startsWith("image/"))
          return false;
        if (fileTypeFilter === "video" && !contentType.startsWith("video/"))
          return false;
        if (fileTypeFilter === "audio" && !contentType.startsWith("audio/"))
          return false;
        if (
          fileTypeFilter === "document" &&
          ![
            "application/pdf",
            "application/msword",
            "application/vnd.openxmlformats-officedocument",
            "text/plain",
          ].some((t) => contentType.includes(t))
        )
          return false;
        if (
          fileTypeFilter === "code" &&
          ![
            "text/javascript",
            "text/typescript",
            "application/json",
            "text/html",
            "text/css",
            "text/x-python",
            "text/x-go",
          ].some((t) => contentType.includes(t)) &&
          ![
            ".js",
            ".ts",
            ".json",
            ".html",
            ".css",
            ".py",
            ".go",
            ".tsx",
            ".jsx",
          ].some((ext) => obj.path.endsWith(ext))
        )
          return false;
        if (
          fileTypeFilter === "archive" &&
          !["application/zip", "application/x-tar", "application/gzip"].some(
            (t) => contentType.includes(t),
          )
        )
          return false;
      }
      return true;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case "name":
          return a.path.localeCompare(b.path);
        case "size":
          return b.size - a.size;
        case "date":
          return (
            new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
          );
        default:
          return 0;
      }
    });

  const totalSize = objects.reduce((sum, obj) => sum + obj.size, 0);
  const selectedCount = selectedFiles.size;

  const handleSelectAll = () =>
    setSelectedFiles(new Set(filteredObjects.map((obj) => obj.path)));
  const handleDeselectAll = () => setSelectedFiles(new Set());

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<HardDrive />}
        title="Storage"
        description="Manage files and buckets in your storage backend"
      />

      <div className="flex flex-1 overflow-hidden p-6">
        <BucketList
          buckets={buckets}
          selectedBucket={selectedBucket}
          onSelectBucket={(name) => {
            setSelectedBucket(name);
            setCurrentPrefix("");
            setSelectedFiles(new Set());
          }}
          onDeleteBucket={(name) => {
            setDeletingBucketName(name);
            setIsDeleteBucketConfirmOpen(true);
          }}
          objectCount={objects.length}
          totalSize={totalSize}
          formatBytes={formatBytes}
          onCreateBucket={() => setIsCreateBucketOpen(true)}
        />

        <div className="flex flex-1 flex-col">
          {selectedBucket ? (
            <>
              <Toolbar
                breadcrumbs={breadcrumbs}
                searchQuery={searchQuery}
                sortBy={sortBy}
                fileTypeFilter={fileTypeFilter}
                selectedCount={selectedCount}
                filteredCount={filteredObjects.length}
                uploading={uploading}
                loading={loading}
                onNavigateToPrefix={navigateToPrefix}
                onSearchChange={setSearchQuery}
                onSortChange={setSortBy}
                onFileTypeFilterChange={setFileTypeFilter}
                onSelectAll={handleSelectAll}
                onDeselectAll={handleDeselectAll}
                onRefresh={loadObjects}
                onCreateFolder={() => setIsCreateFolderOpen(true)}
                onUpload={() => fileInputRef.current?.click()}
                onDeleteSelected={() => setIsDeleteConfirmOpen(true)}
                fileInputRef={fileInputRef}
                onFileInputChange={uploadFiles}
              />

              <UploadProgress uploadProgress={uploadProgress} />

              <FileList
                prefixes={prefixes}
                objects={filteredObjects}
                selectedFiles={selectedFiles}
                currentPrefix={currentPrefix}
                searchQuery={searchQuery}
                loading={loading}
                dragActive={dragActive}
                onNavigateToPrefix={navigateToPrefix}
                onToggleFileSelection={toggleFileSelection}
                onFileMetadata={openFileMetadata}
                onFilePreview={previewFileHandler}
                onFileDownload={downloadFile}
                onFileDelete={(path) => {
                  setDeletingFilePath(path);
                  setIsDeleteFileConfirmOpen(true);
                }}
                onDragEnter={handleDrag}
                onDragLeave={handleDrag}
                onDragOver={handleDrag}
                onDrop={handleDrop}
                formatBytes={formatBytes}
                getFileIcon={getFileIcon}
              />
            </>
          ) : (
            <div className="text-muted-foreground flex flex-1 items-center justify-center">
              <div className="text-center">
                <HardDrive className="mx-auto mb-4 h-12 w-12 opacity-50" />
                <p>Select a bucket to browse files</p>
              </div>
            </div>
          )}
        </div>
      </div>

      <CreateBucketDialog
        isOpen={isCreateBucketOpen}
        onOpenChange={setIsCreateBucketOpen}
        bucketName={newBucketName}
        onBucketNameChange={setNewBucketName}
        onCreate={createBucket}
        loading={loading}
      />

      <CreateFolderDialog
        isOpen={isCreateFolderOpen}
        onOpenChange={setIsCreateFolderOpen}
        folderName={newFolderName}
        onFolderNameChange={setNewFolderName}
        onCreate={createFolder}
        loading={loading}
      />

      <DeleteConfirmDialog
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        selectedCount={selectedCount}
        onDelete={deleteSelected}
        loading={loading}
      />

      <FilePreviewDialog
        isOpen={isFilePreviewOpen}
        onOpenChange={setIsFilePreviewOpen}
        file={previewFile}
        previewUrl={previewUrl}
        onDownload={() => previewFile && downloadFile(previewFile.path)}
        formatBytes={formatBytes}
        formatJson={formatJson}
        formatJsonWithHighlighting={formatJsonWithHighlighting}
        isJsonFile={isJsonFile}
      />

      <DeleteBucketDialog
        isOpen={isDeleteBucketConfirmOpen}
        onOpenChange={setIsDeleteBucketConfirmOpen}
        bucketName={deletingBucketName}
        onDelete={() => deletingBucketName && deleteBucket(deletingBucketName)}
      />

      <DeleteFileDialog
        isOpen={isDeleteFileConfirmOpen}
        onOpenChange={setIsDeleteFileConfirmOpen}
        filePath={deletingFilePath}
        onDelete={() => deletingFilePath && deleteFile(deletingFilePath)}
      />

      <FileMetadataSheet
        isOpen={isMetadataOpen}
        onOpenChange={setIsMetadataOpen}
        file={metadataFile}
        currentPrefix={currentPrefix}
        signedUrl={signedUrl}
        signedUrlExpiry={signedUrlExpiry}
        generatingUrl={generatingUrl}
        onSignedUrlExpiryChange={setSignedUrlExpiry}
        onGenerateSignedUrl={generateSignedURL}
        onDownload={() => metadataFile && downloadFile(metadataFile.path)}
        onPreview={() => metadataFile && previewFileHandler(metadataFile)}
        getPublicUrl={getPublicUrl}
        formatBytes={formatBytes}
        getFileIcon={getFileIcon}
        copyToClipboard={copyToClipboard}
      />
    </div>
  );
}

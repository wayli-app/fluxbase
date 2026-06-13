import { useState } from "react";
import { format } from "date-fns";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
  Building2,
  Plus,
  Trash2,
  Search,
  Users,
  CheckCircle,
  Pencil,
  Copy,
  Download,
  Wrench,
  Mail,
  Key,
  Database,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import {
  tenantsApi,
  type CreateTenantRequest,
  type Tenant,
  type UpdateTenantRequest as UpdateTenantReq,
  type CreateTenantResponse,
} from "@/lib/api";
import { requireInstanceAdmin } from "@/lib/route-guards";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/layout/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export const Route = createFileRoute("/_authenticated/tenants/")({
  beforeLoad: () => {
    requireInstanceAdmin();
  },
  component: TenantsPage,
});

function TenantsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [newTenant, setNewTenant] = useState({
    name: "",
    slug: "",
    slugManuallyEdited: false,
    auto_generate_keys: true,
    admin_email: "",
    send_keys_to_email: false,
    db_mode: "auto" as "auto" | "existing",
    db_name: "",
  });
  const [editTenant, setEditTenant] = useState({ name: "" });
  const [keysDialogOpen, setKeysDialogOpen] = useState(false);
  const [createdResponse, setCreatedResponse] =
    useState<CreateTenantResponse | null>(null);

  const { data: tenants, isLoading } = useQuery({
    queryKey: ["tenants"],
    queryFn: tenantsApi.list,
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateTenantRequest) => tenantsApi.create(data),
    onSuccess: (response: CreateTenantResponse) => {
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
      setCreateDialogOpen(false);

      // If keys were generated, show them in a dialog
      if (response.anon_key || response.service_key) {
        setCreatedResponse(response);
        setKeysDialogOpen(true);
      } else {
        toast.success("Tenant created successfully");
      }

      // Reset form
      setNewTenant({
        name: "",
        slug: "",
        slugManuallyEdited: false,
        auto_generate_keys: true,
        admin_email: "",
        send_keys_to_email: false,
        db_mode: "auto",
        db_name: "",
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tenantsApi.delete(id),
    onSuccess: (_, id) => {
      queryClient.setQueryData<Tenant[]>(
        ["tenants"],
        (old) => old?.filter((t) => t.id !== id) || [],
      );
      toast.success("Tenant deleted successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete tenant: ${error.message}`);
    },
  });

  const repairMutation = useMutation({
    mutationFn: (id: string) => tenantsApi.repair(id),
    onSuccess: () => {
      toast.success("Tenant database repaired successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to repair tenant: ${error.message}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTenantReq }) =>
      tenantsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
      toast.success("Tenant updated successfully");
      setEditDialogOpen(false);
      setEditingTenant(null);
      setEditTenant({ name: "" });
    },
    onError: (error: Error) => {
      toast.error(`Failed to update tenant: ${error.message}`);
    },
  });

  const handleEditTenant = (tenant: Tenant) => {
    setEditingTenant(tenant);
    setEditTenant({ name: tenant.name });
    setEditDialogOpen(true);
  };

  const handleUpdateTenant = () => {
    if (!editingTenant) return;
    if (!editTenant.name.trim()) {
      toast.error("Name is required");
      return;
    }
    updateMutation.mutate({
      id: editingTenant.id,
      data: {
        name: editTenant.name.trim(),
      },
    });
  };

  const handleCreateTenant = () => {
    if (!newTenant.name.trim()) {
      toast.error("Name is required");
      return;
    }
    if (!newTenant.slug.trim()) {
      toast.error("Slug is required");
      return;
    }
    if (!/^[a-z][a-z0-9-]*[a-z0-9]$/.test(newTenant.slug)) {
      toast.error(
        "Slug must start with a lowercase letter, contain only lowercase letters, numbers, and hyphens, and end with a letter or number",
      );
      return;
    }
    if (
      newTenant.admin_email &&
      !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newTenant.admin_email)
    ) {
      toast.error("Invalid admin email address");
      return;
    }
    createMutation.mutate({
      name: newTenant.name.trim(),
      slug: newTenant.slug.trim().toLowerCase(),
      auto_generate_keys: newTenant.auto_generate_keys,
      admin_email: newTenant.admin_email.trim() || undefined,
      send_keys_to_email: newTenant.send_keys_to_email,
      db_mode: newTenant.db_mode === "existing" ? "existing" : undefined,
      db_name:
        newTenant.db_mode === "existing"
          ? newTenant.db_name.trim() || undefined
          : undefined,
    });
  };

  const generateSlug = (name: string) => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  };

  const filteredTenants = tenants?.filter(
    (tenant) =>
      tenant.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      tenant.slug.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        icon={<Building2 />}
        title="Tenants"
        description="Manage multi-tenant organizations"
        actions={
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Tenant
          </Button>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <div className="flex flex-col gap-6">
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Total Tenants
                </CardTitle>
                <Building2 className="text-muted-foreground h-4 w-4" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{tenants?.length || 0}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Default Tenant
                </CardTitle>
                <CheckCircle className="text-muted-foreground h-4 w-4" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {tenants?.filter((t) => t.is_default).length || 0}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  Custom Tenants
                </CardTitle>
                <Users className="text-muted-foreground h-4 w-4" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {tenants?.filter((t) => !t.is_default).length || 0}
                </div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Tenants</CardTitle>
                  <CardDescription>All tenants in the system</CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="mb-4">
                <div className="relative">
                  <Search className="text-muted-foreground absolute top-2.5 left-2 h-4 w-4" />
                  <Input
                    placeholder="Search by name or slug..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-8"
                  />
                </div>
              </div>

              {isLoading ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Slug</TableHead>
                      <TableHead>Default</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {Array(3)
                      .fill(0)
                      .map((_, i) => (
                        <TableRow key={i}>
                          <TableCell>
                            <Skeleton className="h-4 w-32" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-24" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-5 w-16" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-24" />
                          </TableCell>
                          <TableCell className="text-right">
                            <Skeleton className="ml-auto h-8 w-8" />
                          </TableCell>
                        </TableRow>
                      ))}
                  </TableBody>
                </Table>
              ) : filteredTenants && filteredTenants.length > 0 ? (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Slug</TableHead>
                      <TableHead>Default</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredTenants.map((tenant) => (
                      <TableRow
                        key={tenant.id}
                        className="cursor-pointer"
                        onClick={() =>
                          navigate({
                            to: "/tenants/$tenantId",
                            params: { tenantId: tenant.id },
                          })
                        }
                      >
                        <TableCell className="font-medium">
                          {tenant.name}
                        </TableCell>
                        <TableCell>
                          <code className="text-xs">{tenant.slug}</code>
                        </TableCell>
                        <TableCell>
                          {tenant.is_default ? (
                            <Badge variant="default">Default</Badge>
                          ) : (
                            <Badge variant="outline">Custom</Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {format(new Date(tenant.created_at), "MMM d, yyyy")}
                        </TableCell>
                        <TableCell
                          className="text-right"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleEditTenant(tenant)}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                            {!tenant.is_default && (
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    disabled={repairMutation.isPending}
                                  >
                                    {repairMutation.isPending ? (
                                      <Loader2 className="h-4 w-4 animate-spin" />
                                    ) : (
                                      <Wrench className="h-4 w-4" />
                                    )}
                                  </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>
                                      Repair Tenant Database
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                      This will re-bootstrap, re-apply internal
                                      schemas, and re-setup FDW on "
                                      {tenant.name}"'s database. Existing data is
                                      preserved.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>
                                      Cancel
                                    </AlertDialogCancel>
                                    <AlertDialogAction
                                      disabled={repairMutation.isPending}
                                      onClick={() =>
                                        repairMutation.mutate(tenant.id)
                                      }
                                    >
                                      {repairMutation.isPending && (
                                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                      )}
                                      {repairMutation.isPending ? "Repairing..." : "Repair"}
                                    </AlertDialogAction>
                                  </AlertDialogFooter>
                                </AlertDialogContent>
                              </AlertDialog>
                            )}
                            {!tenant.is_default && (
                              <AlertDialog>
                                <AlertDialogTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </AlertDialogTrigger>
                                <AlertDialogContent>
                                  <AlertDialogHeader>
                                    <AlertDialogTitle>
                                      Delete Tenant
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                      Are you sure you want to delete "
                                      {tenant.name}"? This action cannot be
                                      undone.
                                    </AlertDialogDescription>
                                  </AlertDialogHeader>
                                  <AlertDialogFooter>
                                    <AlertDialogCancel>
                                      Cancel
                                    </AlertDialogCancel>
                                    <AlertDialogAction
                                      onClick={() =>
                                        deleteMutation.mutate(tenant.id)
                                      }
                                      className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                    >
                                      Delete
                                    </AlertDialogAction>
                                  </AlertDialogFooter>
                                </AlertDialogContent>
                              </AlertDialog>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <div className="flex flex-col items-center justify-center py-12 text-center">
                  <Building2 className="text-muted-foreground mb-4 h-12 w-12" />
                  <p className="text-muted-foreground">
                    {searchQuery
                      ? "No tenants match your search"
                      : "No tenants yet"}
                  </p>
                  {!searchQuery && (
                    <Button
                      onClick={() => setCreateDialogOpen(true)}
                      variant="outline"
                      className="mt-4"
                    >
                      Create Your First Tenant
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create Tenant</DialogTitle>
            <DialogDescription>
              Create a new tenant organization. Anon and service keys will be
              auto-generated.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">
                Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="name"
                placeholder="Acme Corporation"
                value={newTenant.name}
                onChange={(e) => {
                  const name = e.target.value;
                  setNewTenant({
                    ...newTenant,
                    name,
                    slug: newTenant.slugManuallyEdited
                      ? newTenant.slug
                      : generateSlug(name),
                  });
                }}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="slug">
                Slug <span className="text-destructive">*</span>
              </Label>
              <Input
                id="slug"
                placeholder="acme-corporation"
                value={newTenant.slug}
                onChange={(e) =>
                  setNewTenant({
                    ...newTenant,
                    slug: e.target.value,
                    slugManuallyEdited: true,
                  })
                }
              />
              <p className="text-muted-foreground text-xs">
                Lowercase letters, numbers, and hyphens only. Must start with a
                letter.
              </p>
            </div>

            <div className="border-t pt-4 mt-2">
              <div className="flex items-center gap-2 mb-2">
                <Database className="h-4 w-4 text-muted-foreground" />
                <Label className="font-medium">Database</Label>
              </div>
              <RadioGroup
                value={newTenant.db_mode}
                onValueChange={(value) =>
                  setNewTenant({
                    ...newTenant,
                    db_mode: value as "auto" | "existing",
                  })
                }
                className="gap-2"
              >
                <div className="flex items-center gap-2">
                  <RadioGroupItem value="auto" id="db-auto" />
                  <Label htmlFor="db-auto" className="text-sm font-normal">
                    Create new database
                  </Label>
                </div>
                <div className="flex items-center gap-2">
                  <RadioGroupItem value="existing" id="db-existing" />
                  <Label htmlFor="db-existing" className="text-sm font-normal">
                    Use existing database
                  </Label>
                </div>
              </RadioGroup>
              {newTenant.db_mode === "existing" && (
                <div className="mt-2 ml-6">
                  <Input
                    placeholder="database_name"
                    value={newTenant.db_name}
                    onChange={(e) =>
                      setNewTenant({
                        ...newTenant,
                        db_name: e.target.value,
                      })
                    }
                  />
                  <p className="text-muted-foreground text-xs mt-1">
                    The database must already exist on the same PostgreSQL
                    server. Schemas and roles will be bootstrapped
                    automatically.
                  </p>
                </div>
              )}
            </div>

            <div className="border-t pt-4 mt-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Key className="h-4 w-4 text-muted-foreground" />
                  <Label htmlFor="auto-generate-keys" className="font-medium">
                    Auto-generate keys
                  </Label>
                </div>
                <input
                  id="auto-generate-keys"
                  type="checkbox"
                  checked={newTenant.auto_generate_keys}
                  onChange={(e) =>
                    setNewTenant({
                      ...newTenant,
                      auto_generate_keys: e.target.checked,
                    })
                  }
                  className="h-4 w-4"
                />
              </div>
              {newTenant.auto_generate_keys && (
                <p className="text-muted-foreground text-xs mt-1">
                  Creates an anon key and a service key for this tenant
                </p>
              )}
            </div>

            <div className="border-t pt-4 mt-2">
              <div className="flex items-center gap-2 mb-2">
                <Mail className="h-4 w-4 text-muted-foreground" />
                <Label className="font-medium">Tenant Admin (Optional)</Label>
              </div>
              <Input
                id="admin-email"
                type="email"
                placeholder="admin@example.com"
                value={newTenant.admin_email}
                onChange={(e) =>
                  setNewTenant({ ...newTenant, admin_email: e.target.value })
                }
              />
              <p className="text-muted-foreground text-xs mt-1">
                Invite an admin by email. They'll receive setup instructions.
              </p>

              {newTenant.admin_email && newTenant.auto_generate_keys && (
                <div className="flex items-center gap-2 mt-3">
                  <input
                    id="send-keys-to-email"
                    type="checkbox"
                    checked={newTenant.send_keys_to_email}
                    onChange={(e) =>
                      setNewTenant({
                        ...newTenant,
                        send_keys_to_email: e.target.checked,
                      })
                    }
                    className="h-4 w-4"
                  />
                  <Label htmlFor="send-keys-to-email" className="text-sm">
                    Send keys to this email
                  </Label>
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateTenant}
              disabled={createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create Tenant"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Keys Display Dialog */}
      <Dialog open={keysDialogOpen} onOpenChange={setKeysDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-500" />
              Tenant Created Successfully
            </DialogTitle>
            <DialogDescription>
              Save these API keys - they will only be shown once.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            {createdResponse?.anon_key && (
              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label className="font-medium">Anon Key</Label>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      navigator.clipboard.writeText(createdResponse.anon_key!);
                      toast.success("Anon key copied to clipboard");
                    }}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                <code className="bg-muted p-2 rounded text-xs break-all">
                  {createdResponse.anon_key}
                </code>
              </div>
            )}
            {createdResponse?.service_key && (
              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label className="font-medium">Service Key</Label>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      navigator.clipboard.writeText(
                        createdResponse.service_key!,
                      );
                      toast.success("Service key copied to clipboard");
                    }}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                <code className="bg-muted p-2 rounded text-xs break-all">
                  {createdResponse.service_key}
                </code>
              </div>
            )}
            {createdResponse?.invitation_sent && (
              <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg p-3 mt-2">
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  Invitation email sent to{" "}
                  <strong>{createdResponse.invitation_email}</strong>
                </p>
              </div>
            )}
          </div>
          <DialogFooter className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => {
                const keys = [];
                if (createdResponse?.anon_key)
                  keys.push(`ANON_KEY=${createdResponse.anon_key}`);
                if (createdResponse?.service_key)
                  keys.push(`SERVICE_KEY=${createdResponse.service_key}`);
                const blob = new Blob([keys.join("\n")], {
                  type: "text/plain",
                });
                const url = URL.createObjectURL(blob);
                const a = document.createElement("a");
                a.href = url;
                a.download = `${createdResponse?.tenant.slug}-keys.txt`;
                a.click();
                URL.revokeObjectURL(url);
                toast.success("Keys downloaded");
              }}
            >
              <Download className="h-4 w-4 mr-2" />
              Download as File
            </Button>
            <Button
              onClick={() => {
                setKeysDialogOpen(false);
                setCreatedResponse(null);
              }}
            >
              I've Saved the Keys
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Tenant</DialogTitle>
            <DialogDescription>Update tenant name.</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">
                Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="edit-name"
                value={editTenant.name}
                onChange={(e) =>
                  setEditTenant({ ...editTenant, name: e.target.value })
                }
                placeholder="Acme Corporation"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleUpdateTenant}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? "Updating..." : "Update Tenant"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

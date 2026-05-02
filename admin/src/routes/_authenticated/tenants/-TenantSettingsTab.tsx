import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Settings, RefreshCw, Pencil, Save, X, RotateCcw } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { tenantSettingsApi, type ResolvedSetting } from "@/lib/api";

interface TenantSettingsTabProps {
  tenantId: string;
}

export function TenantSettingsTab({ tenantId }: TenantSettingsTabProps) {
  const queryClient = useQueryClient();
  const [editingPath, setEditingPath] = useState<string | null>(null);
  const [editValue, setEditValue] = useState<unknown>(null);
  const [isResetDialogOpen, setShowResetDialog] = useState<string | null>(null);

  // Fetch tenant settings
  const { data: settingsData, isLoading } = useQuery({
    queryKey: ["tenant-settings", tenantId],
    queryFn: () => tenantSettingsApi.get(tenantId),
    enabled: !!tenantId,
  });

  // Update tenant settings mutation
  const updateSettingsMutation = useMutation({
    mutationFn: (data: { path: string; value: unknown }) =>
      tenantSettingsApi.update(tenantId, {
        settings: { [data.path]: data.value },
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["tenant-settings", tenantId],
      });
      toast.success("Setting updated successfully");
    },
    onError: (error: Error) => {
      toast.error(`Failed to update setting: ${error.message}`);
    },
  });

  // Delete tenant setting mutation (reset to default)
  const deleteSettingMutation = useMutation({
    mutationFn: (path: string) => tenantSettingsApi.delete(tenantId, path),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["tenant-settings", tenantId],
      });
      toast.success("Setting reset to default");
    },
    onError: (error: Error) => {
      toast.error(`Failed to reset setting: ${error.message}`);
    },
  });

  const handleSaveEdit = (path: string) => {
    updateSettingsMutation.mutate({ path, value: editValue });
    setEditingPath(null);
    setEditValue(null);
  };

  const handleResetSetting = (path: string) => {
    deleteSettingMutation.mutate(path);
    setShowResetDialog(null);
  };

  const renderValue = (value: unknown, dataType?: string): string => {
    if (value === null || value === undefined) return "Not set";
    if (dataType === "boolean") {
      return value ? "Enabled" : "Disabled";
    }
    if (dataType === "object" || dataType === "array") {
      return JSON.stringify(value, null, 2);
    }
    return String(value);
  };

  const getSourceBadge = (
    source: string,
    isOverridable?: boolean,
    isReadOnly?: boolean,
  ) => {
    if (isReadOnly) {
      return (
        <Badge variant="outline" className="text-xs bg-gray-100 text-gray-600">
          Locked
        </Badge>
      );
    }
    if (!isOverridable) {
      return (
        <Badge variant="outline" className="text-xs bg-gray-100 text-gray-600">
          Not Overridable
        </Badge>
      );
    }
    switch (source) {
      case "tenant":
        return (
          <Badge variant="outline" className="text-xs bg-blue-500 text-white">
            Tenant
          </Badge>
        );
      case "instance":
        return (
          <Badge variant="outline" className="text-xs bg-purple-500 text-white">
            Instance
          </Badge>
        );
      case "config":
        return (
          <Badge variant="outline" className="text-xs bg-gray-500 text-white">
            Config
          </Badge>
        );
      default:
        return (
          <Badge variant="outline" className="text-xs bg-gray-400 text-white">
            Default
          </Badge>
        );
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <RefreshCw className="text-muted-foreground h-6 w-6 animate-spin" />
        </CardContent>
      </Card>
    );
  }

  const settings = settingsData?.settings || {};
  const settingsEntries = Object.entries(settings);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings className="h-5 w-5" />
          Tenant Settings
        </CardTitle>
        <CardDescription>
          Configure tenant-specific settings. Settings cascade: Config file →
          Instance Settings → Tenant Settings.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {settingsEntries.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No settings configured. Tenant uses instance defaults.
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Setting</TableHead>
                <TableHead>Value</TableHead>
                <TableHead>Source</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {settingsEntries.map(([path, setting]) => {
                const typedSetting = setting as ResolvedSetting;
                const isEditing = editingPath === path;
                const isOverridable = typedSetting.is_overridable !== false;
                const isReadOnly = typedSetting.is_read_only === true;

                return (
                  <TableRow key={path}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-medium">{path}</span>
                        {typedSetting.is_secret && (
                          <span className="text-xs text-muted-foreground">
                            Secret value hidden
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {isEditing ? (
                        <div className="flex items-center gap-2">
                          {typedSetting.data_type === "boolean" ? (
                            <Switch
                              checked={editValue as boolean}
                              onCheckedChange={(checked) =>
                                setEditValue(checked)
                              }
                            />
                          ) : (
                            <Input
                              value={(editValue as string) ?? ""}
                              onChange={(e) => setEditValue(e.target.value)}
                              className="w-[200px]"
                              readOnly={typedSetting.is_secret}
                              type={
                                typedSetting.is_secret ? "password" : "text"
                              }
                            />
                          )}
                          {typedSetting.is_secret && (
                            <span className="text-xs text-muted-foreground">
                              Update secret values via API
                            </span>
                          )}
                        </div>
                      ) : (
                        <span>
                          {renderValue(
                            typedSetting.value,
                            typedSetting.data_type,
                          )}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {getSourceBadge(
                        typedSetting.source,
                        isOverridable,
                        isReadOnly,
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center gap-2 justify-end">
                        {isEditing ? (
                          <>
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => {
                                setEditingPath(null);
                                setEditValue(null);
                              }}
                            >
                              <X className="h-4 w-4" />
                            </Button>
                            <Button
                              size="sm"
                              onClick={() => handleSaveEdit(path)}
                              disabled={updateSettingsMutation.isPending}
                            >
                              <Save className="h-4 w-4" />
                            </Button>
                          </>
                        ) : (
                          <>
                            {isOverridable && !isReadOnly && (
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => {
                                  setEditingPath(path);
                                  setEditValue(typedSetting.value);
                                }}
                              readOnly={typedSetting.is_secret}
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                            )}
                            {typedSetting.source === "tenant" && (
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => setShowResetDialog(path)}
                              >
                                <RotateCcw className="h-4 w-4" />
                              </Button>
                            )}
                          </>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>

      {/* Reset Confirmation Dialog */}
      <AlertDialog
        open={!!isResetDialogOpen}
        onOpenChange={(open) => !open && setShowResetDialog(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset Setting</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to reset this setting to its instance
              default? This will remove the tenant-specific override.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                isResetDialogOpen && handleResetSetting(isResetDialogOpen)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Reset
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}

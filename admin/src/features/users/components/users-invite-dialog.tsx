import React, { useState } from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { MailPlus, Send, Copy, Check, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/get-error-message";
import { userManagementApi } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { PasswordInput } from "@/components/password-input";
import { SelectDropdown } from "@/components/select-dropdown";
import { roles, appRoles } from "../data/data";
import { useUsers } from "./users-provider";

const formSchema = z.object({
  email: z.string().email("Please enter a valid email address."),
  role: z.string().min(1, "Role is required."),
  tenant_id: z.string().optional(),
  password: z
    .string()
    .min(8, "Password must be at least 8 characters.")
    .optional()
    .or(z.literal("")),
  skip_email: z.boolean(),
});

type UserInviteForm = z.infer<typeof formSchema>;

type UserInviteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function UsersInviteDialog({
  open,
  onOpenChange,
}: UserInviteDialogProps) {
  const queryClient = useQueryClient();
  const { userType } = useUsers();
  const { currentTenant, tenants } = useTenantStore();
  const [inviteResult, setInviteResult] = useState<{
    temporaryPassword?: string;
    message: string;
    emailSent: boolean;
  } | null>(null);
  const [copied, setCopied] = useState(false);

  // Default to 'app' if userType is undefined
  const safeUserType = userType ?? "app";

  // Ensure tenants is always an array
  const safeTenants = tenants ?? [];

  // Determine which roles to use based on user type
  const availableRoles = safeUserType === "app" ? appRoles : roles;
  const defaultRole =
    safeUserType === "app" ? "authenticated" : "instance_admin";

  // Ensure availableRoles is always an array
  const safeAvailableRoles = availableRoles ?? [];

  // Check if we need to show tenant selector (for app users when no tenant is selected)
  const showTenantSelector =
    safeUserType === "app" && !currentTenant && safeTenants.length > 0;

  const form = useForm<UserInviteForm>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
      role: defaultRole,
      tenant_id: "",
      password: "",
      skip_email: false,
    },
  });

  // Reset form with correct default role when userType changes
  React.useEffect(() => {
    form.reset({
      email: "",
      role: defaultRole,
      tenant_id: "",
      password: "",
      skip_email: false,
    });
  }, [safeUserType, defaultRole, form]);

  const inviteMutation = useMutation({
    mutationFn: (params: {
      data: {
        email: string;
        role: string;
        tenant_id?: string;
        password?: string;
        skip_email?: boolean;
      };
      userType: "app" | "dashboard";
    }) => userManagementApi.inviteUser(params.data, params.userType),
    onSuccess: (data) => {
      // Invalidate users query to refresh the list
      queryClient.invalidateQueries({ queryKey: ["users"] });

      // Show result
      setInviteResult({
        temporaryPassword: data.temporary_password,
        message: data.message,
        emailSent: data.email_sent,
      });

      // If email was sent, show toast and close dialog
      if (data.email_sent) {
        toast.success("User invited", { description: data.message });
        handleClose();
      }
    },
    onError: (error: unknown) => {
      toast.error("Failed to invite user", {
        description: getErrorMessage(error),
      });
    },
  });

  const onSubmit = (values: UserInviteForm) => {
    // Only send password if it's not empty
    const payload = {
      email: values.email,
      role: values.role,
      // Use selected tenant_id, or currentTenant from store, or nothing
      tenant_id: values.tenant_id || currentTenant?.id,
      ...(values.password && { password: values.password }),
      ...(values.skip_email && { skip_email: values.skip_email }),
    };
    inviteMutation.mutate({ data: payload, userType: safeUserType });
  };

  const handleClose = () => {
    form.reset();
    setInviteResult(null);
    setCopied(false);
    onOpenChange(false);
  };

  const copyToClipboard = async () => {
    if (inviteResult?.temporaryPassword) {
      await navigator.clipboard.writeText(inviteResult.temporaryPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  // Show temporary password result if SMTP is disabled
  if (inviteResult?.temporaryPassword) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader className="text-start">
            <DialogTitle className="flex items-center gap-2">
              <MailPlus /> User Invited
            </DialogTitle>
            <DialogDescription>{inviteResult.message}</DialogDescription>
          </DialogHeader>
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Share this temporary password with the user. They can use it to
              sign in and should change it immediately.
            </AlertDescription>
          </Alert>
          <div className="space-y-2">
            <Label>Temporary Password</Label>
            <div className="flex gap-2">
              <Input
                readOnly
                value={inviteResult.temporaryPassword}
                className="font-mono"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={copyToClipboard}
              >
                {copied ? (
                  <Check className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={handleClose}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader className="text-start">
          <DialogTitle className="flex items-center gap-2">
            <MailPlus /> Invite User
          </DialogTitle>
          <DialogDescription>
            Invite new user to join your team. Assign a role to define their
            access level.
            {safeUserType === "app" && currentTenant && (
              <span className="mt-1 block text-sm font-medium text-foreground">
                Adding to tenant: {currentTenant.name}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            id="user-invite-form"
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      type="email"
                      placeholder="eg: john.doe@gmail.com"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    The invitation will be sent to this email address
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="role"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Role</FormLabel>
                  <FormControl>
                    <SelectDropdown
                      defaultValue={field.value || defaultRole || ""}
                      onValueChange={field.onChange}
                      placeholder="Select a role"
                      items={
                        safeAvailableRoles.length > 0
                          ? safeAvailableRoles.map(({ label, value }) => ({
                              label,
                              value,
                            }))
                          : []
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    Defines what actions the user can perform
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            {showTenantSelector && safeTenants.length > 0 && (
              <FormField
                control={form.control}
                name="tenant_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Tenant</FormLabel>
                    <FormControl>
                      <SelectDropdown
                        defaultValue={field.value || ""}
                        onValueChange={field.onChange}
                        placeholder="Select a tenant"
                        items={safeTenants.map((tenant) => ({
                          label: tenant.name,
                          value: tenant.id,
                        }))}
                      />
                    </FormControl>
                    <FormDescription>
                      Select which tenant to add this user to
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password (optional)</FormLabel>
                  <FormControl>
                    <PasswordInput
                      placeholder="Leave empty to auto-generate"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    If left empty, a secure random password will be generated.
                    Must be at least 8 characters if provided.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="skip_email"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-3">
                  <div className="space-y-0.5">
                    <FormLabel>Skip invitation email</FormLabel>
                    <FormDescription>
                      Don't send an email. You'll need to share the password
                      manually.
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
          </form>
        </Form>
        <DialogFooter className="gap-y-2">
          <DialogClose asChild>
            <Button variant="outline" disabled={inviteMutation.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            type="submit"
            form="user-invite-form"
            disabled={inviteMutation.isPending}
          >
            {inviteMutation.isPending ? "Inviting..." : "Invite"} <Send />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

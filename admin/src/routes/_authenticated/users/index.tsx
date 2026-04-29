import { useState } from "react";
import { z } from "zod";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { Users, UserPlus, UserCheck, Clock } from "lucide-react";
import { userManagementApi } from "@/lib/api";
import { useTenantStore } from "@/stores/tenant-store";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { UsersDialogs } from "@/features/users/components/users-dialogs";
import { UsersInviteDialog } from "@/features/users/components/users-invite-dialog";
import { UsersProvider } from "@/features/users/components/users-provider";
import { UsersTable } from "@/features/users/components/users-table";

const usersSearchSchema = z.object({
  page: z.number().optional(),
  pageSize: z.number().optional(),
  email: z.string().optional(),
  provider: z.array(z.string()).optional(),
  role: z.array(z.string()).optional(),
});

export const Route = createFileRoute("/_authenticated/users/")({
  component: UsersPage,
  validateSearch: usersSearchSchema,
});

function UsersPage() {
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const { currentTenant } = useTenantStore();

  const { data: usersResponse, isLoading } = useQuery({
    queryKey: ["users", "app", currentTenant?.id],
    queryFn: () => userManagementApi.listUsers("app"),
  });

  const rawUsers = usersResponse?.users || [];

  const users = rawUsers.map((user) => ({
    ...user,
    last_sign_in: user.last_sign_in ? new Date(user.last_sign_in) : null,
    created_at: new Date(user.created_at),
    updated_at: new Date(user.updated_at),
  }));

  const totalUsers = users.length;
  const verifiedUsers = users.filter((u) => u.email_verified).length;
  const activeToday = users.filter((u) => {
    if (!u.last_sign_in) return false;
    const lastSignIn = new Date(u.last_sign_in);
    const today = new Date();
    return (
      lastSignIn.getDate() === today.getDate() &&
      lastSignIn.getMonth() === today.getMonth() &&
      lastSignIn.getFullYear() === today.getFullYear()
    );
  }).length;
  const pendingInvites = users.filter(
    (u) => u.provider === "invite_pending",
  ).length;

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading users...</div>
      </div>
    );
  }

  return (
    <UsersProvider userType="app">
      <div className="flex h-full flex-col">
        {/* Header */}
        <div className="bg-background flex items-center justify-between border-b px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
              <Users className="text-primary h-5 w-5" />
            </div>
            <div>
              <h1 className="text-xl font-semibold">Users</h1>
              <p className="text-muted-foreground text-sm">
                Manage application users
              </p>
            </div>
          </div>
          <Button onClick={() => setInviteDialogOpen(true)}>
            <UserPlus className="mr-2 h-4 w-4" />
            Invite User
          </Button>
        </div>

        <div className="flex-1 overflow-auto p-6">
          <div className="space-y-6">
            {/* Stats Cards */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Total Users
                  </CardTitle>
                  <Users className="text-muted-foreground h-4 w-4" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{totalUsers}</div>
                  <p className="text-muted-foreground text-xs">
                    {verifiedUsers} verified
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Active Today
                  </CardTitle>
                  <Clock className="text-muted-foreground h-4 w-4" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{activeToday}</div>
                  <p className="text-muted-foreground text-xs">
                    Users signed in today
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Pending Invites
                  </CardTitle>
                  <UserPlus className="text-muted-foreground h-4 w-4" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{pendingInvites}</div>
                  <p className="text-muted-foreground text-xs">
                    Awaiting first sign in
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">
                    Verified Users
                  </CardTitle>
                  <UserCheck className="text-muted-foreground h-4 w-4" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{verifiedUsers}</div>
                  <p className="text-muted-foreground text-xs">
                    {Math.round((verifiedUsers / totalUsers) * 100) || 0}% of
                    total
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Users Table */}
            <UsersTable data={users} />
          </div>
        </div>

        {/* Invite Dialog */}
        <UsersInviteDialog
          open={inviteDialogOpen}
          onOpenChange={setInviteDialogOpen}
        />
      </div>

      {/* Dialogs for edit/delete actions */}
      <UsersDialogs />
    </UsersProvider>
  );
}

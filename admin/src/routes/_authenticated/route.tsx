import { useState, useEffect } from "react";
import { createFileRoute, isRedirect, redirect } from "@tanstack/react-router";
import { isAuthenticated } from "@/lib/auth";
import { adminAuthAPI } from "@/lib/api/auth";
import { AuthenticatedLayout } from "@/components/layout/authenticated-layout";
import { Loader2 } from "lucide-react";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ location }) => {
    if (!isAuthenticated()) {
      try {
        const status = await adminAuthAPI.getSetupStatus();
        if (status.needs_setup) {
          throw redirect({ to: "/setup" });
        }
      } catch (err) {
        if (isRedirect(err)) throw err;
      }

      throw redirect({
        to: "/login",
        search: {
          redirect: location.href,
        },
      });
    }
  },
  component: AuthenticatedRoute,
});

function AuthenticatedRoute() {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const timer = requestAnimationFrame(() => setLoading(false));
    return () => cancelAnimationFrame(timer);
  }, []);

  if (loading) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return <AuthenticatedLayout />;
}

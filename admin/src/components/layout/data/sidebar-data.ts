// This file is the single source of truth for sidebar navigation AND route metadata.
// When adding a new authenticated route, you must:
// 1. Create the route file in routes/_authenticated/
// 2. Add a sidebar item here (or to EXTRA_TENANT_ROUTES if no sidebar entry is needed)
// The getRouteConfig() function derives requiresTenant and isInstanceLevel from this data.

import {
  LayoutDashboard,
  Database,
  GitFork,
  Code,
  Code2,
  ScrollText,
  Activity,
  Users,
  Shield,
  Zap,
  FileCode,
  FolderOpen,
  Radio,
  ListTodo,
  Terminal,
  Bot,
  BookOpen,
  Wrench,
  Key,
  KeyRound,
  ShieldAlert,
  ShieldCheck,
  Webhook,
  Lock,
  Settings,
  Palette,
  Puzzle,
  Mail,
  HardDrive,
  Command,
  Building2,
  Server,
  Globe,
} from "lucide-react";
import { type SidebarData } from "../types";

export type VisibilityLevel = "all" | "instance-only" | "tenant-only";

export interface SidebarItem {
  title: string;
  url: string;
  icon: React.ComponentType<{ className?: string }>;
  visibility?: VisibilityLevel;
  requiresTenant?: boolean;
  badge?: string;
  external?: boolean;
}

export interface SidebarGroup {
  title: string;
  collapsible?: boolean;
  items: SidebarItem[];
  visibility?: VisibilityLevel;
  requiresTenant?: boolean;
}

export interface SidebarDataWithVisibility extends Omit<
  SidebarData,
  "navGroups"
> {
  navGroups: SidebarGroup[];
}

/**
 * Get the effective route configuration for a given pathname.
 * Walks sidebar groups/items to find the best matching route (longest prefix match).
 * Returns { requiresTenant, isInstanceLevel } derived from group default + item override.
 */
export function getRouteConfig(pathname: string): {
  requiresTenant: boolean;
  isInstanceLevel: boolean;
} {
  let bestMatch: {
    length: number;
    requiresTenant: boolean;
    isInstanceLevel: boolean;
  } | null = null;

  for (const group of sidebarData.navGroups) {
    for (const item of group.items) {
      const url = item.url;
      // Skip non-path items (like "/" for dashboard — too short to be meaningful)
      if (url.length <= 1) continue;

      // Check if pathname starts with this item's URL
      if (
        pathname === url ||
        pathname.startsWith(url + "/") ||
        pathname.startsWith(url + "?")
      ) {
        if (!bestMatch || url.length > bestMatch.length) {
          // Item overrides group default
          const reqTenant =
            item.requiresTenant ?? group.requiresTenant ?? false;
          const isInstanceLevel =
            (item.visibility || group.visibility || "all") === "instance-only";
          bestMatch = {
            length: url.length,
            requiresTenant: reqTenant,
            isInstanceLevel,
          };
        }
      }
    }
  }

  // Check extra routes not in sidebar
  for (const extra of EXTRA_TENANT_ROUTES) {
    if (
      pathname === extra ||
      pathname.startsWith(extra + "/") ||
      pathname.startsWith(extra + "?")
    ) {
      if (!bestMatch || extra.length > bestMatch.length) {
        bestMatch = {
          length: extra.length,
          requiresTenant: true,
          isInstanceLevel: false,
        };
      }
    }
  }

  return bestMatch
    ? {
        requiresTenant: bestMatch.requiresTenant,
        isInstanceLevel: bestMatch.isInstanceLevel,
      }
    : { requiresTenant: false, isInstanceLevel: false };
}

// Extra routes that need tenant context but aren't in the sidebar
const EXTRA_TENANT_ROUTES = ["/quotas"];

export const sidebarData: SidebarDataWithVisibility = {
  user: {
    name: "Admin",
    email: "admin@fluxbase.eu",
    avatar: "",
  },
  teams: [
    {
      name: "Fluxbase",
      logo: Command,
      plan: "Backend as a Service",
    },
  ],
  navGroups: [
    {
      title: "Overview",
      items: [
        {
          title: "Dashboard",
          url: "/",
          icon: LayoutDashboard,
          visibility: "all",
        },
      ],
    },
    {
      title: "Database",
      collapsible: true,
      requiresTenant: true,
      items: [
        {
          title: "Tables",
          url: "/tables",
          icon: Database,
          visibility: "all",
        },
        {
          title: "Schema Viewer",
          url: "/schema",
          icon: GitFork,
          visibility: "all",
        },
        {
          title: "SQL Editor",
          url: "/sql-editor",
          icon: Code,
          visibility: "all",
        },
      ],
    },
    {
      title: "Users & Authentication",
      collapsible: true,
      requiresTenant: true,
      items: [
        {
          title: "Users",
          url: "/users",
          icon: Users,
          visibility: "all",
          requiresTenant: false,
        },
        {
          title: "Authentication",
          url: "/authentication",
          icon: Shield,
          visibility: "all",
          requiresTenant: false,
        },
      ],
    },
    {
      title: "AI",
      collapsible: true,
      requiresTenant: true,
      items: [
        {
          title: "AI Providers",
          url: "/ai-providers",
          icon: Bot,
          visibility: "all",
        },
        {
          title: "Tool Integrations",
          url: "/ai-integrations",
          icon: Globe,
          visibility: "all",
        },
        {
          title: "Knowledge Bases",
          url: "/knowledge-bases",
          icon: BookOpen,
          visibility: "all",
        },
        {
          title: "AI Chatbots",
          url: "/chatbots",
          icon: Bot,
          visibility: "all",
        },
        {
          title: "MCP Tools",
          url: "/mcp-tools",
          icon: Wrench,
          visibility: "all",
        },
      ],
    },
    {
      title: "API & Services",
      collapsible: true,
      requiresTenant: true,
      items: [
        {
          title: "API Explorer",
          url: "/api/rest",
          icon: Code2,
          visibility: "all",
          requiresTenant: false, // Override: works at both levels
        },
        },
        {
          title: "Realtime",
          url: "/realtime",
          icon: Radio,
          visibility: "all",
        },
        {
          title: "Storage",
          url: "/storage",
          icon: FolderOpen,
          visibility: "all",
        },
        {
          title: "Functions",
          url: "/functions",
          icon: FileCode,
          visibility: "all",
        },
        {
          title: "Jobs",
          url: "/jobs",
          icon: ListTodo,
          visibility: "all",
        },
        {
          title: "RPC",
          url: "/rpc",
          icon: Terminal,
          visibility: "all",
        },
        {
          title: "Email",
          url: "/email-settings",
          icon: Mail,
          visibility: "all",
          requiresTenant: false, // Both instance and tenant admins can access
        },
      ],
    },
    {
      title: "Security",
      collapsible: true,
      requiresTenant: true,
      items: [
        {
          title: "RLS Policies",
          url: "/policies",
          icon: ShieldAlert,
          visibility: "all",
        },
        {
          title: "Security Settings",
          url: "/security-settings",
          icon: ShieldCheck,
          visibility: "all",
        },
        {
          title: "Secrets",
          url: "/secrets",
          icon: Lock,
          visibility: "all",
        },
        {
          title: "Client Keys",
          url: "/client-keys",
          icon: Key,
          visibility: "all",
        },
        {
          title: "Service Keys",
          url: "/service-keys",
          icon: KeyRound,
          visibility: "all",
        },
        {
          title: "Webhooks",
          url: "/webhooks",
          icon: Webhook,
          visibility: "all",
        },
      ],
    },
    {
      title: "Monitoring",
      collapsible: true,
      items: [
        {
          title: "Log Stream",
          url: "/logs",
          icon: ScrollText,
          visibility: "all",
        },
        {
          title: "Monitoring",
          url: "/monitoring",
          icon: Activity,
          visibility: "all",
        },
        {
          title: "AI Audit Log",
          url: "/monitoring/ai-audit",
          icon: Shield,
          visibility: "all",
        },
      ],
    },
    {
      title: "Platform",
      collapsible: true,
      items: [
        {
          title: "Tenants",
          url: "/tenants",
          icon: Building2,
          visibility: "instance-only",
        },
        {
          title: "Configuration",
          url: "/features",
          icon: Zap,
          visibility: "instance-only",
        },
        {
          title: "Extensions",
          url: "/extensions",
          icon: Puzzle,
          visibility: "all",
          requiresTenant: false,
        },
        {
          title: "Database Config",
          url: "/database-config",
          icon: Database,
          visibility: "instance-only",
        },
        {
          title: "Instance Settings",
          url: "/instance-settings",
          icon: Server,
          visibility: "instance-only",
        },
        {
          title: "Storage Config",
          url: "/storage-config",
          icon: HardDrive,
          visibility: "instance-only",
        },
      ],
    },
    {
      title: "Account settings",
      collapsible: true,
      items: [
        {
          title: "Account",
          url: "/settings",
          icon: Settings,
          visibility: "all",
        },
        {
          title: "Appearance",
          url: "/settings/appearance",
          icon: Palette,
          visibility: "all",
        },
      ],
    },
  ],
};

export function filterSidebarForContext(
  groups: SidebarGroup[],
  options: { isInstanceAdmin: boolean; actingAsTenantAdmin: boolean },
): SidebarGroup[] {
  const { isInstanceAdmin } = options;

  return groups
    .map((group) => {
      const filteredItems = group.items.filter((item) => {
        const visibility = item.visibility || "all";

        // Instance admin: see everything (instance-only + all + tenant-only)
        if (isInstanceAdmin) {
          return true;
        }

        // Tenant admin: show tenant-only + all, hide instance-only
        if (visibility === "instance-only") {
          return false;
        }

        return true;
      });

      return { ...group, items: filteredItems };
    })
    .filter((group) => group.items.length > 0);
}

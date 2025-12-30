import {
  LayoutDashboard,
  Database,
  Users,
  Settings,
  Palette,
  Command,
  Zap,
  FileCode,
  FolderOpen,
  Radio,
  Key,
  KeyRound,
  Shield,
  Webhook,
  Activity,
  Code,
  Mail,
  ShieldCheck,
  ListTodo,
  Bot,
  Terminal,
  HardDrive,
  Puzzle,
  BookOpen,
  ScrollText,
  Lock,
} from 'lucide-react'
import { type SidebarData } from '../types'

export const sidebarData: SidebarData = {
  user: {
    name: 'Admin',
    email: 'admin@fluxbase.eu',
    avatar: '',
  },
  teams: [
    {
      name: 'Fluxbase',
      logo: Command,
      plan: 'Backend as a Service',
    },
  ],
  navGroups: [
    {
      title: 'General',
      items: [
        {
          title: 'Overview',
          url: '/',
          icon: LayoutDashboard,
        },
        {
          title: 'Tables',
          url: '/tables',
          icon: Database,
        },
        {
          title: 'SQL Editor',
          url: '/sql-editor',
          icon: Code,
        },
        {
          title: 'Monitoring',
          url: '/monitoring',
          icon: Activity,
        },
        {
          title: 'Log Stream',
          url: '/logs',
          icon: ScrollText,
        },
        {
          title: 'Users',
          url: '/users',
          icon: Users,
        },
      ],
    },
    {
      title: 'API & Services',
      items: [
        {
          title: 'REST API',
          url: '/api/rest',
          icon: Zap,
        },
        {
          title: 'Realtime',
          url: '/realtime',
          icon: Radio,
        },
        {
          title: 'Storage',
          url: '/storage',
          icon: FolderOpen,
        },
        {
          title: 'Functions',
          url: '/functions',
          icon: FileCode,
        },
        {
          title: 'Jobs',
          url: '/jobs',
          icon: ListTodo,
        },
        {
          title: 'RPC',
          url: '/rpc',
          icon: Terminal,
        },
        {
          title: 'AI Chatbots',
          url: '/chatbots',
          icon: Bot,
        },
        {
          title: 'Knowledge Bases',
          url: '/knowledge-bases',
          icon: BookOpen,
        },
      ],
    },
    {
      title: 'Configuration',
      items: [
        {
          title: 'Features',
          url: '/features',
          icon: Zap,
        },
        {
          title: 'Extensions',
          url: '/extensions',
          icon: Puzzle,
        },
        {
          title: 'Database',
          url: '/database-config',
          icon: Database,
        },
        {
          title: 'Email',
          url: '/email-settings',
          icon: Mail,
        },
        {
          title: 'Storage',
          url: '/storage-config',
          icon: HardDrive,
        },
        {
          title: 'AI Providers',
          url: '/ai-providers',
          icon: Bot,
        },
      ],
    },
    {
      title: 'Security',
      items: [
        {
          title: 'Authentication',
          url: '/authentication',
          icon: Shield,
        },
        {
          title: 'Security Settings',
          url: '/security-settings',
          icon: ShieldCheck,
        },
        {
          title: 'Secrets',
          url: '/secrets',
          icon: Lock,
        },
        {
          title: 'API Keys',
          url: '/api-keys',
          icon: Key,
        },
        {
          title: 'Service Keys',
          url: '/service-keys',
          icon: KeyRound,
        },
        {
          title: 'Webhooks',
          url: '/webhooks',
          icon: Webhook,
        },
      ],
    },
    {
      title: 'Settings',
      items: [
        {
          title: 'Account',
          url: '/settings',
          icon: Settings,
        },
        {
          title: 'Appearance',
          url: '/settings/appearance',
          icon: Palette,
        },
      ],
    },
  ],
}

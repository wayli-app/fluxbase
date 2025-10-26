import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  // Main documentation sidebar
  tutorialSidebar: [
    {
      type: 'doc',
      id: 'intro',
      label: 'Introduction',
    },
    {
      type: 'doc',
      id: 'authentication',
      label: 'Authentication',
    },
    {
      type: 'doc',
      id: 'realtime',
      label: 'Realtime',
    },
    {
      type: 'doc',
      id: 'storage',
      label: 'Storage',
    },
    {
      type: 'doc',
      id: 'testing-guide',
      label: 'Testing Guide',
    },
  ],

  // SDKs sidebar
  sdksSidebar: [
    {
      type: 'doc',
      id: 'sdks/index',
      label: 'Overview',
    },
    {
      type: 'doc',
      id: 'sdks/getting-started',
      label: 'Getting Started',
    },
    {
      type: 'doc',
      id: 'sdks/database',
      label: 'Database Operations',
    },
    {
      type: 'doc',
      id: 'sdks/react-hooks',
      label: 'React Hooks',
    },
    {
      type: 'link',
      label: 'API Reference - TypeScript SDK',
      href: '/api/sdk/',
    },
    {
      type: 'link',
      label: 'API Reference - React SDK',
      href: '/api/sdk-react/',
    },
  ],
};

export default sidebars;
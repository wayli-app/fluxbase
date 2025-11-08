import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "Fluxbase",
  tagline: "Lightweight Backend-as-a-Service Alternative to Supabase",
  favicon: "img/favicon.ico",

  url: "https://fluxbase.eu",
  baseUrl: "/",

  organizationName: "fluxbase",
  projectName: "fluxbase",

  onBrokenLinks: "warn",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  scripts: [
    {
      src: "https://umami.wayli.app/umami",
      defer: true,
      "data-website-id": "846445c5-4f05-4ec7-a3ec-46f06f94a314",
    },
  ],

  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },

  themes: ["@docusaurus/theme-mermaid"],

  plugins: [
    ...(process.env.SKIP_TYPEDOC !== 'true' ? [
      [
        "docusaurus-plugin-typedoc",
        {
          id: "sdk",
          entryPoints: ["../sdk/src/index.ts"],
          tsconfig: "../sdk/tsconfig.json",
          out: "docs/api/sdk",
          readme: "none",
          disableSources: true,
          excludePrivate: true,
          excludeProtected: true,
          excludeInternal: true,
          useCodeBlocks: true,
          useHTMLEncodedBrackets: true,
          parametersFormat: "table",
          propertiesFormat: "table",
          enumMembersFormat: "table",
          typeDeclarationFormat: "table",
          expandObjects: false,
          sidebar: {
            autoConfiguration: true,
          },
        },
      ],
      [
        "docusaurus-plugin-typedoc",
        {
          id: "sdk-react",
          entryPoints: ["../sdk-react/src/index.ts"],
          tsconfig: "../sdk-react/tsconfig.json",
          out: "docs/api/sdk-react",
          readme: "none",
          disableSources: true,
          excludePrivate: true,
          excludeProtected: true,
          excludeInternal: true,
          useCodeBlocks: true,
          useHTMLEncodedBrackets: true,
          parametersFormat: "table",
          propertiesFormat: "table",
          enumMembersFormat: "table",
          typeDeclarationFormat: "table",
          expandObjects: false,
          sidebar: {
            autoConfiguration: true,
          },
        },
      ],
    ] : []),
  ],

  presets: [
    [
      "classic",
      {
        docs: {
          sidebarPath: "./sidebars.ts",
          editUrl: "https://github.com/wayli-app/fluxbase/tree/main/docs/",
          showLastUpdateAuthor: true,
          showLastUpdateTime: true,
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: "img/fluxbase-social-card.jpg",
    navbar: {
      title: "Fluxbase",
      logo: {
        alt: "",
        src: "img/logo.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "docsSidebar",
          position: "left",
          label: "Docs",
        },
        {
          type: "docSidebar",
          sidebarId: "guidesSidebar",
          position: "left",
          label: "Guides",
        },
        {
          type: "docSidebar",
          sidebarId: "apiSidebar",
          position: "left",
          label: "API Reference",
        },
        {
          href: "https://github.com/wayli-app/fluxbase",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "light",
      links: [
        {
          title: "Docs",
          items: [
            {
              label: "Introduction",
              to: "/docs/intro",
            },
            {
              label: "Authentication",
              to: "/docs/guides/authentication",
            },
            {
              label: "Realtime",
              to: "/docs/guides/realtime",
            },
            {
              label: "Storage",
              to: "/docs/guides/storage",
            },
          ],
        },
        {
          title: "SDKs",
          items: [
            {
              label: "Getting Started",
              to: "/docs/guides/typescript-sdk/getting-started",
            },
            {
              label: "Database Operations",
              to: "/docs/guides/typescript-sdk/database",
            },
            {
              label: "React Hooks",
              to: "/docs/guides/typescript-sdk/react-hooks",
            },
          ],
        },
        {
          title: "Community",
          items: [
            {
              label: "GitHub",
              href: "https://github.com/wayli-app/fluxbase",
            },
            {
              label: "Discord",
              href: "https://discord.gg/BXPRHkQzkA",
            },
          ],
        },
        {
          title: "More",
          items: [
            {
              label: "Releases",
              href: "https://github.com/wayli-app/fluxbase/releases",
            },
            {
              label: "Roadmap",
              href: "https://github.com/wayli-app/fluxbase/projects",
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Fluxbase. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ["bash", "go", "sql", "yaml", "docker", "json"],
    },
    algolia: {
      appId: "YOUR_APP_ID",
      apiKey: "YOUR_API_KEY",
      indexName: "fluxbase",
      contextualSearch: true,
      searchPagePath: "search",
    },
  } satisfies Preset.ThemeConfig,
};

export default config;

import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightTypeDoc from "starlight-typedoc";
import sitemap from "@astrojs/sitemap";

export default defineConfig({
  site: "https://fluxbase.eu",
  integrations: [
    sitemap(),
    starlight({
      title: "Fluxbase",
      description: "Lightweight Backend-as-a-Service Alternative to Supabase",
      logo: {
        src: "./src/assets/logo-icon.svg",
        replacesTitle: false,
      },
      favicon: "/favicon.png",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/nimbleflux/fluxbase",
        },
        {
          icon: "discord",
          label: "Discord",
          href: "https://discord.gg/BXPRHkQzkA",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/nimbleflux/fluxbase/edit/main/docs/",
      },
      credits: true,
      components: {
        Footer: "./src/components/Footer.astro",
        PageSidebar: "./src/components/PageSidebar.astro",
        Header: "./src/components/Header.astro",
      },
      head: [
        // OpenGraph meta tags for social sharing
        {
          tag: "meta",
          attrs: { property: "og:type", content: "website" },
        },
        {
          tag: "meta",
          attrs: { property: "og:site_name", content: "Fluxbase" },
        },
        {
          tag: "meta",
          attrs: {
            property: "og:image",
            content: "https://fluxbase.eu/og-image.png",
          },
        },
        {
          tag: "meta",
          attrs: { property: "og:image:width", content: "1200" },
        },
        {
          tag: "meta",
          attrs: { property: "og:image:height", content: "630" },
        },
        {
          tag: "meta",
          attrs: {
            property: "og:image:alt",
            content: "Fluxbase - Lightweight Backend-as-a-Service",
          },
        },
        // Twitter Card meta tags
        {
          tag: "meta",
          attrs: { name: "twitter:card", content: "summary_large_image" },
        },
        {
          tag: "meta",
          attrs: {
            name: "twitter:image",
            content: "https://fluxbase.eu/og-image.png",
          },
        },
        // JSON-LD structured data
        {
          tag: "script",
          attrs: { type: "application/ld+json" },
          content: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Fluxbase",
            applicationCategory: "DeveloperApplication",
            operatingSystem: "Cross-platform",
            description:
              "Lightweight Backend-as-a-Service (BaaS) - a single-binary Supabase alternative with PostgreSQL as the only dependency.",
            url: "https://fluxbase.eu",
            offers: {
              "@type": "Offer",
              price: "0",
              priceCurrency: "USD",
            },
            author: {
              "@type": "Organization",
              name: "Fluxbase",
              url: "https://fluxbase.eu",
            },
          }),
        },
        // Preconnect for external resources
        {
          tag: "link",
          attrs: { rel: "preconnect", href: "https://umami.wayli.app" },
        },
        {
          tag: "link",
          attrs: { rel: "dns-prefetch", href: "https://cdn.jsdelivr.net" },
        },
        // Analytics
        {
          tag: "script",
          attrs: {
            src: "https://umami.wayli.app/umami",
            defer: true,
            "data-website-id": "846445c5-4f05-4ec7-a3ec-46f06f94a314",
          },
        },
        {
          tag: "script",
          attrs: {
            type: "module",
          },
          content: `
            import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
            mermaid.initialize({ startOnLoad: false, theme: 'neutral' });
            document.addEventListener('DOMContentLoaded', () => {
              const codeBlocks = document.querySelectorAll('pre[data-language="mermaid"]');
              codeBlocks.forEach((pre) => {
                const wrapper = pre.closest('.expressive-code');
                const copyBtn = wrapper?.querySelector('button[data-code]');
                if (!wrapper || !copyBtn) return;
                const text = copyBtn.getAttribute('data-code').split(String.fromCharCode(127)).join(String.fromCharCode(10));
                const container = document.createElement('div');
                container.className = 'mermaid';
                container.textContent = text;
                pre.replaceWith(container);
                // Force light background on wrapper for Mermaid diagrams
                wrapper.style.setProperty('background', '#ffffff', 'important');
                wrapper.style.setProperty('border-radius', '0.5rem');
                wrapper.style.setProperty('border', '1px solid #e5e7eb');
                const figure = wrapper.querySelector('figure');
                if (figure) {
                  figure.style.setProperty('background', '#ffffff', 'important');
                  figure.style.setProperty('margin', '0');
                }
              });
              mermaid.run();
            });
          `,
        },
        {
          tag: "style",
          content: `
            .expressive-code .copy button {
              width: 2rem !important;
              height: 2rem !important;
              padding: 0 !important;
              position: relative !important;
            }
            .expressive-code .copy button::after {
              position: absolute !important;
              top: 50% !important;
              left: 50% !important;
              transform: translate(-50%, -50%) !important;
              width: 1rem !important;
              height: 1rem !important;
              margin: 0 !important;
              mask-size: contain !important;
              mask-position: center !important;
              -webkit-mask-size: contain !important;
              -webkit-mask-position: center !important;
            }
          `,
        },
      ],
      customCss: ["./src/styles/custom.css"],
      expressiveCode: {
        themes: ["github-light", "dracula"],
        emitExternalStylesheet: true,
        styleOverrides: {
          frames: {
            inlineButtonBackgroundIdleOpacity: '0',
            inlineButtonBackgroundHoverOrFocusOpacity: '0.2',
            inlineButtonBackgroundActiveOpacity: '0.3',
          },
        },
      },
      plugins: [
        starlightTypeDoc({
          entryPoints: ["../sdk/src/index.ts"],
          tsconfig: "../sdk/tsconfig.json",
          output: "api/sdk",
          typeDoc: {
            readme: "none",
            disableSources: true,
            excludePrivate: true,
            excludeProtected: true,
            excludeInternal: true,
            parametersFormat: "table",
            propertiesFormat: "table",
            enumMembersFormat: "table",
            typeDeclarationFormat: "table",
          },
        }),
        starlightTypeDoc({
          entryPoints: ["../sdk-react/src/index.ts"],
          tsconfig: "../sdk-react/tsconfig.json",
          output: "api/sdk-react",
          typeDoc: {
            readme: "none",
            disableSources: true,
            excludePrivate: true,
            excludeProtected: true,
            excludeInternal: true,
            parametersFormat: "table",
            propertiesFormat: "table",
            enumMembersFormat: "table",
            typeDeclarationFormat: "table",
          },
        }),
        starlightTypeDoc({
          entryPoints: ["../sdk-svelte/src/index.ts"],
          tsconfig: "../sdk-svelte/tsconfig.json",
          output: "api/sdk-svelte",
          typeDoc: {
            readme: "none",
            disableSources: true,
            excludePrivate: true,
            excludeProtected: true,
            excludeInternal: true,
            parametersFormat: "table",
            propertiesFormat: "table",
            enumMembersFormat: "table",
            typeDeclarationFormat: "table",
          },
        }),
        starlightTypeDoc({
          entryPoints: ["../sdk-next/src/index.ts"],
          tsconfig: "../sdk-next/tsconfig.json",
          output: "api/sdk-next",
          typeDoc: {
            readme: "none",
            disableSources: true,
            excludePrivate: true,
            excludeProtected: true,
            excludeInternal: true,
            parametersFormat: "table",
            propertiesFormat: "table",
            enumMembersFormat: "table",
            typeDeclarationFormat: "table",
          },
        }),
        starlightTypeDoc({
          entryPoints: ["../sdk-vue/src/index.ts"],
          tsconfig: "../sdk-vue/tsconfig.json",
          output: "api/sdk-vue",
          typeDoc: {
            readme: "none",
            disableSources: true,
            excludePrivate: true,
            excludeProtected: true,
            excludeInternal: true,
            parametersFormat: "table",
            propertiesFormat: "table",
            enumMembersFormat: "table",
            typeDeclarationFormat: "table",
          },
        }),
      ],
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { label: "Introduction", link: "/intro/" },
            { label: "Quick Start", link: "/getting-started/quick-start/" },
            { label: "Building from Source", link: "/getting-started/building-from-source/" },
            { label: "Developer Guide", link: "/guides/developer-guide/" },
          ],
        },
        {
          label: "Configuration",
          items: [
            {
              label: "Configuration Reference",
              link: "/reference/configuration/",
            },
          ],
        },
        {
          label: "Resources",
          items: [
            { label: "API Cookbook", link: "/api-cookbook/" },
            { label: "Supabase Comparison", link: "/supabase-comparison/" },
          ],
        },
        {
          label: "Guides",
          collapsed: true,
          items: [
            // Core features (most important first)
            { label: "Authentication", link: "/guides/authentication/" },
            { label: "Storage", link: "/guides/storage/" },
            { label: "Realtime", link: "/guides/realtime/" },
            { label: "Edge Functions", link: "/guides/edge-functions/" },
            { label: "Background Jobs", link: "/guides/jobs/" },
            { label: "RPC", link: "/guides/rpc/" },

            // Database
            { label: "Multi-Tenancy", link: "/guides/multi-tenancy/" },
            { label: "Row-Level Security", link: "/guides/row-level-security/" },
            { label: "Database Migrations", link: "/guides/database-migrations/" },
            {
              label: "Database Branching",
              collapsed: true,
              items: [{ autogenerate: { directory: "guides/branching" } }],
            },
            { label: "Client Keys", link: "/guides/client-keys/" },
            { label: "Service Keys", link: "/guides/service-keys/" },

            // Advanced Auth
            { label: "Dashboard Auth", link: "/guides/dashboard-auth/" },
            { label: "OAuth Providers", link: "/guides/oauth-providers/" },
            { label: "SAML SSO", link: "/guides/saml-sso/" },
            { label: "Captcha", link: "/guides/captcha/" },

            // AI Features
            { label: "Vector Search", link: "/guides/vector-search/" },
            { label: "AI Chatbots", link: "/guides/ai-chatbots/" },
            { label: "Knowledge Bases", link: "/guides/knowledge-bases/" },

            // Integration
            {
              label: "MCP Server",
              collapsed: true,
              items: [{ autogenerate: { directory: "guides/mcp" } }],
            },
            { label: "Webhooks", link: "/guides/webhooks/" },
            { label: "User Settings", link: "/guides/user-settings/" },

            // Operations
            { label: "Secrets Management", link: "/guides/secrets-management/" },
            { label: "Rate Limiting", link: "/guides/rate-limiting/" },
            { label: "Logging", link: "/guides/logging/" },
            { label: "Monitoring", link: "/guides/monitoring-observability/" },
            { label: "Email Services", link: "/guides/email-services/" },
            { label: "PostgreSQL Extensions", link: "/guides/extensions/" },
            { label: "Image Transformations", link: "/guides/image-transformations/" },
            { label: "Testing", link: "/guides/testing/" },

            // Operations (advanced)
            { label: "Backup & Restore", link: "/guides/backup-restore/" },
            { label: "Automation", link: "/guides/automation/" },
            { label: "Distributed Tracing", link: "/guides/distributed-tracing/" },
            { label: "Operational Runbook", link: "/guides/operational-runbook/" },
            { label: "App Settings", link: "/settings/app-settings-guide/" },

            // Admin
            {
              label: "Admin Dashboard",
              collapsed: true,
              items: [{ autogenerate: { directory: "guides/admin" } }],
            },

            // Tutorials
            {
              label: "Tutorials",
              collapsed: true,
              items: [{ autogenerate: { directory: "guides/tutorials" } }],
            },
          ],
        },
        {
          label: "Security",
          collapsed: true,
          items: [{ autogenerate: { directory: "security" } }],
        },
        {
          label: "TypeScript SDK",
          collapsed: true,
          items: [{ autogenerate: { directory: "sdk" } }],
        },
        {
          label: "React SDK",
          collapsed: true,
          items: [
            { label: "Getting Started", link: "/sdk-react/getting-started/" },
          ],
        },
        {
          label: "Svelte SDK",
          collapsed: true,
          items: [
            { label: "Getting Started", link: "/sdk-svelte/getting-started/" },
          ],
        },
        {
          label: "Next.js SDK",
          collapsed: true,
          items: [
            { label: "Getting Started", link: "/sdk-next/getting-started/" },
          ],
        },
        {
          label: "Vue SDK",
          collapsed: true,
          items: [
            { label: "Getting Started", link: "/sdk-vue/getting-started/" },
          ],
        },
        {
          label: "CLI",
          collapsed: true,
          items: [
            { label: "Installation", link: "/cli/installation/" },
            { label: "Getting Started", link: "/cli/getting-started/" },
            { label: "Configuration", link: "/cli/configuration/" },
            {
              label: "Command Reference",
              collapsed: true,
              items: [
                { label: "Overview", link: "/cli/commands/" },
                { label: "Auth", link: "/cli/commands/#authentication-commands" },
                { label: "Functions", link: "/cli/commands/#functions-commands" },
                { label: "Jobs", link: "/cli/commands/#jobs-commands" },
                { label: "Storage", link: "/cli/commands/#storage-commands" },
                { label: "Chatbots", link: "/cli/commands/#chatbot-commands" },
                { label: "Knowledge Bases (kb)", link: "/cli/commands/#knowledge-base-commands" },
                { label: "AI Providers", link: "/cli/commands/#ai-providers-commands" },
                { label: "Tables", link: "/cli/commands/#table-commands" },
                { label: "Types", link: "/cli/commands/#type-generation-commands" },
                { label: "GraphQL", link: "/cli/commands/#graphql-commands" },
                { label: "RPC", link: "/cli/commands/#rpc-commands" },
                { label: "Webhooks", link: "/cli/commands/#webhook-commands" },
                { label: "Client Keys", link: "/cli/commands/#client-key-commands" },
                { label: "Migrations", link: "/cli/commands/#migration-commands" },
                { label: "Schema", link: "/cli/commands/#schema-commands" },
                { label: "Internal Schema", link: "/cli/commands/#internal-schema-commands" },
                { label: "Extensions", link: "/cli/commands/#extension-commands" },
                { label: "Realtime", link: "/cli/commands/#realtime-commands" },
                { label: "Settings", link: "/cli/commands/#settings-commands" },
                { label: "Settings Secrets", link: "/cli/commands/#settings-secrets-commands" },
                { label: "Service Keys", link: "/cli/commands/#service-key-commands" },
                { label: "Config", link: "/cli/commands/#config-commands" },
                { label: "Secrets (Legacy)", link: "/cli/commands/#secrets-commands-legacy" },
                { label: "Logs", link: "/cli/commands/#logs-commands" },
                { label: "MCP Tools", link: "/cli/commands/#mcp-commands" },
                { label: "Sync", link: "/cli/commands/#sync-command" },
                { label: "Branch", link: "/cli/commands/#branch-commands" },
                { label: "Admin", link: "/cli/commands/#admin-commands" },
                { label: "Users", link: "/cli/commands/#user-commands" },
                { label: "Version", link: "/cli/commands/#version-command" },
                { label: "Completion", link: "/cli/commands/#completion-command" },
                { label: "Command Aliases", link: "/cli/commands/#command-aliases" },
              ],
            },
            { label: "Workflows", link: "/cli/workflows/" },
          ],
        },
        {
          label: "Deployment",
          collapsed: true,
          items: [{ autogenerate: { directory: "deployment" } }],
        },
        {
          label: "API Reference",
          collapsed: true,
          items: [
            {
              label: "TypeScript SDK",
              collapsed: true,
              items: [{ autogenerate: { directory: "api/sdk" } }],
            },
            {
              label: "React SDK",
              collapsed: true,
              items: [{ autogenerate: { directory: "api/sdk-react" } }],
            },
            {
              label: "Svelte SDK",
              collapsed: true,
              items: [{ autogenerate: { directory: "api/sdk-svelte" } }],
            },
            {
              label: "Next.js SDK",
              collapsed: true,
              items: [{ autogenerate: { directory: "api/sdk-next" } }],
            },
            {
              label: "Vue SDK",
              collapsed: true,
              items: [{ autogenerate: { directory: "api/sdk-vue" } }],
            },
            { label: "HTTP API", link: "/api/http/" },
            { label: "GraphQL API", link: "/api/http/graphql/" },
          ],
        },
        { label: "Pricing", link: "/pricing/" },
        { label: "AI & Development Transparency", link: "/about/ai-transparency/" },
      ],
    }),
  ],
});

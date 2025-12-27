import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightTypeDoc from "starlight-typedoc";

export default defineConfig({
  site: "https://fluxbase.eu",
  integrations: [
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
          href: "https://github.com/fluxbase-eu/fluxbase",
        },
        {
          icon: "discord",
          label: "Discord",
          href: "https://discord.gg/BXPRHkQzkA",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/fluxbase-eu/fluxbase/edit/main/docs/",
      },
      head: [
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
                wrapper.replaceWith(container);
              });
              mermaid.run();
            });
          `,
        },
      ],
      customCss: ["./src/styles/custom.css"],
      expressiveCode: {
        themes: ["github-light", "dracula"],
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
      ],
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { label: "Introduction", link: "/intro/" },
            { label: "Installation", link: "/getting-started/installation/" },
            { label: "Quick Start", link: "/getting-started/quick-start/" },
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
          autogenerate: { directory: "guides" },
        },
        {
          label: "Security",
          collapsed: true,
          autogenerate: { directory: "security" },
        },
        {
          label: "TypeScript SDK",
          collapsed: true,
          autogenerate: { directory: "sdk" },
        },
        {
          label: "CLI",
          collapsed: true,
          autogenerate: { directory: "cli" },
        },
        {
          label: "Deployment",
          collapsed: true,
          autogenerate: { directory: "deployment" },
        },
        {
          label: "API Reference",
          collapsed: true,
          items: [
            {
              label: "TypeScript SDK",
              collapsed: true,
              autogenerate: { directory: "api/sdk" },
            },
            {
              label: "React SDK",
              collapsed: true,
              autogenerate: { directory: "api/sdk-react" },
            },
            { label: "HTTP API", link: "/api/http/" },
          ],
        },
      ],
    }),
  ],
});

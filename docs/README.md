# Fluxbase Documentation

This directory contains the Fluxbase documentation site built with [Docusaurus](https://docusaurus.io/).

## Running the Documentation Server

### Option 1: Using Make (Recommended)

From the project root:

```bash
# Install dependencies (first time only)
make docs-install

# Start the development server
make docs-dev
```

The documentation will be available at [http://localhost:3000](http://localhost:3000)

### Option 2: Using npm directly

From this directory (`/workspace/docs`):

```bash
# Install dependencies (first time only)
npm install

# Start the development server
npm start
```

### Option 3: Build static site

To build a production-ready static site:

```bash
# From project root
make docs-build

# Or from this directory
npm run build
```

The built site will be in the `build/` directory.

## Project Structure

```
docs/
├── docs/                  # Documentation pages (Markdown)
│   ├── intro.md          # Homepage
│   ├── authentication.md # Auth guide
│   └── testing-guide.md  # Testing guide
├── src/                   # React components
│   ├── components/       # Custom components
│   ├── css/             # Custom CSS
│   └── pages/           # Custom pages
├── static/               # Static assets
│   └── img/             # Images
├── docusaurus.config.ts  # Docusaurus configuration
├── sidebars.ts           # Sidebar configuration
└── package.json          # Dependencies

## Adding New Documentation

1. Create a new Markdown file in `docs/` directory:

```bash
touch docs/new-feature.md
```

2. Add frontmatter to your file:

```markdown
---
sidebar_position: 3
title: New Feature
---

# New Feature

Your content here...
```

3. The page will automatically appear in the sidebar

## Adding Images

Place images in `static/img/` and reference them:

```markdown
![Alt text](/img/my-image.png)
```

## Configuration

Edit `docusaurus.config.ts` to customize:
- Site title and tagline
- Navigation links
- Footer content
- Theme colors

Edit `sidebars.ts` to customize the sidebar structure.

## Writing Guide

### Code Blocks

\`\`\`javascript
const example = "code block with syntax highlighting";
\`\`\`

### Tabs

```mdx
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="js" label="JavaScript">
    \`\`\`javascript
    console.log('Hello from JavaScript');
    \`\`\`
  </TabItem>
  <TabItem value="py" label="Python">
    \`\`\`python
    print('Hello from Python')
    \`\`\`
  </TabItem>
</Tabs>
```

### Admonitions

```markdown
:::tip
This is a tip
:::

:::note
This is a note
:::

:::warning
This is a warning
:::

:::danger
This is important!
:::
```

## Hot Reload

When you edit documentation files, the browser will automatically refresh to show your changes.

## Troubleshooting

### Port 3000 already in use

If port 3000 is already in use, you can specify a different port:

```bash
npm start -- --port 3001
```

### Dependencies missing

If you see errors about missing dependencies:

```bash
npm install
```

Or from project root:

```bash
make docs-install
```

### Build fails

Try cleaning the build cache:

```bash
npm run clear
npm run build
```

## Deployment

The documentation can be deployed to:
- GitHub Pages
- Netlify
- Vercel
- Any static hosting service

See [Docusaurus Deployment Guide](https://docusaurus.io/docs/deployment) for details.

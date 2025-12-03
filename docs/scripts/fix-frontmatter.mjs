#!/usr/bin/env node
/**
 * Script to add frontmatter to markdown files that are missing it
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const DOCS_DIR = path.join(__dirname, '../src/content/docs');

async function getAllMarkdownFiles(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...await getAllMarkdownFiles(fullPath));
    } else if (entry.name.endsWith('.md') || entry.name.endsWith('.mdx')) {
      files.push(fullPath);
    }
  }

  return files;
}

function extractTitle(content) {
  // Try to find first H1
  const h1Match = content.match(/^#\s+(.+)$/m);
  if (h1Match) {
    return h1Match[1].trim();
  }

  // Fall back to filename
  return null;
}

async function processFile(filePath) {
  try {
    let content = await fs.readFile(filePath, 'utf-8');

    // Skip if already has frontmatter
    if (content.startsWith('---')) {
      return false;
    }

    // Extract title from content
    let title = extractTitle(content);
    if (!title) {
      // Use filename as title
      const filename = path.basename(filePath, path.extname(filePath));
      title = filename.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
    }

    // Create frontmatter
    const frontmatter = `---
title: "${title.replace(/"/g, '\\"')}"
---

`;

    // Add frontmatter to content
    content = frontmatter + content;

    await fs.writeFile(filePath, content);
    console.log(`Fixed: ${path.relative(DOCS_DIR, filePath)} -> "${title}"`);
    return true;
  } catch (error) {
    console.error(`Error processing ${filePath}:`, error.message);
    return false;
  }
}

async function main() {
  console.log('Fixing missing frontmatter in markdown files...\n');

  const files = await getAllMarkdownFiles(DOCS_DIR);
  console.log(`Found ${files.length} markdown files\n`);

  let fixedCount = 0;
  for (const file of files) {
    if (await processFile(file)) {
      fixedCount++;
    }
  }

  console.log(`\nFixed ${fixedCount} files.`);
}

main().catch(console.error);

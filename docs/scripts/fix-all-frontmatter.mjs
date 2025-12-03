#!/usr/bin/env node
/**
 * Script to fix all frontmatter issues in markdown files
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
  return null;
}

function parseFrontmatter(content) {
  const match = content.match(/^---\s*\n([\s\S]*?)\n---\s*\n/);
  if (match) {
    return {
      frontmatter: match[1],
      body: content.slice(match[0].length),
    };
  }
  return { frontmatter: '', body: content };
}

async function processFile(filePath) {
  try {
    let content = await fs.readFile(filePath, 'utf-8');
    const { frontmatter, body } = parseFrontmatter(content);

    // Check if title exists in frontmatter
    if (frontmatter.includes('title:')) {
      return false;
    }

    // Extract title from content
    let title = extractTitle(body);
    if (!title) {
      // Use filename as title
      const filename = path.basename(filePath, path.extname(filePath));
      title = filename.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');
    }

    // Escape quotes in title
    title = title.replace(/"/g, '\\"');

    // Build new frontmatter
    let newFrontmatter = `title: "${title}"`;
    if (frontmatter.trim()) {
      newFrontmatter += '\n' + frontmatter.trim();
    }

    // Reconstruct file
    const newContent = `---\n${newFrontmatter}\n---\n\n${body}`;

    await fs.writeFile(filePath, newContent);
    console.log(`Fixed: ${path.relative(DOCS_DIR, filePath)} -> "${title}"`);
    return true;
  } catch (error) {
    console.error(`Error processing ${filePath}:`, error.message);
    return false;
  }
}

async function main() {
  console.log('Fixing all frontmatter issues...\n');

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

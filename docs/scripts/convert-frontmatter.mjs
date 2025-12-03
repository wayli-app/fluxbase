#!/usr/bin/env node
/**
 * Script to convert Docusaurus frontmatter and syntax to Starlight format
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

function convertContent(content, filePath) {
  let modified = content;

  // Remove sidebar_position from frontmatter
  modified = modified.replace(/^---\n([\s\S]*?)---/m, (match, frontmatter) => {
    // Remove sidebar_position lines
    let newFrontmatter = frontmatter.replace(/sidebar_position:\s*\d+\n?/g, '');
    // Remove sidebar_label if it duplicates title
    newFrontmatter = newFrontmatter.replace(/sidebar_label:\s*["']?[^"'\n]+["']?\n?/g, '');
    return `---\n${newFrontmatter.trim()}\n---`;
  });

  // Convert Docusaurus admonitions to Starlight format
  // :::note -> :::note (same)
  // :::tip -> :::tip (same)
  // :::caution -> :::caution (same)
  // :::warning -> :::caution
  // :::danger -> :::danger (same)
  // :::info -> :::note

  // Convert :::warning to :::caution
  modified = modified.replace(/:::warning\b/g, ':::caution');

  // Convert :::info to :::note
  modified = modified.replace(/:::info\b/g, ':::note');

  // Add titles to admonitions that don't have them
  // :::note\n -> :::note[Note]\n
  // But only if there's no title already
  modified = modified.replace(/:::(note|tip|caution|danger)\s*\n(?!\[)/g, (match, type) => {
    const titles = {
      note: 'Note',
      tip: 'Tip',
      caution: 'Caution',
      danger: 'Danger'
    };
    return `:::${type}[${titles[type]}]\n`;
  });

  // Handle admonitions with inline titles: :::caution Pre-Release -> :::caution[Pre-Release]
  modified = modified.replace(/:::(note|tip|caution|danger)\s+([^\n\[]+)\n/g, (match, type, title) => {
    return `:::${type}[${title.trim()}]\n`;
  });

  // Fix import statements that Starlight doesn't use
  // Remove Docusaurus-specific imports
  modified = modified.replace(/import\s+.*from\s+['"]@docusaurus\/.*['"];\n?/g, '');
  modified = modified.replace(/import\s+.*from\s+['"]@site\/.*['"];\n?/g, '');

  // Convert Docusaurus <Tabs> and <TabItem> to Starlight format
  // This is a simplified conversion - complex cases may need manual review
  modified = modified.replace(/<Tabs>/g, '');
  modified = modified.replace(/<\/Tabs>/g, '');
  modified = modified.replace(/<TabItem value="([^"]+)"[^>]*>/g, '\n**$1:**\n');
  modified = modified.replace(/<\/TabItem>/g, '');

  return modified;
}

async function processFile(filePath) {
  try {
    const content = await fs.readFile(filePath, 'utf-8');
    const converted = convertContent(content, filePath);

    if (content !== converted) {
      await fs.writeFile(filePath, converted);
      console.log(`Converted: ${path.relative(DOCS_DIR, filePath)}`);
      return true;
    }
    return false;
  } catch (error) {
    console.error(`Error processing ${filePath}:`, error.message);
    return false;
  }
}

async function main() {
  console.log('Converting Docusaurus content to Starlight format...\n');

  const files = await getAllMarkdownFiles(DOCS_DIR);
  console.log(`Found ${files.length} markdown files\n`);

  let convertedCount = 0;
  for (const file of files) {
    if (await processFile(file)) {
      convertedCount++;
    }
  }

  console.log(`\nConversion complete! ${convertedCount} files modified.`);
}

main().catch(console.error);

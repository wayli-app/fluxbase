#!/usr/bin/env node
/**
 * Script to generate markdown documentation from OpenAPI spec
 *
 * Usage:
 *   node scripts/generate-openapi-docs.mjs [openapi.json]
 *
 * If no file is provided, it will try to fetch from OPENAPI_URL or use a default file.
 */

import fs from 'fs/promises';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const OUTPUT_DIR = path.join(__dirname, '../src/content/docs/api/http');

async function fetchSpec(source) {
  // Try file first
  if (source && !source.startsWith('http')) {
    try {
      const content = await fs.readFile(source, 'utf-8');
      return JSON.parse(content);
    } catch (e) {
      console.log(`Could not read file ${source}, trying as URL...`);
    }
  }

  // Try URL
  const url = source || process.env.OPENAPI_URL || 'http://localhost:8080/openapi.json';
  try {
    const response = await fetch(url);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return await response.json();
  } catch (e) {
    console.log(`Could not fetch from ${url}: ${e.message}`);
    return null;
  }
}

function slugify(str) {
  return str.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
}

function getMethodBadge(method) {
  const badges = {
    get: 'api-badge--get',
    post: 'api-badge--post',
    put: 'api-badge--put',
    patch: 'api-badge--patch',
    delete: 'api-badge--delete',
  };
  return badges[method.toLowerCase()] || '';
}

function generateParametersTable(parameters) {
  if (!parameters || parameters.length === 0) return '';

  let md = '## Parameters\n\n';
  md += '| Name | Location | Type | Required | Description |\n';
  md += '|------|----------|------|----------|-------------|\n';

  for (const param of parameters) {
    const type = param.schema?.type || 'string';
    const required = param.required ? 'Yes' : 'No';
    md += `| \`${param.name}\` | ${param.in} | ${type} | ${required} | ${param.description || '-'} |\n`;
  }

  return md + '\n';
}

function generateRequestBody(requestBody) {
  if (!requestBody) return '';

  let md = '## Request Body\n\n';

  const content = requestBody.content?.['application/json'];
  if (content?.schema) {
    md += '```json\n';
    md += JSON.stringify(content.schema, null, 2);
    md += '\n```\n\n';
  }

  return md;
}

function generateResponses(responses) {
  if (!responses) return '';

  let md = '## Responses\n\n';

  for (const [code, response] of Object.entries(responses)) {
    md += `### ${code} - ${response.description}\n\n`;

    const content = response.content?.['application/json'];
    if (content?.schema) {
      md += '```json\n';
      md += JSON.stringify(content.schema, null, 2);
      md += '\n```\n\n';
    }
  }

  return md;
}

function generateEndpointPage(path, method, operation) {
  const methodUpper = method.toUpperCase();
  const title = operation.summary || `${methodUpper} ${path}`;

  let md = `---
title: "${title}"
description: "${operation.description || title}"
---

<span class="${getMethodBadge(method)}">${methodUpper}</span> \`${path}\`

${operation.description || ''}

`;

  md += generateParametersTable(operation.parameters);
  md += generateRequestBody(operation.requestBody);
  md += generateResponses(operation.responses);

  // Add example
  md += '## Example\n\n```bash\n';
  md += `curl -X ${methodUpper} "http://localhost:8080${path}"`;
  if (operation.security?.length > 0) {
    md += ' \\\n  -H "Authorization: Bearer YOUR_TOKEN"';
  }
  if (operation.requestBody) {
    md += ' \\\n  -H "Content-Type: application/json" \\\n  -d \'{"key": "value"}\'';
  }
  md += '\n```\n';

  return md;
}

async function generateDocs(spec) {
  if (!spec) {
    console.log('No OpenAPI spec available. Using static documentation only.');
    return;
  }

  console.log(`Generating docs from OpenAPI spec: ${spec.info?.title || 'Unknown'}`);

  // Group endpoints by tag
  const endpointsByTag = {};

  for (const [path, pathItem] of Object.entries(spec.paths || {})) {
    for (const [method, operation] of Object.entries(pathItem)) {
      if (['get', 'post', 'put', 'patch', 'delete', 'head'].includes(method)) {
        const tag = operation.tags?.[0] || 'Other';
        if (!endpointsByTag[tag]) {
          endpointsByTag[tag] = [];
        }
        endpointsByTag[tag].push({ path, method, operation });
      }
    }
  }

  // Generate pages for each tag
  for (const [tag, endpoints] of Object.entries(endpointsByTag)) {
    const tagSlug = slugify(tag);
    const tagDir = path.join(OUTPUT_DIR, tagSlug);
    await fs.mkdir(tagDir, { recursive: true });

    // Generate tag index
    let indexMd = `---
title: ${tag}
description: ${tag} API endpoints
---

# ${tag}

`;

    for (const { path: endpointPath, method, operation } of endpoints) {
      const methodUpper = method.toUpperCase();
      indexMd += `- <span class="${getMethodBadge(method)}">${methodUpper}</span> \`${endpointPath}\` - ${operation.summary || ''}\n`;
    }

    await fs.writeFile(path.join(tagDir, 'index.md'), indexMd);

    // Generate individual endpoint pages
    for (const { path: endpointPath, method, operation } of endpoints) {
      const filename = `${operation.operationId || slugify(`${method}-${endpointPath}`)}.md`;
      const content = generateEndpointPage(endpointPath, method, operation);
      await fs.writeFile(path.join(tagDir, filename), content);
    }

    console.log(`  Generated ${endpoints.length} pages for ${tag}`);
  }
}

async function main() {
  const source = process.argv[2];

  console.log('OpenAPI Documentation Generator\n');

  const spec = await fetchSpec(source);
  await generateDocs(spec);

  console.log('\nDone!');
}

main().catch(console.error);

#!/bin/bash

# Fix unescaped angle brackets in TypeDoc-generated markdown files
# This script only escapes angle brackets in type expressions, not HTML tags

# Check if docs/api directory exists, if not exit gracefully
if [ ! -d "docs/api" ]; then
  echo "docs/api directory not found, skipping angle bracket fixes"
  exit 0
fi

find docs/api -name "*.md" -type f | while read -r file; do
  # Only escape Promise<...>, Array<...>, Record<...>, and similar TypeScript generic types
  # that appear outside of code blocks and are not already escaped
  sed -i 's/\bPromise<\([^>]*\)>/Promise\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bArray<\([^>]*\)>/Array\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bRecord<\([^>]*\)>/Record\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bMap<\([^>]*\)>/Map\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bSet<\([^>]*\)>/Set\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bPartial<\([^>]*\)>/Partial\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bRequired<\([^>]*\)>/Required\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bReadonly<\([^>]*\)>/Readonly\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bOmit<\([^>]*\)>/Omit\&lt;\1\&gt;/g' "$file"
  sed -i 's/\bPick<\([^>]*\)>/Pick\&lt;\1\&gt;/g' "$file"

  # Fix standalone <void> tags (common in TypeScript return types)
  sed -i 's/<void>/\&lt;void\&gt;/g' "$file"
done

echo "Fixed angle brackets in TypeDoc markdown files"

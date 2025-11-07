#!/bin/bash

# Fix unescaped angle brackets in TypeDoc-generated markdown files
# This script only escapes angle brackets in type expressions, not HTML tags

# Check if docs/api directory exists, if not exit gracefully
if [ ! -d "docs/api" ]; then
  echo "docs/api directory not found, skipping angle bracket fixes"
  exit 0
fi

echo "Fixing TypeDoc markdown files..."
find docs/api -name "*.md" -type f | while read -r file; do
  echo "  Processing: $file"
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

  # Fix standalone <void> and other TypeScript primitive types (common in return types)
  # These must be escaped anywhere they appear as they look like HTML tags to MDX
  sed -i 's/<void>/\&lt;void\&gt;/g' "$file"
  sed -i 's/<any>/\&lt;any\&gt;/g' "$file"
  sed -i 's/<unknown>/\&lt;unknown\&gt;/g' "$file"
  sed -i 's/<never>/\&lt;never\&gt;/g' "$file"

  # Fix any angle brackets that look like HTML tags (not in code blocks)
  # This is a more aggressive approach that catches most patterns
  # Pattern: Any < followed by a letter (not already escaped with &)
  sed -i 's/\([^&`]\)<\([a-zA-Z][a-zA-Z0-9]*\)>/\1\&lt;\2\&gt;/g' "$file"

  # Fix at the start of a line
  sed -i 's/^<\([a-zA-Z][a-zA-Z0-9]*\)>/\&lt;\1\&gt;/g' "$file"

  # Fix after whitespace
  sed -i 's/\s<\([a-zA-Z][a-zA-Z0-9]*\)>/ \&lt;\1\&gt;/g' "$file"

  # Fix arrow function syntax that might be in type definitions
  # Example: () => <ReturnType> becomes () =\&gt; \&lt;ReturnType\&gt;
  sed -i 's/=> </=\&gt; \&lt;/g' "$file"

  # Fix cases where < appears after : (common in type annotations)
  sed -i 's/: <\([a-zA-Z][a-zA-Z0-9]*\)>/: \&lt;\1\&gt;/g' "$file"

  # Count remaining angle brackets for debugging
  remaining=$(grep -o '<[a-zA-Z]' "$file" | wc -l || echo "0")
  if [ "$remaining" -gt 0 ]; then
    echo "    WARNING: $remaining potential unescaped angle brackets remain"
  fi
done

echo "Completed fixing angle brackets in TypeDoc markdown files"

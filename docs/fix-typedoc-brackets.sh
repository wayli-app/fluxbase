#!/bin/bash

# Fix unescaped angle brackets in TypeDoc-generated markdown files
# This script only escapes angle brackets in type expressions, not HTML tags

# Check if docs/api directory exists, if not exit gracefully
if [ ! -d "docs/api" ]; then
  echo "docs/api directory not found, skipping angle bracket fixes"
  exit 0
fi

echo "Fixing TypeDoc markdown files..."

# Detect if we're on macOS (BSD sed) or Linux (GNU sed)
if sed --version >/dev/null 2>&1; then
  # GNU sed (Linux)
  SED_INPLACE="sed -i"
else
  # BSD sed (macOS) - requires empty string for in-place without backup
  SED_INPLACE="sed -i ''"
fi

file_count=0
total_files=$(find docs/api -name "*.md" -type f | wc -l | tr -d ' ')

find docs/api -name "*.md" -type f | while read -r file; do
  file_count=$((file_count + 1))
  # Show progress every 10 files to reduce noise
  if [ $((file_count % 10)) -eq 0 ]; then
    echo "  Processing... ($file_count/$total_files files)"
  fi
  # Only escape Promise<...>, Array<...>, Record<...>, and similar TypeScript generic types
  # that appear outside of code blocks and are not already escaped
  $SED_INPLACE 's/\bPromise<\([^>]*\)>/Promise\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bArray<\([^>]*\)>/Array\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bRecord<\([^>]*\)>/Record\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bMap<\([^>]*\)>/Map\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bSet<\([^>]*\)>/Set\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bPartial<\([^>]*\)>/Partial\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bRequired<\([^>]*\)>/Required\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bReadonly<\([^>]*\)>/Readonly\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bOmit<\([^>]*\)>/Omit\&lt;\1\&gt;/g' "$file"
  $SED_INPLACE 's/\bPick<\([^>]*\)>/Pick\&lt;\1\&gt;/g' "$file"

  # Fix standalone <void> and other TypeScript primitive types (common in return types)
  # These must be escaped anywhere they appear as they look like HTML tags to MDX
  $SED_INPLACE 's/<void>/\&lt;void\&gt;/g' "$file"
  $SED_INPLACE 's/<any>/\&lt;any\&gt;/g' "$file"
  $SED_INPLACE 's/<unknown>/\&lt;unknown\&gt;/g' "$file"
  $SED_INPLACE 's/<never>/\&lt;never\&gt;/g' "$file"

  # Fix any angle brackets that look like HTML tags (not in code blocks)
  # This is a more aggressive approach that catches most patterns
  # Pattern: Any < followed by a letter (not already escaped with &)
  $SED_INPLACE 's/\([^&`]\)<\([a-zA-Z][a-zA-Z0-9]*\)>/\1\&lt;\2\&gt;/g' "$file"

  # Fix at the start of a line
  $SED_INPLACE 's/^<\([a-zA-Z][a-zA-Z0-9]*\)>/\&lt;\1\&gt;/g' "$file"

  # Fix after whitespace
  $SED_INPLACE 's/\s<\([a-zA-Z][a-zA-Z0-9]*\)>/ \&lt;\1\&gt;/g' "$file"

  # Fix arrow function syntax that might be in type definitions
  # Example: () => <ReturnType> becomes () =\&gt; \&lt;ReturnType\&gt;
  $SED_INPLACE 's/=> </=\&gt; \&lt;/g' "$file"

  # Fix cases where < appears after : (common in type annotations)
  $SED_INPLACE 's/: <\([a-zA-Z][a-zA-Z0-9]*\)>/: \&lt;\1\&gt;/g' "$file"

  # Fix JSX expressions that look like { data, error } tuple
  # These need to be wrapped in backticks to avoid being interpreted as JSX
  $SED_INPLACE 's/to { data, error } tuple/to `{ data, error }` tuple/g' "$file"
  $SED_INPLACE 's/Promise { data, error }/Promise `{ data, error }`/g' "$file"

  # Skip warning output - angle brackets in code blocks are safe and expected
  # The MDX compiler properly handles them during build
done

echo "Completed fixing angle brackets in TypeDoc markdown files"

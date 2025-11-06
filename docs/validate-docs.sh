#!/bin/bash
set -e

echo "Validating documentation files for MDX compatibility..."

# Check for unescaped angle brackets in type expressions
ERRORS=0

# Check for Promise<, Array<, Record<, etc. that aren't escaped
echo "Checking for unescaped TypeScript generic types..."
if grep -r --include="*.md" "^Promise<\|^Array<\|^Record<\|^Map<\|^Set<" docs/api/ 2>/dev/null; then
  echo "❌ ERROR: Found unescaped generic types at line start"
  ERRORS=$((ERRORS + 1))
fi

# Check for common patterns like "Promise<void>" in plain text (not in code blocks)
# This is a simplified check - looks for patterns outside backticks
if grep -r --include="*.md" "[^&]Promise<" docs/api/ 2>/dev/null | grep -v '`Promise<' | grep -v '```'; then
  echo "❌ ERROR: Found unescaped Promise< outside code blocks"
  ERRORS=$((ERRORS + 1))
fi

# Check for broken markdown links
echo "Checking for broken internal links..."
if grep -r --include="*.md" '\[.*\](.*\.md)' docs/ | grep -v "node_modules" | while IFS=: read -r file link; do
  # Extract the link path
  link_path=$(echo "$link" | sed -n 's/.*](\([^)]*\.md\)).*/\1/p')
  if [ -n "$link_path" ]; then
    # Resolve relative path
    dir=$(dirname "$file")
    if [ ! -f "$dir/$link_path" ] && [ ! -f "docs/$link_path" ]; then
      echo "❌ Broken link in $file: $link_path"
      ERRORS=$((ERRORS + 1))
    fi
  fi
done; then
  : # Success
fi

if [ $ERRORS -eq 0 ]; then
  echo "✅ All documentation validation checks passed!"
  exit 0
else
  echo "❌ Documentation validation failed with $ERRORS error(s)"
  exit 1
fi

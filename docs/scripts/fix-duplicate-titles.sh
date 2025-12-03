#!/bin/bash

# This script removes duplicate H1 headings from Starlight docs
# Starlight automatically renders the frontmatter title as the page heading,
# so having an H1 in the content creates a duplicate

for file in $(find /workspace/docs/src/content/docs -name "*.md" -type f); do
  # Check if file has frontmatter (starts with ---)
  if head -1 "$file" | grep -q "^---"; then
    # Use awk to remove the first H1 heading after frontmatter
    awk '
      BEGIN { in_frontmatter = 0; frontmatter_done = 0; removed_h1 = 0 }
      /^---$/ && !in_frontmatter { in_frontmatter = 1; print; next }
      /^---$/ && in_frontmatter { in_frontmatter = 0; frontmatter_done = 1; print; next }
      in_frontmatter { print; next }
      frontmatter_done && !removed_h1 && /^$/ { next }  # Skip empty lines after frontmatter
      frontmatter_done && !removed_h1 && /^# / { removed_h1 = 1; next }  # Skip first H1
      frontmatter_done && !removed_h1 && /^[^#]/ { removed_h1 = 1; print; next }  # Non-H1 content, stop looking
      { print }
    ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
    echo "Processed: $file"
  fi
done

echo "Done!"

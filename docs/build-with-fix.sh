#!/bin/bash

# Try building - this will generate TypeDoc files
# Don't exit on error yet
set +e
npx docusaurus build > /tmp/build.log 2>&1
BUILD_EXIT_CODE=$?
set -e

# Always show the build log
cat /tmp/build.log

if [ $BUILD_EXIT_CODE -ne 0 ]; then
  # Build failed, check if it's due to angle bracket issues
  if grep -q "Expected a closing tag" /tmp/build.log; then
    echo "======================================"
    echo "Build failed due to angle bracket issues in generated markdown"
    echo "Applying fixes to TypeDoc-generated files..."
    echo "======================================"

    # Check if docs/api exists
    if [ -d "docs/api" ]; then
      echo "Found docs/api directory"
      echo "Files to fix:"
      find docs/api -name "*.md" -type f | head -5
    else
      echo "WARNING: docs/api directory not found!"
    fi

    bash fix-typedoc-brackets.sh

    # Show a sample of what was fixed
    echo "Sample of fixes applied:"
    grep -n "&lt;void&gt;" docs/api/sdk/classes/SystemSettingsManager.md 2>/dev/null | head -3 || echo "No <void> fixes found in SystemSettingsManager.md"

    echo "======================================"
    echo "Retrying build after fixes..."
    echo "NOTE: Skipping TypeDoc regeneration to preserve fixes"
    echo "======================================"
    SKIP_TYPEDOC=true npx docusaurus build
  else
    # Some other error, re-throw it
    echo "Build failed with a different error"
    exit 1
  fi
fi

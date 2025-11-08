#!/bin/bash

echo "======================================"
echo "Starting optimized docs build process"
echo "======================================"

# Step 1: Run a minimal build just to generate TypeDoc markdown files
# We use 'docusaurus build' which will fail due to angle brackets, but will generate the files
echo "Generating TypeDoc API documentation..."
set +e
npx docusaurus build > /tmp/typedoc-gen.log 2>&1
TYPEDOC_GEN_EXIT=$?
set -e

# Check if TypeDoc files were generated (even if build failed)
if [ -d "docs/api/sdk" ] || [ -d "docs/api/sdk-react" ]; then
  echo "TypeDoc files generated successfully"

  # Step 2: Apply angle bracket fixes to generated files
  echo "======================================"
  echo "Applying angle bracket fixes..."
  echo "======================================"
  bash fix-typedoc-brackets.sh

  # Step 3: Build again with SKIP_TYPEDOC to use fixed files
  echo "======================================"
  echo "Building Docusaurus site with fixed files..."
  echo "======================================"
  set +e
  SKIP_TYPEDOC=true npx docusaurus build > /tmp/build.log 2>&1
  BUILD_EXIT_CODE=$?
  set -e

  cat /tmp/build.log

  if [ $BUILD_EXIT_CODE -ne 0 ]; then
    echo "======================================"
    echo "ERROR: Build failed even after fixes"
    echo "======================================"
    exit 1
  fi

  echo "======================================"
  echo "Build succeeded!"
  echo "======================================"
else
  # No TypeDoc files generated, might be a clean build without API source
  # Just show the log and check if it succeeded
  cat /tmp/typedoc-gen.log

  if [ $TYPEDOC_GEN_EXIT -eq 0 ]; then
    echo "======================================"
    echo "Build succeeded!"
    echo "======================================"
  else
    echo "======================================"
    echo "ERROR: Build failed and no TypeDoc files were generated"
    echo "======================================"
    exit 1
  fi
fi

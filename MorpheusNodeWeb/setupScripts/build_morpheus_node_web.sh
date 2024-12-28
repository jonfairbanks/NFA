#!/usr/bin/env zsh

# Stop on first error
set -e

PROJECT_NAME="MorpheusNodeWeb"

# Verify the project folder exists
if [ ! -d "$PROJECT_NAME" ]; then
  echo "Error: Folder '$PROJECT_NAME' does not exist. Please ensure you've completed previous steps."
  exit 1
fi

echo "Navigating into '$PROJECT_NAME'..."
cd "$PROJECT_NAME"

echo "Installing dependencies (npm install)..."
npm install

echo "Building the Next.js app (npm run build)..."
npm run build

echo "Done! Your Next.js app is built."
echo "You can now run 'npm start' to serve the production build, or 'npm run dev' for development mode."
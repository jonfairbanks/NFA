#!/usr/bin/env zsh

# Stop on first error
set -e

PROJECT_NAME="MorpheusNodeWeb"

# 1) Check if MorpheusNodeWeb folder exists
if [ ! -d "$PROJECT_NAME" ]; then
  echo "Error: Folder '$PROJECT_NAME' does not exist."
  echo "Please ensure you have created the project first."
  exit 1
fi

echo "Navigating to '$PROJECT_NAME'..."
cd "$PROJECT_NAME"

# 2) Create or overwrite a global.d.ts file at the project root
echo "Creating 'global.d.ts' to declare window.ethereum..."
cat << 'EOF' > global.d.ts
// global.d.ts
// Ensures TypeScript recognizes window.ethereum

export {};

declare global {
  interface Window {
    ethereum?: any;
  }
}
EOF

# 3) (Optional) If you want to ensure TS sees global.d.ts, 
# verify that tsconfig.json includes `"include": ["**/*.ts", "**/*.tsx"]`.
# We'll just remind you here:
echo "Checking tsconfig.json for 'include' patterns..."
if grep -q '"include"' tsconfig.json; then
  echo "tsconfig.json already has an 'include' section. Great!"
else
  echo "Warning: tsconfig.json might need an 'include' section."
  echo "Example: \"include\": [\"next-env.d.ts\", \"**/*.ts\", \"**/*.tsx\"]"
fi

# 4) Install dependencies if needed
echo "Installing dependencies (npm install)..."
npm install

# 5) Build the Next.js app
echo "Building the Next.js app (npm run build)..."
npm run build

echo "Done! The 'window.ethereum' type error should be resolved."
echo "You can now run 'npm start' or 'npm run dev' to see your app."
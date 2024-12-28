#!/usr/bin/env zsh

# Stop on first error
set -e

# The name of the project folder
PROJECT_NAME="MorpheusNodeWeb"

# Ensure the project folder already exists (from Step 2).
if [ ! -d "$PROJECT_NAME" ]; then
  echo "Error: Folder '$PROJECT_NAME' does not exist. Please create it first."
  exit 1
fi

##############################################
# Create and populate each required file
##############################################

# -- package.json
cat << 'EOF' > "$PROJECT_NAME/package.json"
{
  "name": "MorpheusNodeWeb",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "13.4.12",
    "react": "18.2.0",
    "react-dom": "18.2.0",
    "ethers": "^6.0.0"
  },
  "devDependencies": {
    "@types/node": "^18.0.0",
    "@types/react": "^18.0.0",
    "@types/react-dom": "^18.0.0",
    "typescript": "^5.0.0"
  }
}
EOF

# -- tsconfig.json
cat << 'EOF' > "$PROJECT_NAME/tsconfig.json"
{
  "compilerOptions": {
    "target": "es6",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
EOF

# -- next.config.js
cat << 'EOF' > "$PROJECT_NAME/next.config.js"
/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true
};

module.exports = nextConfig;
EOF

# -- styles/globals.css
cat << 'EOF' > "$PROJECT_NAME/styles/globals.css"
/* Global placeholder styles */
body {
  margin: 0;
  font-family: Arial, sans-serif;
}
EOF

# -- pages/_app.tsx
cat << 'EOF' > "$PROJECT_NAME/pages/_app.tsx"
import type { AppProps } from 'next/app';
import '../styles/globals.css';

function MyApp({ Component, pageProps }: AppProps) {
  return <Component {...pageProps} />;
}

export default MyApp;
EOF

# -- pages/index.tsx
cat << 'EOF' > "$PROJECT_NAME/pages/index.tsx"
import React from 'react';

export default function Home() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Welcome to MorpheusNodeWeb</h1>
      <p>This is a placeholder home page.</p>
    </div>
  );
}
EOF

# -- components/NavBar.tsx
cat << 'EOF' > "$PROJECT_NAME/components/NavBar.tsx"
import React from 'react';
import Link from 'next/link';

export const NavBar: React.FC = () => {
  return (
    <nav style={{ background: '#f1f1f1', padding: '1rem' }}>
      <Link href="/">Home</Link>
      {/* Add more links as needed */}
    </nav>
  );
};
EOF

# -- components/WalletConnectButton.tsx
cat << 'EOF' > "$PROJECT_NAME/components/WalletConnectButton.tsx"
import React, { useState } from 'react';
import { ethers } from 'ethers';

export const WalletConnectButton: React.FC = () => {
  const [account, setAccount] = useState<string>('');

  const connectWallet = async () => {
    if (!window.ethereum) {
      alert('Metamask not installed!');
      return;
    }
    try {
      const provider = new ethers.BrowserProvider(window.ethereum);
      const accounts = await provider.send('eth_requestAccounts', []);
      setAccount(accounts[0]);
    } catch (err) {
      console.error('Failed to connect wallet:', err);
    }
  };

  return (
    <>
      {!account ? (
        <button onClick={connectWallet}>Connect Metamask</button>
      ) : (
        <p><strong>Wallet:</strong> {account}</p>
      )}
    </>
  );
};
EOF

# OPTIONAL placeholders for other pages
cat << 'EOF' > "$PROJECT_NAME/pages/providers.tsx"
import React from 'react';

export default function ProvidersPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Providers</h1>
      <p>Placeholder page for listing providers.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/registerProvider.tsx"
import React from 'react';

export default function RegisterProviderPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Register Provider</h1>
      <p>Placeholder page for registering a provider.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/registerModel.tsx"
import React from 'react';

export default function RegisterModelPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Register Model</h1>
      <p>Placeholder page for registering a model.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/registerBid.tsx"
import React from 'react';

export default function RegisterBidPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Register Bid</h1>
      <p>Placeholder page for registering a bid.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/troubleshooting.tsx"
import React from 'react';

export default function TroubleshootingPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Troubleshooting Guide</h1>
      <p>Placeholder docs for troubleshooting steps.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/advanced-configuration.tsx"
import React from 'react';

export default function AdvancedConfigPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Advanced Configuration</h1>
      <p>Placeholder docs for advanced config steps.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/direct-consumer.tsx"
import React from 'react';

export default function DirectConsumerPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Direct Consumer Interaction</h1>
      <p>Placeholder docs for direct consumer usage with proxy-router.</p>
    </div>
  );
}
EOF

cat << 'EOF' > "$PROJECT_NAME/pages/shared-blockchain-proxy.tsx"
import React from 'react';

export default function SharedBlockchainProxyPage() {
  return (
    <div style={{ padding: '1rem' }}>
      <h1>Shared, Blockchain &amp; Proxy Docs</h1>
      <p>Placeholder doc with references to Blockchain/Proxy resources.</p>
    </div>
  );
}
EOF

echo "Step 3 complete! Created files in '$PROJECT_NAME'."
echo "You can now open them and replace placeholder code with your actual Next.js code."
echo "Or, run 'npm install' and 'npm run dev' to see the placeholders in action."
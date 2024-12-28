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

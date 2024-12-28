// global.d.ts
// Ensures TypeScript recognizes window.ethereum

export {};

declare global {
  interface Window {
    ethereum?: any;
  }
}

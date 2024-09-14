import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";
import '@typechain/hardhat'
import '@nomicfoundation/hardhat-ethers'
import '@nomicfoundation/hardhat-chai-matchers'


const config: HardhatUserConfig = {
  solidity: "0.8.24",
  networks: {
    hardhat: {
      chainId: 1337, // Specify chain ID for Hardhat Network
    },
    localhost: {
      url: "http://127.0.0.1:8545", // Localhost network (used with `npx hardhat node`)
      chainId: 1337,
      accounts: [
        "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
        "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
      ]
    },
    // Example for configuring a testnet like Ropsten
    // arbitrum: {
    //   url: "http://localhost:8547",
    //   accounts: [`0x${process.env.ARBITRUM_PRIVATE_KEY}`], // Replace with your private key
    // },
  },
};

export default config;

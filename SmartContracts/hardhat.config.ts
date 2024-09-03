import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: "0.8.24",
  networks: {
    hardhat: {
      chainId: 1337, // Specify chain ID for Hardhat Network
    },
    localhost: {
      url: "http://127.0.0.1:8545", // Localhost network (used with `npx hardhat node`)
      chainId: 1337,
    },
    // Example for configuring a testnet like Ropsten
    arbitrum: {
      url: "http://localhost:8547",
      accounts: [`0x${process.env.HARDHAT_PRIVATE_KEY}`], // Replace with your private key
    },
  },
};

export default config;

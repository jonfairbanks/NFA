import { ethers } from "hardhat";
import { createTestNFA } from "../lib/createTestContract";

async function main() {
  const [deployer] = await ethers.getSigners();
  process.env.TEST_NFA_ADDRESS = await createTestNFA(deployer);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });

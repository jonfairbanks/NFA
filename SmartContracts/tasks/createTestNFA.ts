import fs from "fs";
import { requireEnvsSet } from "../lib/utils";
import { ethers } from "hardhat";
import { createTestNFA } from "../lib/createTestContract";

async function main() {
  const [deployer] = await ethers.getSigners();
  const address = await createTestNFA(deployer);
  fs.writeFileSync("test-nfa-addr.tmp", address);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
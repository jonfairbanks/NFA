import { ethers } from "hardhat";

export async function getContractFactory() {
  const [deployer] = await ethers.getSigners();
  
  console.log("Deploying contracts with the account:", deployer.address);

  const balance = await deployer.provider.getBalance(deployer.address)
  console.log("Account balance:", balance.toString());

  // Deploy NFAFactory contract
  return await ethers.getContractFactory("NFAFactory");
}

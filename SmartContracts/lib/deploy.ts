import fs from "fs"
import { ethers } from "hardhat"
import { requireEnvsSet } from "./utils"

export async function deploy() {
    const [deployer] = await ethers.getSigners();

    console.log("Deploying contracts with the account:", deployer.address);

    const balance = await deployer.provider.getBalance(deployer.address);
    console.log("Account balance:", balance.toString());

    // Deploy NFAFactory contract
    const NFAFactoryContract = await ethers.getContractFactory("NFAFactory");
    const nfaFactoryContract = await NFAFactoryContract.deploy();

    nfaFactoryContract.initialize(deployer.address);

    const address = await nfaFactoryContract.getAddress()
    console.log("Contract deployed to:", address);

    fs.writeFileSync("nfa-factory-addr.tmp", address);
    // Deploy NFA contract
    const NFAContract = await ethers.getContractFactory("NFA");
    const nfaContract = await NFAContract.deploy();
    console.log("Contract deployed to:", await nfaContract.getAddress());

  }
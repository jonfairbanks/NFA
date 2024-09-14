import { ethers } from "hardhat"

export async function deploy() {
    const [deployer] = await ethers.getSigners();

    console.log("Deploying contracts with the account:", deployer.address);

    const balance = await deployer.provider.getBalance(deployer.address);
    console.log("Account balance:", balance.toString());

    // Deploy NFAFactory contract
    const NFAFactoryContract = await ethers.getContractFactory("NFAFactory");
    const nfaFactoryContract = await NFAFactoryContract.deploy();

    process.env.NFA_FACTORY_CONTRACT_ADDRESS = await nfaFactoryContract.getAddress()
    console.log("Contract deployed to:", process.env.NFA_FACTORY_CONTRACT_ADDRESS);

    // Deploy NFA contract
    const NFAContract = await ethers.getContractFactory("NFA");
    const nfaContract = await NFAContract.deploy();
    console.log("Contract deployed to:", await nfaContract.getAddress());

  }
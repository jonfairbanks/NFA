async function main() {
    const [deployer] = await ethers.getSigners();

    console.log("Deploying contracts with the account:", deployer.address);

    const balance = await deployer.getBalance();
    console.log("Account balance:", balance.toString());

    // Deploy NFAFactory contract
    const NFAFactoryContract = await ethers.getContractFactory("NFAFactory");
    const nfaFactoryContract = await NFAFactoryContract.deploy();

    process.env.NFA_FACTORY_CONTRACT_ADDRESS = nfaFactoryContract.address
    console.log("Contract deployed to:", nfaFactoryContract.address);

    // Deploy NFA contract
    const NFAContract = await ethers.getContractFactory("NFA");
    const nfaContract = await NFAContract.deploy();
    console.log("Contract deployed to:", nfaContract.address);

    
  }
  
  main()
    .then(() => process.exit(0))
    .catch(error => {
      console.error(error);
      process.exit(1);
    });

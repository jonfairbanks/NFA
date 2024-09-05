async function main() {
    const [deployer] = await ethers.getSigners();

    console.log("Deploying contracts with the account:", deployer.address);

    const balance = await deployer.getBalance();
    console.log("Account balance:", balance.toString());

    // Deploy NFAFactory contract
    const NFAFactoryContract = await ethers.getContractFactory("NFAFactory");
    const nfaFactoryContract = await NFAFactoryContract.attach(process.env.NFA_CONTRACT_ADDRESS);

    console.log("attached nfa contract address:", nfaFactoryContract.address);

    // nfaContract

    const appInfo = new AppInfo(false, "free", [""], "", [""], [""], "0.0.1");
    const versionInfo = new VersionInfo("0.0.1", [""], "", [""], [""]);

    nfaFactoryContract.createNFAContract("Test NFA", "TST", appInfo, versionInfo, deployer.address);

    process.env.NFA_CONTRACT_ADDRESS = nfaFactoryContract.address

    
  }

  main()
    .then(() => process.exit(0))
    .catch(error => {
      console.error(error);
      process.exit(1);
    });
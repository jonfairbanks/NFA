import { getContractFactory } from "./getContractFactory";
// import {} from "hardhat/types"

export async function createTestNFA(deployer) {
  const NFAFactoryContract = await getContractFactory();
console.log("environment NFA_FACTORY_CONTRACT_ADDRESS: ", process.env.NFA_FACTORY_CONTRACT_ADDRESS)
  const nfaFactoryContract = await NFAFactoryContract.attach(
    process.env.NFA_FACTORY_CONTRACT_ADDRESS
  );

  console.log("attached nfa contract address:", nfaFactoryContract.address);

  // nfaContract
  const appInfo = new AppInfo(false, "free", ["https://github.com/MORpheus-Software/NFA"], "518c4bf773cea6b73b940ff8525167d33343aecfef4edb56d928fd77aa6d89f1", [""], [""], "0.0.1");
  const versionInfo = new VersionInfo("0.0.1", [""], "", [""], "");

  return await nfaFactoryContract.createNFAContract(
    "Test NFA",
    "TST",
    appInfo,
    versionInfo,
    deployer.address
  );
}
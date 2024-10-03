import { ethers } from "hardhat";
import { getContractFactory } from "./getContractFactory";
import { requireEnvsSet } from "./utils";
import { AppInfo, VersionInfo } from "./dto";

export async function createTestNFA(deployer) {
  const NFAFactoryContract = await getContractFactory();
  console.log(
    "NFA_FACTORY_CONTRACT_ADDRESS: ",
    process.env.NFA_FACTORY_CONTRACT_ADDRESS
  );

  const env = requireEnvsSet("NFA_FACTORY_CONTRACT_ADDRESS");

  console.log(
    "environment NFA_FACTORY_CONTRACT_ADDRESS: ",
    env.NFA_FACTORY_CONTRACT_ADDRESS
  );

  const nfaFactoryContract = await NFAFactoryContract.attach(
    env.NFA_FACTORY_CONTRACT_ADDRESS
  );

  console.log("attached nfa contract address:", await nfaFactoryContract.getAddress());

  // nfaContract
  const versionInfo = new VersionInfo(
    "0.0.1",
    ["https://github.com/MORpheus-Software/NFA"],
    `0x${"518c4bf773cea6b73b940ff8525167d33343aecfef4edb56d928fd77aa6d89f1"}`,
    [""],
    "0x0000000000000000000000000000000000000000000000000000000000000000"    
  );

  const appInfo = new AppInfo(false, "free", versionInfo);
 
  const createNFAContractTx = await nfaFactoryContract.createNFAContract(
    "Test NFA",
    "TST",
    appInfo,
    versionInfo,
    deployer.address
  );

  // console.log("create nfa contract tx: ", createNFAContractTx);

  await createNFAContractTx.wait()

  // console.log("create nfa contract tx receipt: ", createNFAContractTxReceipt);

  //NFAContractCreated
  
  const logs = await nfaFactoryContract.queryFilter("NFAContractCreated");
// console.log("NFAContractCreated log: ", logs);
  const newContractAddress = logs[0].args[0];

  // console.log("New NFA Contract Address:", newContractAddress);
  return newContractAddress;
}
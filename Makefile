initSmartContracts:
	make deployContracts
	make createTestNFA

runBlockchainLocal:
	cd ./SmartContracts && ./runLocalTestNode.sh
testContracts:
	cd ./SmartContracts && npx hardhat test --network localhost
deployContracts:
	cd ./SmartContracts && npx hardhat run tasks/deploy.ts --network localhost
createTestNFA:
	cd ./SmartContracts && npx hardhat run tasks/createTestNFA.ts --network localhost

TEMP_FILE_PATH ?= ./temp-files
REPOS ?= 

hashRepos:
	npx ts-node ./Utilities/hashRepos.ts $(TEMP_FILE_PATH) $(REPOS)

FILE_DOWNLOADS ?= 

hashFiles:
	npx ts-node ./Utilities/hashFiles.ts $(TEMP_FILE_PATH) $(FILE_DOWNLOADS)
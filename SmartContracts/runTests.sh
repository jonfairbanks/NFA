NFA_FACTORY_CONTRACT_ADDRESS="$(cat ./nfa-factory-addr.tmp)" \
TEST_NFA_ADDRESS="$(cat ./test-nfa-addr.tmp)" \
npx hardhat test --network localhost
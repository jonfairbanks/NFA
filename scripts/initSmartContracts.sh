make deployContracts
export NFA_FACTORY_CONTRACT_ADDRESS="$(cat ./SmartContracts/nfa-factory-addr.tmp)"
make createTestNFA
export TEST_NFA_ADDRESS="$(cat ./SmartContracts/test-nfa-addr.tmp)"
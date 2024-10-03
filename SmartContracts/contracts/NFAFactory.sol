// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721URIStorageUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "./AppInfo.sol";
import "./VersionInfo.sol";
import "./NFA.sol";

contract NFAFactory is OwnableUpgradeable {
    event NFAContractCreated(address indexed nfaAddress);

    function initialize(address owner) public initializer {
        __Ownable_init(owner);
    }

    function createNFAContract(
        string memory name,
        string memory symbol,
        AppInfo memory appInfo,
        address initialOwner
    ) public onlyOwner returns (address) {
        NFA newNFA = new NFA();
        newNFA.initialize(name, symbol, appInfo, initialOwner); // Pass the initial owner
        emit NFAContractCreated(address(newNFA));

        return address(newNFA);
    }
}

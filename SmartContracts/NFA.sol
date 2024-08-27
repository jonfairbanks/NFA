// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

contract NFAToken is ERC721, Ownable {
    using Counters for Counters.Counter;
    Counters.Counter private _tokenIds;

    // Struct to hold the NFA configuration
    struct NFAConfig {
        uint256 appId;
        uint256 ownerId;
        string githubUrl;
        string clusterConfig;
    }

    // Mapping from token ID to NFAConfig
    mapping(uint256 => NFAConfig) private _nfaConfigs;

    constructor() ERC721("NFAToken", "NFA") {}

    // Function to mint a new NFT
    function mintNFA(uint256 appId, uint256 ownerId, string memory githubUrl, string memory clusterConfig) public onlyOwner returns (uint256) {
        _tokenIds.increment();
        uint256 newTokenId = _tokenIds.current();

        // Store the NFAConfig on-chain
        _nfaConfigs[newTokenId] = NFAConfig(appId, ownerId, githubUrl, clusterConfig);

        // Mint the NFT
        _mint(msg.sender, newTokenId);

        return newTokenId;
    }

    // Function to retrieve the NFAConfig for a given token ID
    function getNFAConfig(uint256 tokenId) public view returns (uint256, uint256, string memory) {
        require(_exists(tokenId), "Token ID does not exist.");
        NFAConfig memory config = _nfaConfigs[tokenId];
        return (config.appId, config.ownerId, config.githubUrl);
    }
}

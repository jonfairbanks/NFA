// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721URIStorageUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";

struct AppInfo {
    bool routerRequired;
    string paymentModel;
    string[] downloadURIs;
    bytes32 codeHash;
    string[] abiURIs;
    bytes32 abiHash;
    string versionId;
}

struct VersionInfo {
    string versionId;
    string[] downloadURIs;
    bytes32 codeHash;
    string[] abiURIs;
    bytes32 abiHash;
}

contract NFA is
    Initializable,
    ERC721URIStorageUpgradeable,
    OwnableUpgradeable,
    UUPSUpgradeable
{
    uint256 private _tokenIds;
    mapping(uint256 => AppInfo) private appInfos;
    mapping(string => VersionInfo) public nfaVersions;

    event NFACreated(uint256 indexed tokenId);
    event VersionUpdated(uint256 indexed tokenId, VersionInfo newVersion);
    event AppInfoUpdated(uint256 indexed tokenId, AppInfo newAppInfo);

    function initialize(
        string memory name,
        string memory symbol,
        AppInfo memory appInfo,
        VersionInfo memory versionInfo,
        address initialOwner
    ) public initializer {
        __ERC721_init(name, symbol);
        __ERC721URIStorage_init();
        __Ownable_init(initialOwner);
        appInfos[_tokenIds] = appInfo;
        versionInfo.versionId = appInfo.versionId;
    }

    function tokenExists(uint256 tokenId) public view returns (bool) {
        return ownerOf(tokenId) != address(0);
    }

    function mint(
        address to,
        string memory tokenURI,
        AppInfo memory appInfo,
        string memory versionId
    ) public onlyOwner returns (uint256) {
        _tokenIds += 1;
        uint256 newItemId = _tokenIds;
        appInfos[newItemId] = appInfo;

        _mint(to, newItemId);
        _setTokenURI(newItemId, tokenURI);
        nfaVersions[versionId] = VersionInfo(
            appInfo.versionId,
            appInfo.downloadURIs,
            appInfo.codeHash,
            appInfo.abiURIs,
            appInfo.abiHash
        );

        emit NFACreated(newItemId);

        return newItemId;
    }

    function updateVersion(
        uint256 tokenId,
        VersionInfo memory newVersion,
        string memory versionId
    ) public onlyOwner {
        require(
            tokenExists(tokenId),
            "NFA: Version update for nonexistent token"
        );
        nfaVersions[versionId] = newVersion;
        emit VersionUpdated(tokenId, newVersion);
    }

    function updateAppInfo(
        uint256 tokenId,
        AppInfo memory newAppInfo
    ) public onlyOwner {
        require(
            tokenExists(tokenId),
            "NFA: AppInfo update for nonexistent token"
        );
        appInfos[tokenId] = newAppInfo;
        emit AppInfoUpdated(tokenId, newAppInfo);
    }

    function getAppInfo(uint256 tokenId) public view returns (AppInfo memory) {
        require(
            tokenExists(tokenId),
            "NFA: AppInfo query for nonexistent token"
        );
        return appInfos[tokenId];
    }

    function getVersionInfo(
        uint256 tokenId
    ) public view returns (VersionInfo memory) {
        require(
            tokenExists(tokenId),
            "NFA: Version query for nonexistent token"
        );
        return nfaVersions[appInfos[tokenId].versionId];
    }

    function withdraw() public onlyOwner {
        //Should we specify the amount to withdraw in the parameter?
        uint256 balance = address(this).balance;
        require(balance > 0, "NFA: No funds available for withdrawal");
        payable(owner()).transfer(balance);
    }

    receive() external payable {}
    fallback() external payable {}

    function _authorizeUpgrade(
        address newImplementation
    ) internal override onlyOwner {}
}

contract NFAFactory is OwnableUpgradeable {
    event NFAContractCreated(address indexed nfaAddress);

    function createNFAContract(
        string memory name,
        string memory symbol,
        AppInfo memory appInfo,
        VersionInfo memory versionInfo,
        address initialOwner
    ) public onlyOwner returns (address) {
        NFA newNFA = new NFA();
        newNFA.initialize(name, symbol, appInfo, versionInfo, initialOwner); // Pass the initial owner
        emit NFAContractCreated(address(newNFA));

        return address(newNFA);
    }
}


// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

struct VersionInfo {
    string versionId;
    string[] downloadURIs;
    bytes32 codeHash;
    string[] abiURIs;
    bytes32 abiHash;
}
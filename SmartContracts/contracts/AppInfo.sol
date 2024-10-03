// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import "./VersionInfo.sol";

struct AppInfo {
    bool routerRequired;
    string paymentModel;
    VersionInfo versionInfo;
}
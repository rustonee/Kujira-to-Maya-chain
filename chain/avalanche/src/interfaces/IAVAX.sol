// SPDX-License-Identifier: AGPL-3.0-or-later
pragma solidity 0.8.9;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

interface IAVAX is IERC20 {
    function deposit() external payable;

    function withdraw(uint256) external;
}

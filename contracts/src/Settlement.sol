// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IERC20} from "./IERC20.sol";

/// @title Settlement
/// @notice Lesson 6's on-chain settlement point. It is the last hop of
/// UCP -> AP2 -> X402: by the time a transaction reaches `pay`, the backend
/// has already verified an AP2 payment mandate and quoted the exact amount
/// via an X402 402 response. This contract does not know about mandates —
/// it only knows how to pull USDC from the payer to the merchant exactly
/// once per order and emit a receipt the backend can verify from the tx log,
/// which is deliberately the smallest amount of on-chain trust needed.
contract Settlement {
    IERC20 public immutable usdc;
    address public immutable merchant;

    mapping(bytes32 => bool) public paid;

    event OrderSettled(bytes32 indexed orderId, address indexed payer, uint256 amount);

    error AlreadyPaid(bytes32 orderId);
    error ZeroAmount();

    constructor(address usdcAddress, address merchantAddress) {
        usdc = IERC20(usdcAddress);
        merchant = merchantAddress;
    }

    /// @notice Pulls `amount` USDC from msg.sender (the payer must have
    /// called `usdc.approve(settlement, amount)` first) and forwards it to
    /// the merchant. Reverts if this orderId was already settled, so a
    /// replayed X-Payment tx hash can never double-charge.
    function pay(bytes32 orderId, uint256 amount) external {
        if (amount == 0) revert ZeroAmount();
        if (paid[orderId]) revert AlreadyPaid(orderId);
        paid[orderId] = true;

        require(usdc.transferFrom(msg.sender, merchant, amount), "USDC transferFrom failed");
        emit OrderSettled(orderId, msg.sender, amount);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {Settlement} from "../src/Settlement.sol";

/// Usage (against a local anvil fork, see contracts/README.md):
///   forge script script/Deploy.s.sol --rpc-url http://127.0.0.1:8545 \
///     --private-key $DEPLOYER_KEY --broadcast
contract Deploy is Script {
    address constant USDC = 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48;

    function run() external {
        address merchant = vm.envOr("MERCHANT_WALLET_ADDRESS", address(0));
        require(merchant != address(0), "set MERCHANT_WALLET_ADDRESS");

        vm.startBroadcast();
        Settlement settlement = new Settlement(USDC, merchant);
        vm.stopBroadcast();

        console.log("Settlement deployed at:", address(settlement));
        console.log("Put this in .env as SETTLEMENT_CONTRACT_ADDRESS");
    }
}

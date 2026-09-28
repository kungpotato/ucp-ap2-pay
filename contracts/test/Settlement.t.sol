// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {Settlement} from "../src/Settlement.sol";
import {IERC20} from "../src/IERC20.sol";

/// @notice Runs against a real mainnet fork (see contracts/README.md) so
/// USDC in this test is the actual mainnet token, not a mock — proving the
/// contract works against the real thing lesson 6 settles with.
contract SettlementTest is Test {
    address constant USDC = 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48;
    address constant USDC_WHALE = 0xF977814e90dA44bFA03b6295A0616a897441aceC;

    Settlement settlement;
    address merchant = makeAddr("merchant");
    address payer = makeAddr("payer");

    function setUp() public {
        settlement = new Settlement(USDC, merchant);

        // Seed the test payer with real mainnet-forked USDC by impersonating
        // a known large holder — the same anvil_impersonateAccount trick
        // lesson 6's docs walk through by hand with `cast`.
        vm.prank(USDC_WHALE);
        (bool ok,) = USDC.call(abi.encodeWithSignature("transfer(address,uint256)", payer, 50_000_000));
        require(ok, "seed transfer from whale failed");
    }

    function test_PaySettlesFullAmountToMerchant() public {
        bytes32 orderId = keccak256("order_demo");
        uint256 amount = 12_900_000; // 12.9 USDC, 6 decimals

        vm.startPrank(payer);
        IERC20(USDC).approve(address(settlement), amount);
        settlement.pay(orderId, amount);
        vm.stopPrank();

        assertEq(IERC20(USDC).balanceOf(merchant), amount, "merchant should receive the exact amount");
        assertTrue(settlement.paid(orderId), "order should be marked paid");
    }

    function test_CannotPayTheSameOrderTwice() public {
        bytes32 orderId = keccak256("order_replay");
        uint256 amount = 5_000_000;

        vm.startPrank(payer);
        IERC20(USDC).approve(address(settlement), amount * 2);
        settlement.pay(orderId, amount);

        vm.expectRevert(abi.encodeWithSelector(Settlement.AlreadyPaid.selector, orderId));
        settlement.pay(orderId, amount);
        vm.stopPrank();
    }

    function test_RevertsOnZeroAmount() public {
        vm.prank(payer);
        vm.expectRevert(Settlement.ZeroAmount.selector);
        settlement.pay(keccak256("order_zero"), 0);
    }
}

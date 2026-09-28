# contracts — lesson 6 on-chain settlement

`Settlement.sol` is the last hop of the workshop's payment chain: UCP → AP2
→ X402 → **this contract**. It pulls USDC from a payer to the merchant for
exactly one order, once, and emits an event the Go backend verifies from
the transaction receipt (`backend/x402.go`). See `../docs/lesson-06.md` for
the full write-up.

## Setup

```bash
foundryup                     # installs forge/anvil/cast if you don't have them
forge install foundry-rs/forge-std --no-commit   # contracts/lib/ is gitignored
forge build
```

## Fork mainnet locally

The whole point of this lesson is settling against the **real** USDC
contract, not a mock — so anvil forks live Ethereum mainnet state:

```bash
anvil --fork-url https://ethereum-rpc.publicnode.com --chain-id 31337 --port 8545
```

Any mainnet RPC works (Alchemy/Infura are more reliable than the free
public endpoint above if it rate-limits you). Set `MAINNET_RPC_URL` in the
repo root `.env` to override it — see `.env.example`.

## Fund a test wallet with real (forked) USDC

Nobody in this workshop owns mainnet USDC, so we borrow some from a wallet
that does, using anvil's `anvil_impersonateAccount` cheat — this only works
on the local fork, it never touches real mainnet:

```bash
WHALE=0xF977814e90dA44bFA03b6295A0616a897441aceC   # a large exchange hot wallet
USDC=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48
PAYER=<your anvil test account address>

cast rpc anvil_impersonateAccount $WHALE --rpc-url http://127.0.0.1:8545
cast send $USDC "transfer(address,uint256)(bool)" $PAYER 50000000 \
  --from $WHALE --unlocked --rpc-url http://127.0.0.1:8545
cast rpc anvil_stopImpersonatingAccount $WHALE --rpc-url http://127.0.0.1:8545
```

## Deploy

```bash
export MERCHANT_WALLET_ADDRESS=<address that should receive payments>
forge script script/Deploy.s.sol --rpc-url http://127.0.0.1:8545 \
  --private-key <deployer key, e.g. anvil account 0> --broadcast
```

Copy the printed address into the repo root `.env` as
`SETTLEMENT_CONTRACT_ADDRESS` (the Go backend reads it from there).

## Test

```bash
forge test --fork-url http://127.0.0.1:8545 -vv
```

`test/Settlement.t.sol` runs the same whale-impersonation funding flow
inside a Foundry test (`vm.prank`), so it's a reproducible, one-command
version of the manual steps above — proving `pay()` moves real forked USDC
and can't be replayed for the same order.

## Manually pay an order (what the agent does in lesson 6)

```bash
SETTLEMENT=<deployed address>
cast send $USDC "approve(address,uint256)(bool)" $SETTLEMENT <amount> \
  --private-key <payer key> --rpc-url http://127.0.0.1:8545
cast send $SETTLEMENT "pay(bytes32,uint256)" <orderId as bytes32> <amount> \
  --private-key <payer key> --rpc-url http://127.0.0.1:8545
```

The resulting transaction hash is what the frontend/agent sends back to
the backend as the `X-Payment` header on `POST /x402/orders/{id}/pay`.

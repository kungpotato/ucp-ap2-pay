# Agentic Commerce Bookstore — UCP → AP2 → X402 → On-Chain Settlement

A workshop that teaches building a bookstore system where **AI agents can autonomously purchase goods on behalf of humans**, from book discovery all the way to USDC arriving in the merchant wallet on a real blockchain (local anvil fork of Ethereum mainnet via Foundry) — zero mock steps.

![architecture](docs/diagrams/00-architecture.svg)

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go (pure `net/http` stdlib, zero external dependencies) |
| Frontend | Next.js (App Router) + Tailwind CSS |
| Agent wallet | Shells out to `cast` (Foundry) — see [lesson 6](docs/lesson-06.md) |
| Payment protocol | UCP → [AP2](https://github.com/google-agentic-commerce/AP2) mandates (ed25519) → [X402](https://x402.org) |
| Settlement | Solidity (`Settlement.sol`) on an anvil fork of Ethereum mainnet, real USDC |

## Lessons

| # | Topic | Demo |
|---|---|---|
| 0 | [Course Overview](docs/00-ภาพรวม-หลักสูตร.md) | — |
| 1 | [UCP: Catalog & Cart (Go Backend)](docs/lesson-01.md) | ![](docs/gifs/lesson-01.gif) |
| 2 | [UCP: Next.js Storefront](docs/lesson-02.md) | ![](docs/gifs/lesson-02.gif) |
| 3 | [AP2: Intent → Cart → Payment Mandate](docs/lesson-03.md) | ![](docs/gifs/lesson-03.gif) |
| 4 | [X402: Machine-to-Machine Agent Payments](docs/lesson-04.md) | ![](docs/gifs/lesson-04.gif) |
| 5 | [On-Chain Settlement (Foundry Mainnet Fork)](docs/lesson-05.md) | ![](docs/gifs/lesson-05.gif) |
| 6 | [Agent Self-Checkout: End-to-End Integration](docs/lesson-06.md) | ![](docs/gifs/lesson-06.gif) |

Each lesson covers **Goals, Why learn this, What you'll build, Architecture Diagrams, and Alternatives Considered** — not just a blind walkthrough. Follow them sequentially in the [`docs/`](docs/) directory.

## Getting Started (Local)

### 1. Foundry (anvil mainnet fork)

```bash
foundryup
anvil --fork-url https://ethereum-rpc.publicnode.com --chain-id 31337 --port 8545
```

### 2. Deploy Settlement.sol

```bash
cd contracts
forge install foundry-rs/forge-std --no-commit
export MERCHANT_WALLET_ADDRESS=<choose any address from anvil>
forge script script/Deploy.s.sol --rpc-url http://127.0.0.1:8545 \
  --private-key <anvil account 0 private key> --broadcast
```

Copy the deployed contract address and set it in `.env` as `SETTLEMENT_CONTRACT_ADDRESS` (see [`contracts/README.md`](contracts/README.md) on how to fund test wallets with real USDC from the mainnet fork).

### 3. Backend

```bash
cd backend
go run .
```

Runs on `:8080` and automatically loads `../.env`. Set `AGENT_PRIVATE_KEY` to enable the "Pay via Agent" button (lesson 6).

### 4. Frontend

```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev
```

Open `http://localhost:3000`.

## Generating Documentation GIFs (Reproducible Docs)

```bash
# Ensure anvil + backend + frontend are running first (ports 8545, 8080, 3000)
cd docs/scripts
npm install
node capture.mjs
```

See details in [lesson 6](docs/lesson-06.md).

## Project Structure

```
backend/    Go: UCP catalog/cart/order + AP2 mandates + X402 + agent wallet
frontend/   Next.js: Bookstore UI + mandate chain inspector + agent checkout button
contracts/  Foundry: Settlement.sol, tested against real mainnet fork
docs/       Lessons + diagrams (svg) + gifs + capture script
```

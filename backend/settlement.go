package main

// usdcMainnetAddress is the real USDC contract address on Ethereum
// mainnet. Because lesson 6's anvil node is forked *from* mainnet (not a
// clean chain with a fake token deployed), this is the actual token
// contract with its actual code and actual total supply — only the state
// changes we make locally (via anvil_impersonateAccount) are not real.
const usdcMainnetAddress = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"

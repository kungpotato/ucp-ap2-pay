package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
)

// orderSettledTopic0 = keccak256("OrderSettled(bytes32,address,uint256)"),
// precomputed with `cast keccak` (see contracts/README.md) so this backend
// never needs a Keccak-256 implementation of its own — just string
// comparison against what anvil returns in the transaction receipt.
const orderSettledTopic0 = "0x68e713d60e84c8b6b6860d62e99941708dcbaf7ecfdc28739e3986e1a2399914"

// X402PaymentRequired is the 402 response body: a simplified version of the
// x402 protocol's "accepts" array (see https://x402.org). Real x402 lets
// the client pay via an off-chain signed authorization a facilitator
// relays; this workshop has the agent submit the on-chain tx itself and
// present the resulting hash — see docs/lesson-05.md for the trade-off.
type X402PaymentRequired struct {
	X402Version int                    `json:"x402Version"`
	Accepts     []X402PaymentRequirement `json:"accepts"`
}

type X402PaymentRequirement struct {
	Scheme            string `json:"scheme"`  // "exact"
	Network           string `json:"network"` // "anvil-mainnet-fork"
	MaxAmountRequired string `json:"maxAmountRequired"` // USDC smallest unit, as decimal string
	Resource          string `json:"resource"`
	PayTo             string `json:"payTo"`             // Settlement contract address
	Asset             string `json:"asset"`             // USDC token address (mainnet)
	Extra             map[string]string `json:"extra"`  // orderId, settlementContract
}

// handleX402Pay is the machine-to-machine payment gate (lesson 5-6):
//   1. No X-Payment header  -> 402 with what to pay and where.
//   2. X-Payment: <tx hash> -> verify the on-chain receipt (lesson 6) and,
//      only if it really paid this exact order, mark it settled.
func (s *Server) handleX402Pay(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	order, ok := s.orders.Get(orderID)
	if !ok {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if order.PaymentMandateID == "" {
		writeError(w, http.StatusForbidden, "no payment mandate on this order yet (see /ap2/orders/{id}/payment-mandate)")
		return
	}
	if order.Status == OrderPaid {
		writeJSON(w, http.StatusOK, order)
		return
	}

	txHash := r.Header.Get("X-Payment")
	if txHash == "" {
		req := X402PaymentRequired{
			X402Version: 1,
			Accepts: []X402PaymentRequirement{{
				Scheme:            "exact",
				Network:           "anvil-mainnet-fork",
				MaxAmountRequired: fmt.Sprintf("%d", order.TotalUSDC),
				Resource:          "/x402/orders/" + orderID + "/pay",
				PayTo:             s.settlementAddress,
				Asset:             usdcMainnetAddress,
				Extra: map[string]string{
					"orderId":            orderIDToBytes32(orderID),
					"merchantWallet":     s.payTo,
					"settlementContract": s.settlementAddress,
				},
			}},
		}
		w.Header().Set("X-Payment-Required", "true")
		writeJSON(w, http.StatusPaymentRequired, req)
		return
	}

	receipt, err := s.chain.GetTransactionReceipt(txHash)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not verify payment: "+err.Error())
		return
	}
	if receipt.Status != "0x1" {
		writeError(w, http.StatusPaymentRequired, "on-chain transaction reverted")
		return
	}

	wantOrderID := orderIDToBytes32(orderID)
	var paidAmount *big.Int
	for _, lg := range receipt.Logs {
		if !strings.EqualFold(lg.Address, s.settlementAddress) {
			continue
		}
		if len(lg.Topics) < 2 || !strings.EqualFold(lg.Topics[0], orderSettledTopic0) {
			continue
		}
		if !strings.EqualFold(strings.TrimPrefix(lg.Topics[1], "0x"), strings.TrimPrefix(wantOrderID, "0x")) {
			continue
		}
		// orderId and payer are both `indexed` (they live in topics[1] and
		// topics[2]), so data holds only the non-indexed amount: one word.
		data := strings.TrimPrefix(lg.Data, "0x")
		if len(data) < 64 {
			continue
		}
		amountHex := data[0:64]
		amt := new(big.Int)
		amt.SetString(amountHex, 16)
		paidAmount = amt
		break
	}

	if paidAmount == nil {
		writeError(w, http.StatusPaymentRequired, "no matching OrderSettled event for this order in that transaction")
		return
	}
	if paidAmount.Cmp(big.NewInt(order.TotalUSDC)) < 0 {
		writeError(w, http.StatusPaymentRequired, "on-chain payment amount is less than the order total")
		return
	}

	s.orders.Update(orderID, func(o *Order) {
		o.Status = OrderPaid
		o.TxHash = txHash
	})
	order, _ = s.orders.Get(orderID)
	writeJSON(w, http.StatusOK, order)
}

// orderIDToBytes32 turns "order_ab12cd34..." into a left-padded bytes32 hex
// string the same way the Settlement contract's Solidity caller does
// (see contracts/script/Pay.s.sol) — right-aligning the raw id bytes.
func orderIDToBytes32(orderID string) string {
	b := []byte(orderID)
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return "0x" + hex.EncodeToString(padded)
}

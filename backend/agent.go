package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
)

// This is agentic commerce, so it's the agent — not a human clicking
// "connect wallet" in the browser — that holds the money and submits the
// on-chain payment. AgentWallet shells out to `cast` (already on the
// workshop machine from `foundryup`) instead of vendoring a secp256k1 +
// RLP transaction signer into this zero-dependency Go backend. A real
// agent would sign with a proper wallet SDK/HSM; see docs/lesson-06.md for
// that trade-off.
type AgentWallet struct {
	privateKey string
	rpcURL     string
	castBin    string
}

func NewAgentWallet(privateKey, rpcURL string) *AgentWallet {
	return &AgentWallet{privateKey: privateKey, rpcURL: rpcURL, castBin: findCast()}
}

func findCast() string {
	if p, err := exec.LookPath("cast"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	candidate := home + "/.foundry/bin/cast"
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return "cast" // let exec.Command fail loudly with a clear error if truly missing
}

func (a *AgentWallet) cast(args ...string) (string, error) {
	full := append(args, "--rpc-url", a.rpcURL)
	cmd := exec.Command(a.castBin, full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cast %v: %w: %s", args[:1], err, stderr.String())
	}
	return stdout.String(), nil
}

// send calls `cast send`, extracting the transaction hash from --json output.
func (a *AgentWallet) send(to string, sig string, args ...string) (string, error) {
	callArgs := append([]string{"send", to, sig}, args...)
	callArgs = append(callArgs, "--private-key", a.privateKey, "--json")
	out, err := a.cast(callArgs...)
	if err != nil {
		return "", err
	}
	var parsed struct {
		TransactionHash string `json:"transactionHash"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil || parsed.TransactionHash == "" {
		return "", fmt.Errorf("cast send did not return a transaction hash: %s", out)
	}
	return parsed.TransactionHash, nil
}

// handleAgentCheckout is the one button a human clicks: "let the agent pay
// this order". It replays what a real agent does by hand in lesson 6 —
// approve the Settlement contract, call pay(orderId, amount), then present
// the resulting tx hash back to X402 for verification — all server-side.
func (s *Server) handleAgentCheckout(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if s.agent == nil {
		writeError(w, http.StatusServiceUnavailable, "AGENT_PRIVATE_KEY is not configured on this backend")
		return
	}

	quoteReq, _ := http.NewRequest(http.MethodPost, "", nil)
	quoteReq.SetPathValue("id", orderID)
	quoteRec := &responseRecorder{}
	s.handleX402Pay(quoteRec, quoteReq)
	if quoteRec.status != http.StatusPaymentRequired {
		// Already paid, or not eligible yet (no payment mandate) — surface
		// whatever handleX402Pay already decided.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(quoteRec.status)
		_, _ = w.Write(quoteRec.body)
		return
	}

	var quote X402PaymentRequired
	if err := json.Unmarshal(quoteRec.body, &quote); err != nil || len(quote.Accepts) == 0 {
		writeError(w, http.StatusBadGateway, "could not parse X402 quote")
		return
	}
	accept := quote.Accepts[0]

	if _, err := a402send(s.agent, accept.Asset, "approve(address,uint256)(bool)", accept.PayTo, accept.MaxAmountRequired); err != nil {
		writeError(w, http.StatusBadGateway, "agent approve failed: "+err.Error())
		return
	}
	txHash, err := a402send(s.agent, accept.PayTo, "pay(bytes32,uint256)", accept.Extra["orderId"], accept.MaxAmountRequired)
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent pay failed: "+err.Error())
		return
	}

	verifyReq, _ := http.NewRequest(http.MethodPost, "", nil)
	verifyReq.SetPathValue("id", orderID)
	verifyReq.Header.Set("X-Payment", txHash)
	verifyRec := &responseRecorder{}
	s.handleX402Pay(verifyRec, verifyReq)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(verifyRec.status)
	_, _ = w.Write(verifyRec.body)
}

func a402send(a *AgentWallet, to, sig string, args ...string) (string, error) {
	return a.send(to, sig, args...)
}

// responseRecorder is a minimal http.ResponseWriter so handleAgentCheckout
// can call handleX402Pay in-process instead of making an HTTP hop to
// itself.
type responseRecorder struct {
	status int
	body   []byte
	header http.Header
}

func (rr *responseRecorder) Header() http.Header {
	if rr.header == nil {
		rr.header = http.Header{}
	}
	return rr.header
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body = append(rr.body, b...)
	return len(b), nil
}

func (rr *responseRecorder) WriteHeader(status int) {
	rr.status = status
}

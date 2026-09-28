package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ChainClient is a minimal JSON-RPC client — just enough eth_* calls to
// verify an on-chain settlement (lesson 6), with zero external
// dependencies. It talks to the anvil node forked from mainnet.
type ChainClient struct {
	rpcURL string
	http   *http.Client
}

func NewChainClient(rpcURL string) *ChainClient {
	return &ChainClient{rpcURL: rpcURL, http: &http.Client{}}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *ChainClient) call(method string, params []any, out any) error {
	body, _ := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	resp, err := c.http.Post(c.rpcURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("rpc call %s: %w", method, err)
	}
	defer resp.Body.Close()
	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return fmt.Errorf("rpc decode %s: %w", method, err)
	}
	if rr.Error != nil {
		return fmt.Errorf("rpc error %s: %s", method, rr.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rr.Result, out)
}

// TxLog is the sliver of an Ethereum log entry this workshop needs to
// recognise the Settlement contract's OrderSettled event.
type TxLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

type TxReceipt struct {
	Status string  `json:"status"` // "0x1" success, "0x0" reverted
	Logs   []TxLog `json:"logs"`
}

func (c *ChainClient) GetTransactionReceipt(txHash string) (*TxReceipt, error) {
	var receipt TxReceipt
	if err := c.call("eth_getTransactionReceipt", []any{txHash}, &receipt); err != nil {
		return nil, err
	}
	if receipt.Status == "" {
		return nil, fmt.Errorf("transaction %s not yet mined", txHash)
	}
	return &receipt, nil
}

func (c *ChainClient) ChainID() (int64, error) {
	var hex string
	if err := c.call("eth_chainId", nil, &hex); err != nil {
		return 0, err
	}
	return parseHexInt(hex)
}

func parseHexInt(hex string) (int64, error) {
	hex = strings.TrimPrefix(hex, "0x")
	if hex == "" {
		return 0, nil
	}
	return strconv.ParseInt(hex, 16, 64)
}

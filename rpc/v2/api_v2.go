/*
	SCHEMA_V2: dict =
		"icx_sendTransaction": icx_sendTransaction_v2,
		"icx_getTransactionResult": icx_getTransactionResult_v2,
		"icx_getBalance": icx_getBalance_v2,
		"icx_getTotalSupply": icx_getTotalSupply,
		"icx_getLastBlock": icx_getLastBlock,
		"icx_getBlockByHash": icx_getBlockByHash_v2,
		"icx_getBlockByHeight": icx_getBlockByHeight_v2,
		// Deprecated
		"icx_getTransactionByAddress": icx_getTransactionByAddress_v2
	}
*/
package v2

import (
	"github.com/icon-project/goloop/module"
	"github.com/osamingo/jsonrpc"
)

const (
	sendTransaction      string = "icx_sendTransaction"
	getTransactionResult string = "icx_getTransactionResult"
	getBalance           string = "icx_getBalance"
	getTotalSupply       string = "icx_getTotalSupply"
	getLastBlock         string = "icx_getLastBlock"
	getBlockByHash       string = "icx_getBlockByHash"
	getBlockByHeight     string = "icx_getBlockByHeight"
)

func MethodRepository(bm module.BlockManager, sm module.ServiceManager) *jsonrpc.MethodRepository {

	v2 := jsonrpc.NewMethodRepository()

	// api v2
	v2.RegisterMethod(sendTransaction, sendTransactionHandler{sm: sm}, sendTransactionParam{}, sendTranscationResult{})
	v2.RegisterMethod(getTransactionResult, getTransactionResultHandler{}, getTransactionResultParam{}, getTransactionResultResult{})
	v2.RegisterMethod(getBalance, getBalanceHandler{}, getBalanceParam{}, getBalanceResult{})
	v2.RegisterMethod(getTotalSupply, getTotalSupplyeHandler{}, nil, getTotalSupplyResult{})
	v2.RegisterMethod(getLastBlock, getLastBlockHandler{bm: bm}, nil, blockResult{})
	v2.RegisterMethod(getBlockByHash, getBlockByHashHandler{bm: bm}, getBlockByHashParam{}, blockResult{})
	v2.RegisterMethod(getBlockByHeight, getBlockByHeightHandler{bm: bm}, getBlockByHeightParam{}, blockResult{})

	return v2
}

package service

import (
	"errors"
	"github.com/icon-project/goloop/module"
	"math/big"
)

const (
	GIGA = 1000 * 1000 * 1000
	TERA = 1000 * GIGA
	PETA = 1000 * TERA
	EXA  = 1000 * PETA
)

var (
	ErrNotEnoughBalance   = errors.New("NotEnoughBalance")
	ErrTimeOut            = errors.New("TimeOut")
	ErrInvalidFeeValue    = errors.New("InvalidFeeValue")
	ErrNotEnoughStep      = errors.New("NotEnoughStep")
	ErrContractIsRequired = errors.New("ContractIsRequired")
	ErrInvalidHashValue   = errors.New("InvalidHashValue")
)

type WorldContext interface {
	WorldState
	StepPrice() *big.Int
	TimeStamp() int64
	BlockHeight() uint64
	Treasury() module.Address
	WorldStateChanged(ws WorldState) WorldContext
}

type Transaction interface {
	module.Transaction
	PreValidate(wc WorldContext, update bool) error
	Prepare(wvs WorldVirtualState) (WorldVirtualState, error)
	Execute(wc WorldContext) (Receipt, error)
	Timestamp() int64
}

type Receipt interface {
	module.Receipt
	AddLog(addr module.Address, indexed, data []string)
	SetCumulativeStepUsed(cumulativeUsed *big.Int)
	SetResult(success bool, used, price *big.Int)
	CumulativeStepUsed() *big.Int
	StepPrice() *big.Int
	StepUsed() *big.Int
	Success() bool
}

package service

import (
	"bytes"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"log"
	"math/big"
	"testing"
)

func TestReceiptList(t *testing.T) {
	rslice := make([]Receipt, 0)

	var used, price big.Int

	addr1 := common.NewAddressFromString("hx8888888888888888888888888888888888888888")
	for i := 0; i < 5; i++ {
		r1 := NewReceipt(addr1)
		used.SetInt64(int64(i * 100))
		price.SetInt64(int64(i * 10))
		r1.SetResult(module.StatusOutOfBalance, &used, &price, nil)
		rslice = append(rslice, r1)
	}
	addr2 := common.NewAddressFromString("cx0003737589788888888888888888888888888888")
	for i := 0; i < 5; i++ {
		r2 := NewReceipt(addr2)
		used.SetInt64(int64(i * 100))
		price.SetInt64(int64(i * 10))
		r2.SetResult(module.StatusOutOfBalance, &used, &price, nil)
		rslice = append(rslice, r2)
	}

	mdb := db.NewMapDB()

	rl1 := NewReceiptListFromSlice(mdb, rslice)
	hash := rl1.Hash()
	rl1.Flush()

	log.Printf("Hash for receipts : <%x>", hash)

	rl2 := NewReceiptListFromHash(mdb, hash)
	idx := 0
	for itr := rl2.Iterator(); itr.Has(); itr.Next() {
		r, err := itr.Get()
		if err != nil {
			t.Errorf("Fail on Get Receipt from ReceiptList from Hash err=%+v", err)
		}
		if !bytes.Equal(rslice[idx].Bytes(), r.Bytes()) {
			t.Errorf("Fail on comparing bytes for Receipt[%d]", idx)
		}
		if err := rslice[idx].Check(r); err != nil {
			t.Errorf("Fail on Check Receipt[%d] err=%+v", idx, err)
		}
		idx++
	}
}

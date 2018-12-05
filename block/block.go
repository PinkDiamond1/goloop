package block

import (
	"bytes"
	"errors"

	"github.com/icon-project/goloop/module"
)

func verifyBlock(b module.Block, prev module.Block, validators module.ValidatorList) error {
	if b.Height() != prev.Height()+1 {
		return errors.New("bad height")
	}
	if !bytes.Equal(b.PrevID(), prev.ID()) {
		return errors.New("bad prev ID")
	}
	if err := b.Votes().Verify(prev, validators); err != nil {
		return err
	}
	return nil
}

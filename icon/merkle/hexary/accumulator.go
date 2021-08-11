/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package hexary

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

const (
	defaultAccumulatorKey = "accumulator"
)

type Accumulator interface {
	Add(hash []byte) error

	// Len returns number of added hashes.
	Len() int64

	GetMerkleHeader() *MerkleHeader

	// Finalize finalizes node data
	Finalize() (header *MerkleHeader, err error)
}

type accumulatorData struct {
	Len   int64
	Roots []*node
}

func (ad *accumulatorData) Clone() *accumulatorData {
	var roots []*node
	if ad.Roots != nil {
		roots = make([]*node, len(ad.Roots))
		for i := range ad.Roots {
			roots[i] = ad.Roots[i].Clone()
		}
	}
	return &accumulatorData{
		Len:   ad.Len,
		Roots: roots,
	}
}

func (ad *accumulatorData) finalize() *MerkleHeader {
	if ad.Len == 0 {
		return &MerkleHeader{}
	}
	var prevHash []byte
	for _, r := range ad.Roots {
		if prevHash != nil {
			r.Add(prevHash)
		}
		prevHash = r.Hash()
	}
	root := ad.Roots[len(ad.Roots)-1]
	if root.Len() != 1 {
		root = newNode()
		root.Add(prevHash)
		ad.Roots = append(ad.Roots, root)
	}
	return &MerkleHeader{
		root.Get(0),
		ad.Len,
	}
}

type accumulator struct {
	data               accumulatorData
	treeBucket         db.Bucket
	accumulatorBucket  *db.CodedBucket
	accumulatorDataKey []byte
}

func (ba *accumulator) add(i int, hash []byte) error {
	if i >= len(ba.data.Roots) {
		ba.data.Roots = append(ba.data.Roots, newNode())
	}
	rb := ba.data.Roots[i]
	rb.Add(hash)
	if rb.Full() {
		if err := ba.treeBucket.Set(rb.Hash(), rb.Bytes()); err != nil {
			return err
		}
		hash := rb.Hash()
		rb.Clear()
		if err := ba.add(i+1, hash); err != nil {
			return err
		}
	}
	return nil
}

func (ba *accumulator) Add(hash []byte) error {
	if err := ba.add(0, hash); err != nil {
		return err
	}
	ba.data.Len++
	return ba.accumulatorBucket.Set(db.Raw(ba.accumulatorDataKey), &ba.data)
}

func (ba *accumulator) Len() int64 {
	return ba.data.Len
}

func (ba *accumulator) GetMerkleHeader() *MerkleHeader {
	data := ba.data.Clone()
	return data.finalize()
}

func (ba *accumulator) Finalize() (header *MerkleHeader, err error) {
	if ba.data.Len == 0 {
		return &MerkleHeader{}, nil
	}
	var prevHash []byte
	for _, r := range ba.data.Roots {
		if prevHash != nil {
			r.Add(prevHash)
		}
		if err = ba.treeBucket.Set(r.Hash(), r.Bytes()); err != nil {
			return nil, err
		}
		prevHash = r.Hash()
	}
	root := ba.data.Roots[len(ba.data.Roots)-1]
	if root.Len() != 1 {
		root = newNode()
		root.Add(prevHash)
		if err = ba.treeBucket.Set(root.Hash(), root.Bytes()); err != nil {
			return nil, err
		}
		ba.data.Roots = append(ba.data.Roots, root)
	}
	return &MerkleHeader{
		RootHash: root.Get(0),
		Leaves: ba.data.Len,
	}, nil
}

// NewAccumulator creates a new accumulator. Merkle node is written in tree
// bucket, accumulator is written on accumulator data key in accumulator bucket.
func NewAccumulator(
	treeBucket db.Bucket,
	accumulatorBucket db.Bucket,
	accumulatorDataKey string,
) (Accumulator, error) {
	if len(accumulatorDataKey) == 0 {
		accumulatorDataKey = defaultAccumulatorKey
	}
	ba := &accumulator{
		treeBucket:         treeBucket,
		accumulatorBucket:  db.NewCodedBucketFromBucket(accumulatorBucket, nil),
		accumulatorDataKey: []byte(accumulatorDataKey),
	}
	err := ba.accumulatorBucket.Get(db.Raw(accumulatorDataKey), &ba.data)
	if err != nil && !errors.NotFoundError.Equals(err) {
		return nil, err
	}
	return ba, nil
}

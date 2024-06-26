package common

import (
	ssz "github.com/ferranbt/fastssz"
)

// MarshalSSZ ssz marshals the UltrasoundStreamBid object
func (u *TopBidWebsocketStreamBid) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(u)
}

// MarshalSSZTo ssz marshals the UltrasoundStreamBid object to a target array
func (u *TopBidWebsocketStreamBid) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Timestamp'
	dst = ssz.MarshalUint64(dst, u.Timestamp)

	// Field (1) 'Slot'
	dst = ssz.MarshalUint64(dst, u.Slot)

	// Field (2) 'BlockNumber'
	dst = ssz.MarshalUint64(dst, u.BlockNumber)

	// Field (3) 'BlockHash'
	dst = append(dst, u.BlockHash[:]...)

	// Field (4) 'ParentHash'
	dst = append(dst, u.ParentHash[:]...)

	// Field (5) 'BuilderPubkey'
	dst = append(dst, u.BuilderPubkey[:]...)

	// Field (6) 'FeeRecipient'
	dst = append(dst, u.FeeRecipient[:]...)

	// Field (7) 'Value'
	dst = append(dst, u.Value[:]...)

	return
}

// UnmarshalSSZ ssz unmarshals the UltrasoundStreamBid object
func (u *TopBidWebsocketStreamBid) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 188 {
		return ssz.ErrSize
	}

	// Field (0) 'Timestamp'
	u.Timestamp = ssz.UnmarshallUint64(buf[0:8])

	// Field (1) 'Slot'
	u.Slot = ssz.UnmarshallUint64(buf[8:16])

	// Field (2) 'BlockNumber'
	u.BlockNumber = ssz.UnmarshallUint64(buf[16:24])

	// Field (3) 'BlockHash'
	copy(u.BlockHash[:], buf[24:56])

	// Field (4) 'ParentHash'
	copy(u.ParentHash[:], buf[56:88])

	// Field (5) 'BuilderPubkey'
	copy(u.BuilderPubkey[:], buf[88:136])

	// Field (6) 'FeeRecipient'
	copy(u.FeeRecipient[:], buf[136:156])

	// Field (7) 'Value'
	copy(u.Value[:], buf[156:188])

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the UltrasoundStreamBid object
func (u *TopBidWebsocketStreamBid) SizeSSZ() (size int) {
	size = 188
	return
}

// HashTreeRoot ssz hashes the UltrasoundStreamBid object
func (u *TopBidWebsocketStreamBid) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(u)
}

// HashTreeRootWith ssz hashes the UltrasoundStreamBid object with a hasher
func (u *TopBidWebsocketStreamBid) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Timestamp'
	hh.PutUint64(u.Timestamp)

	// Field (1) 'Slot'
	hh.PutUint64(u.Slot)

	// Field (2) 'BlockNumber'
	hh.PutUint64(u.BlockNumber)

	// Field (3) 'BlockHash'
	hh.PutBytes(u.BlockHash[:])

	// Field (4) 'ParentHash'
	hh.PutBytes(u.ParentHash[:])

	// Field (5) 'BuilderPubkey'
	hh.PutBytes(u.BuilderPubkey[:])

	// Field (6) 'FeeRecipient'
	hh.PutBytes(u.FeeRecipient[:])

	// Field (7) 'Value'
	hh.PutBytes(u.Value[:])

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the UltrasoundStreamBid object
func (u *TopBidWebsocketStreamBid) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(u)
}

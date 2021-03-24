package btc

import (
	"encoding/hex"
	"fmt"
)

// TODO: Refactor `remote` and `local` packages. Make just `btc/client.go`
//  and `btc/client_local.go` with `Connect` and `ConnectLocal` functions
//  respectively.

// Handle represents a handle to the Bitcoin chain.
type Handle interface {
	// GetHeaderByHeight returns the block header for the given block height.
	GetHeaderByHeight(height int64) (*Header, error)

	// GetHeaderByDigest returns the block header for given digest (hash).
	GetHeaderByDigest(digest Digest) (*Header, error)

	// GetBlockCount returns the number of blocks in the longest blockchain
	GetBlockCount() (int64, error)
}

// Digests represents a 32-byte little-endian Bitcoin digest.
type Digest [32]byte

func (d Digest) String() string {
	return hex.EncodeToString(d[:])
}

// Header represents a Bitcoin block header.
type Header struct {
	// Hash is the hash of the block.
	Hash Digest
	// Height is the height of the block in the Bitcoin blockchain.
	Height int64
	// PrevHash is the hash of the previous block.
	PrevHash Digest
	// MerkleRoot is the hash of the root of the merkle tree of transactions in
	// the block.
	MerkleRoot Digest
	// Raw is the serialized data of the block header (80-byte little-endian).
	Raw []byte
}

func (header *Header) Equals(other *Header) bool {
	if other == nil {
		return false
	}
	return header.Hash == other.Hash &&
		header.Height == other.Height &&
		header.PrevHash == other.PrevHash &&
		header.MerkleRoot == other.MerkleRoot
}

func (header *Header) String() string {
	return fmt.Sprintf(
		"Hash: %s, Height: %d, PrevHash: %s, MerkleRoot: %s, Raw: %s",
		header.Hash,
		header.Height,
		header.PrevHash,
		header.MerkleRoot,
		hex.EncodeToString(header.Raw),
	)
}

// Config is a struct that contains the configuration needed to connect to a
// Bitcoin node.
type Config struct {
	URL      string
	Password string
	Username string
}

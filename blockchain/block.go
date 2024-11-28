package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Version    uint32
	Timestamp  int64
	PrevHash   [32]byte
	MerkleRoot [32]byte
	Difficulty *big.Int
	Nonce      uint32
	Hash       [32]byte
	Transactions []Transaction
}

// NewBlock creates a new block with the given parameters
func NewBlock(version uint32, prevHash [32]byte, difficulty *big.Int) *Block {
	return &Block{
		Version:    version,
		Timestamp:  time.Now().Unix(),
		PrevHash:   prevHash,
		Difficulty: difficulty,
		Nonce:      0,
	}
}

// CalculateHash calculates the SHA-256 hash of the block header
func (b *Block) CalculateHash() [32]byte {
	header := bytes.NewBuffer(nil)
	
	// Write block header fields
	binary.Write(header, binary.LittleEndian, b.Version)
	binary.Write(header, binary.LittleEndian, b.Timestamp)
	header.Write(b.PrevHash[:])
	header.Write(b.MerkleRoot[:])
	binary.Write(header, binary.LittleEndian, b.Difficulty.Bytes())
	binary.Write(header, binary.LittleEndian, b.Nonce)
	
	return sha256.Sum256(header.Bytes())
}

// Mine performs proof-of-work mining on the block
func (b *Block) Mine() {
	target := new(big.Int).Div(new(big.Int).Lsh(big.NewInt(1), 256), b.Difficulty)
	
	for {
		hash := b.CalculateHash()
		hashInt := new(big.Int).SetBytes(hash[:])
		
		if hashInt.Cmp(target) == -1 {
			b.Hash = hash
			return
		}
		b.Nonce++
	}
}

// ValidatePoW validates the proof-of-work for this block
func (b *Block) ValidatePoW() bool {
	target := new(big.Int).Div(new(big.Int).Lsh(big.NewInt(1), 256), b.Difficulty)
	hashInt := new(big.Int).SetBytes(b.Hash[:])
	return hashInt.Cmp(target) == -1
}

// CalculateMerkleRoot calculates the Merkle root of the block's transactions
func (b *Block) CalculateMerkleRoot() [32]byte {
	if len(b.Transactions) == 0 {
		return [32]byte{}
	}

	var hashes [][]byte
	for _, tx := range b.Transactions {
		hashes = append(hashes, tx.Hash[:])
	}

	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		var nextLevel [][]byte
		for i := 0; i < len(hashes); i += 2 {
			hash := sha256.Sum256(append(hashes[i], hashes[i+1]...))
			nextLevel = append(nextLevel, hash[:])
		}
		hashes = nextLevel
	}

	var root [32]byte
	copy(root[:], hashes[0])
	return root
}

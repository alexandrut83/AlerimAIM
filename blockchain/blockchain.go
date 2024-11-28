package blockchain

import (
	"encoding/binary"
	"errors"
	"math/big"
	"sync"
	"time"
)

// Initial difficulty (can be adjusted based on network hash power)
var InitialDifficulty = new(big.Int).Exp(big.NewInt(2), big.NewInt(240), nil) // Target: 2^240

// Blockchain manages the chain of blocks
type Blockchain struct {
	blocks     []*Block
	mempool    []*Transaction
	difficulty *big.Int
	mu         sync.RWMutex
}

// NewBlockchain creates a new blockchain with genesis block
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		difficulty: InitialDifficulty,
		mempool:    make([]*Transaction, 0),
	}
	
	// Create genesis block
	genesis := NewBlock(1, [32]byte{}, bc.difficulty)
	genesis.Timestamp = 1640995200 // 2022-01-01 00:00:00 UTC
	genesis.Mine()
	
	bc.blocks = append(bc.blocks, genesis)
	return bc
}

// AddBlock mines and adds a new block to the chain
func (bc *Blockchain) AddBlock(transactions []*Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if len(bc.blocks) == 0 {
		return errors.New("blockchain not initialized")
	}
	
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(1, prevBlock.Hash, bc.difficulty)
	
	// Add coinbase transaction first
	coinbase := CreateCoinbase(CalculateBlockReward(len(bc.blocks)), []byte{})
	newBlock.Transactions = append(newBlock.Transactions, coinbase)
	
	// Add other transactions
	newBlock.Transactions = append(newBlock.Transactions, transactions...)
	
	// Calculate merkle root
	newBlock.MerkleRoot = newBlock.CalculateMerkleRoot()
	
	// Mine the block
	newBlock.Mine()
	
	// Validate the block
	if !newBlock.ValidatePoW() {
		return errors.New("invalid proof of work")
	}
	
	bc.blocks = append(bc.blocks, newBlock)
	
	// Remove added transactions from mempool
	bc.removeFromMempool(transactions)
	
	return nil
}

// AddTransaction adds a transaction to the mempool
func (bc *Blockchain) AddTransaction(tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}
	
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	// Verify transaction
	if !tx.IsCoinbase() {
		// Add verification logic here
		// - Check if inputs exist and are unspent
		// - Verify signatures
		// - Check if total input value >= total output value
	}
	
	bc.mempool = append(bc.mempool, tx)
	return nil
}

// GetBalance returns the balance for a given address
func (bc *Blockchain) GetBalance(address []byte) uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	var balance uint64
	spentOutputs := make(map[string]bool)
	
	// Iterate through all blocks
	for _, block := range bc.blocks {
		for _, tx := range block.Transactions {
			// Check outputs
			for i, out := range tx.Outputs {
				if bytes.Equal(out.Script, address) {
					key := fmt.Sprintf("%x:%d", tx.Hash, i)
					if !spentOutputs[key] {
						balance += out.Value
					}
				}
			}
			
			// Mark spent outputs
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if bytes.Equal(in.Script, address) {
						key := fmt.Sprintf("%x:%d", in.PrevTxHash, in.PrevTxIndex)
						spentOutputs[key] = true
					}
				}
			}
		}
	}
	
	return balance
}

// CalculateBlockReward calculates the mining reward for a given block height
func CalculateBlockReward(height int) uint64 {
	// Initial reward is 0.01 AIM
	initialReward := uint64(1000000) // 0.01 AIM in smallest unit
	
	// Halving every 210,000 blocks (approximately 4 years with 1-minute blocks)
	halvings := height / 210000
	
	if halvings >= 64 {
		return 0
	}
	
	// Right shift to implement halving
	return initialReward >> uint(halvings)
}

// removeFromMempool removes the given transactions from the mempool
func (bc *Blockchain) removeFromMempool(transactions []*Transaction) {
	txMap := make(map[[32]byte]bool)
	for _, tx := range transactions {
		txMap[tx.Hash] = true
	}
	
	newMempool := make([]*Transaction, 0)
	for _, tx := range bc.mempool {
		if !txMap[tx.Hash] {
			newMempool = append(newMempool, tx)
		}
	}
	
	bc.mempool = newMempool
}

// GetLatestBlock returns the most recent block in the chain
func (bc *Blockchain) GetLatestBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	if len(bc.blocks) == 0 {
		return nil
	}
	return bc.blocks[len(bc.blocks)-1]
}

// ValidateChain validates the entire blockchain
func (bc *Blockchain) ValidateChain() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	for i := 1; i < len(bc.blocks); i++ {
		currentBlock := bc.blocks[i]
		previousBlock := bc.blocks[i-1]
		
		// Check hash connection
		if !bytes.Equal(currentBlock.PrevHash[:], previousBlock.Hash[:]) {
			return false
		}
		
		// Validate proof of work
		if !currentBlock.ValidatePoW() {
			return false
		}
		
		// Validate merkle root
		if currentBlock.MerkleRoot != currentBlock.CalculateMerkleRoot() {
			return false
		}
	}
	
	return true
}

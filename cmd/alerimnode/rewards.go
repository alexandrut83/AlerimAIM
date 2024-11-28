package main

import (
	"math/big"
	"sync"
	"time"
)

// RewardConfig defines the pool's reward distribution configuration
type RewardConfig struct {
	BlockReward       *big.Int // Base reward per block
	PoolFee          float64   // Pool fee percentage (0-100)
	PayoutThreshold  *big.Int // Minimum amount for payout
	MaturityDepth    uint64   // Number of confirmations before rewards are paid
	PayoutInterval   time.Duration
}

// RewardManager handles reward calculations and distributions
type RewardManager struct {
	mu            sync.RWMutex
	config        *RewardConfig
	pendingShares map[string]int64    // minerID -> shares
	balances      map[string]*big.Int // minerID -> balance
	blockchain    *blockchain.Blockchain
}

// NewRewardManager creates a new reward manager instance
func NewRewardManager(bc *blockchain.Blockchain) *RewardManager {
	return &RewardManager{
		config: &RewardConfig{
			BlockReward:      new(big.Int).Mul(big.NewInt(50), big.NewInt(1e18)), // 50 AIM
			PoolFee:         2.0, // 2%
			PayoutThreshold: new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)),  // 1 AIM
			MaturityDepth:   100,
			PayoutInterval:  24 * time.Hour,
		},
		pendingShares: make(map[string]int64),
		balances:      make(map[string]*big.Int),
		blockchain:    bc,
	}
}

// AddShare records a share for reward calculation
func (rm *RewardManager) AddShare(minerID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.pendingShares[minerID]++
}

// ProcessBlockReward distributes rewards when a block is found
func (rm *RewardManager) ProcessBlockReward(block *blockchain.Block) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Calculate total shares
	var totalShares int64
	for _, shares := range rm.pendingShares {
		totalShares += shares
	}

	if totalShares == 0 {
		return
	}

	// Calculate pool fee
	poolFeeAmount := new(big.Int).Mul(rm.config.BlockReward, big.NewInt(int64(rm.config.PoolFee)))
	poolFeeAmount.Div(poolFeeAmount, big.NewInt(100))

	// Calculate reward per share
	remainingReward := new(big.Int).Sub(rm.config.BlockReward, poolFeeAmount)
	rewardPerShare := new(big.Float).Quo(
		new(big.Float).SetInt(remainingReward),
		new(big.Float).SetInt64(totalShares),
	)

	// Distribute rewards to miners
	for minerID, shares := range rm.pendingShares {
		minerReward := new(big.Float).Mul(rewardPerShare, new(big.Float).SetInt64(shares))
		rewardInt, _ := minerReward.Int(nil)

		if _, exists := rm.balances[minerID]; !exists {
			rm.balances[minerID] = new(big.Int)
		}
		rm.balances[minerID].Add(rm.balances[minerID], rewardInt)
	}

	// Clear pending shares for next round
	rm.pendingShares = make(map[string]int64)
}

// GetMinerBalance returns a miner's current balance
func (rm *RewardManager) GetMinerBalance(minerID string) *big.Int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if balance, exists := rm.balances[minerID]; exists {
		return new(big.Int).Set(balance)
	}
	return new(big.Int)
}

// ProcessPayouts processes pending payouts for all miners
func (rm *RewardManager) ProcessPayouts() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for minerID, balance := range rm.balances {
		if balance.Cmp(rm.config.PayoutThreshold) >= 0 {
			// Create payout transaction
			tx := &blockchain.Transaction{
				From:      "pool",
				To:        minerID,
				Amount:    new(big.Int).Set(balance),
				Timestamp: time.Now(),
			}

			if err := rm.blockchain.AddTransaction(tx); err != nil {
				return err
			}

			// Reset balance after successful payout
			rm.balances[minerID] = new(big.Int)
		}
	}

	return nil
}

// StartPayoutProcessor starts the automatic payout processor
func (rm *RewardManager) StartPayoutProcessor() {
	go func() {
		ticker := time.NewTicker(rm.config.PayoutInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := rm.ProcessPayouts(); err != nil {
				log.Printf("Error processing payouts: %v", err)
			}
		}
	}()
}

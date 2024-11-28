package main

import (
	"sync"
	"time"

	"github.com/alexandrut83/alerimAIM/blockchain"
)

// MiningPool manages mining workers and distributes work
type MiningPool struct {
	mu            sync.RWMutex
	miners        map[string]*Miner
	currentBlock  *blockchain.Block
	blockchain    *blockchain.Blockchain
	difficulty    *big.Int
	totalHashrate float64
	rewards       *RewardManager
	stratum       *StratumServer
	workerDiffs   map[string]*big.Int // Worker-specific difficulties
	vardiff       *VarDiffManager     // Add vardiff manager
}

// NewMiningPool creates a new mining pool instance
func NewMiningPool(bc *blockchain.Blockchain) *MiningPool {
	pool := &MiningPool{
		miners:      make(map[string]*Miner),
		blockchain:  bc,
		difficulty:  new(big.Int).Set(blockchain.InitialDifficulty),
		workerDiffs: make(map[string]*big.Int),
	}

	// Initialize reward manager
	pool.rewards = NewRewardManager(bc)

	// Initialize stratum server on port 3333
	stratum, err := NewStratumServer(pool, pool.rewards, 3333)
	if err != nil {
		log.Printf("Failed to initialize stratum server: %v", err)
	} else {
		pool.stratum = stratum
	}

	// Initialize vardiff manager
	pool.vardiff = NewVarDiffManager(pool)

	return pool
}

// AddMiner registers a new miner in the pool
func (p *MiningPool) AddMiner(miner *Miner) {
	p.mu.Lock()
	defer p.mu.Unlock()

	miner.LastSeen = time.Now()
	p.miners[miner.ID] = miner
}

// RemoveMiner removes a miner from the pool
func (p *MiningPool) RemoveMiner(minerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.miners, minerID)
}

// UpdateMinerStats updates a miner's statistics
func (p *MiningPool) UpdateMinerStats(minerID string, hashrate float64, shares int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if miner, exists := p.miners[minerID]; exists {
		miner.Hashrate = hashrate
		miner.TotalShares += shares
		miner.LastSeen = time.Now()
	}
}

// GetTotalHashrate calculates the total hashrate of all active miners
func (p *MiningPool) GetTotalHashrate() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var total float64
	now := time.Now()
	timeout := 5 * time.Minute

	for _, miner := range p.miners {
		if now.Sub(miner.LastSeen) < timeout {
			total += miner.Hashrate
		}
	}

	return total
}

// GetActiveMiners returns a list of currently active miners
func (p *MiningPool) GetActiveMiners() []*Miner {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var active []*Miner
	now := time.Now()
	timeout := 5 * time.Minute

	for _, miner := range p.miners {
		if now.Sub(miner.LastSeen) < timeout {
			active = append(active, miner)
		}
	}

	return active
}

// UpdateDifficulty adjusts the mining difficulty based on network hashrate
func (p *MiningPool) UpdateDifficulty() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Target block time in seconds (2 minutes)
	const targetBlockTime = 120
	// Adjustment window in blocks
	const difficultyAdjustmentWindow = 2016 // About 2 weeks worth of blocks
	// Maximum difficulty adjustment factor
	const maxAdjustmentFactor = 4

	// Get the current block height
	height := p.blockchain.GetHeight()
	if height < difficultyAdjustmentWindow {
		return // Not enough blocks for adjustment
	}

	// Get timestamps of the first and last block in the window
	startBlock := p.blockchain.GetBlockByHeight(height - difficultyAdjustmentWindow)
	endBlock := p.blockchain.GetLatestBlock()
	if startBlock == nil || endBlock == nil {
		return
	}

	// Calculate the actual time taken for the window
	actualTimespan := endBlock.Timestamp.Sub(startBlock.Timestamp).Seconds()
	targetTimespan := float64(targetBlockTime * difficultyAdjustmentWindow)

	// Calculate adjustment factor
	adjustment := targetTimespan / actualTimespan
	if adjustment > maxAdjustmentFactor {
		adjustment = maxAdjustmentFactor
	} else if adjustment < 1/maxAdjustmentFactor {
		adjustment = 1 / maxAdjustmentFactor
	}

	// Apply the adjustment to current difficulty
	newDifficulty := new(big.Int).Set(p.difficulty)
	adjustmentBig := new(big.Float).SetFloat64(adjustment)
	difficultyFloat := new(big.Float).SetInt(newDifficulty)
	
	// Multiply current difficulty by adjustment factor
	difficultyFloat.Mul(difficultyFloat, adjustmentBig)
	
	// Convert back to big.Int
	newDifficulty, _ = difficultyFloat.Int(nil)

	// Ensure difficulty doesn't go below initial difficulty
	if newDifficulty.Cmp(blockchain.InitialDifficulty) < 0 {
		newDifficulty.Set(blockchain.InitialDifficulty)
	}

	p.difficulty.Set(newDifficulty)
}

// UpdateWorkerDifficulty adjusts a worker's difficulty based on share rate
func (p *MiningPool) UpdateWorkerDifficulty(minerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	miner, exists := p.miners[minerID]
	if !exists {
		return
	}

	// Target share time in seconds
	const targetShareTime = 10
	// Maximum difficulty adjustment factor
	const maxAdjustment = 2.0

	// Calculate time since last share
	timeSinceLastShare := time.Since(miner.LastSeen).Seconds()
	if timeSinceLastShare == 0 {
		return
	}

	// Calculate adjustment factor
	adjustment := targetShareTime / timeSinceLastShare
	if adjustment > maxAdjustment {
		adjustment = maxAdjustment
	} else if adjustment < 1/maxAdjustment {
		adjustment = 1 / maxAdjustment
	}

	// Get current worker difficulty
	currentDiff := p.workerDiffs[minerID]
	if currentDiff == nil {
		currentDiff = new(big.Int).Set(p.difficulty)
		p.workerDiffs[minerID] = currentDiff
	}

	// Apply adjustment
	adjustmentBig := new(big.Float).SetFloat64(adjustment)
	difficultyFloat := new(big.Float).SetInt(currentDiff)
	difficultyFloat.Mul(difficultyFloat, adjustmentBig)
	
	newDiff, _ := difficultyFloat.Int(nil)
	p.workerDiffs[minerID] = newDiff

	// Notify stratum client of difficulty change
	if p.stratum != nil {
		if client, exists := p.stratum.clients[minerID]; exists {
			client.difficulty = newDiff
			// Send difficulty change notification
			client.sendResponse(StratumResponse{
				Method: "mining.set_difficulty",
				Params: []interface{}{fmt.Sprintf("%x", newDiff)},
			})
		}
	}
}

// SubmitShare processes a share submission from a miner
func (p *MiningPool) SubmitShare(minerID string, nonce uint64, hash []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Record share for vardiff adjustment
	p.vardiff.RecordShare(minerID)

	// Get miner's specific difficulty
	minerDiff := p.workerDiffs[minerID]
	if minerDiff == nil {
		minerDiff = p.difficulty
	}

	// Verify the share meets the worker's difficulty
	if !blockchain.MeetsDifficulty(hash, minerDiff) {
		return fmt.Errorf("share difficulty too low")
	}

	// Update miner statistics
	miner, exists := p.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found: %s", minerID)
	}

	miner.TotalShares++
	miner.LastSeen = time.Now()

	// Add share for reward calculation
	p.rewards.AddShare(minerID)

	// If share meets network difficulty, submit to blockchain
	networkDifficulty := p.blockchain.GetCurrentDifficulty()
	if blockchain.MeetsDifficulty(hash, networkDifficulty) {
		block := p.currentBlock.Clone()
		block.Nonce = nonce
		block.Hash = hash

		if err := p.blockchain.AddBlock(block); err != nil {
			return fmt.Errorf("failed to add block: %v", err)
		}

		// Process block reward
		p.rewards.ProcessBlockReward(block)

		// Create new block template for mining
		p.createNewBlockTemplate()

		// Notify all stratum clients of new work
		if p.stratum != nil {
			p.stratum.mu.RLock()
			for _, client := range p.stratum.clients {
				client.sendWork()
			}
			p.stratum.mu.RUnlock()
		}
	}

	// Update worker difficulty based on share time
	go p.UpdateWorkerDifficulty(minerID)

	return nil
}

// createNewBlockTemplate creates a new block for miners to work on
func (p *MiningPool) createNewBlockTemplate() {
	transactions := p.blockchain.GetPendingTransactions()
	previousBlock := p.blockchain.GetLatestBlock()

	p.currentBlock = &blockchain.Block{
		Version:        1,
		PreviousHash:  previousBlock.Hash,
		Timestamp:     time.Now(),
		Transactions:  transactions,
		MerkleRoot:    blockchain.CalculateMerkleRoot(transactions),
		Difficulty:    p.difficulty,
		Nonce:        0,
	}
}

// StartMining begins the mining process
func (p *MiningPool) StartMining() {
	// Create initial block template
	p.createNewBlockTemplate()

	// Start difficulty adjustment routine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			p.UpdateDifficulty()
		}
	}()

	// Start mining coordination routine
	go func() {
		for {
			// Update mining statistics
			p.mu.Lock()
			p.totalHashrate = p.GetTotalHashrate()
			activeMiners := len(p.GetActiveMiners())
			p.mu.Unlock()

			// Update global stats for admin panel
			stats.mu.Lock()
			stats.TotalHashrate = p.totalHashrate
			stats.ActiveMiners = activeMiners
			stats.Difficulty.Set(p.difficulty)
			stats.mu.Unlock()

			// Sleep briefly before next update
			time.Sleep(time.Second)
		}
	}()
}

// StopMining stops the mining process
func (p *MiningPool) StopMining() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear all miners
	p.miners = make(map[string]*Miner)
	p.totalHashrate = 0
	
	// Reset mining stats
	stats.mu.Lock()
	stats.TotalHashrate = 0
	stats.ActiveMiners = 0
	stats.mu.Unlock()
}

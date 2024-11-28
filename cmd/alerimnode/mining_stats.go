package main

import (
	"math/big"
	"sync"
	"time"
)

// TimeWindow represents a time window for statistics
type TimeWindow struct {
	Duration time.Duration
	Shares   int64
	Blocks   int64
	Hashrate float64
	StartTime time.Time
}

// MinerStats tracks detailed statistics for a miner
type MinerStats struct {
	mu              sync.RWMutex
	TotalShares     int64
	ValidShares     int64
	InvalidShares   int64
	BlocksFound     int64
	LastShare       time.Time
	LastBlock       time.Time
	CurrentHashrate float64
	AverageHashrate float64
	Windows         map[time.Duration]*TimeWindow // Different time windows (1h, 24h, 7d)
	ShareHistory    []ShareEntry
	Difficulties    []DifficultyEntry
}

// ShareEntry represents a single share submission
type ShareEntry struct {
	Timestamp  time.Time
	Difficulty *big.Int
	Valid      bool
}

// DifficultyEntry tracks difficulty changes
type DifficultyEntry struct {
	Timestamp  time.Time
	Difficulty *big.Int
	Reason     string
}

// PoolStats tracks overall pool statistics
type PoolStats struct {
	mu                sync.RWMutex
	TotalHashrate     float64
	NetworkHashrate   float64
	ActiveWorkers     int
	ConnectedWorkers  int
	BlocksFound       int64
	LastBlockTime     time.Time
	CurrentDifficulty *big.Int
	NetworkDifficulty *big.Int
	SharesPerSecond   float64
	Windows           map[time.Duration]*TimeWindow
	BlockHistory      []BlockEntry
}

// BlockEntry represents a found block
type BlockEntry struct {
	Timestamp time.Time
	Height    uint64
	Hash      []byte
	Miner     string
	Reward    *big.Int
}

// NewMinerStats creates a new miner statistics tracker
func NewMinerStats() *MinerStats {
	return &MinerStats{
		Windows: map[time.Duration]*TimeWindow{
			time.Hour:          {Duration: time.Hour, StartTime: time.Now()},
			24 * time.Hour:     {Duration: 24 * time.Hour, StartTime: time.Now()},
			7 * 24 * time.Hour: {Duration: 7 * 24 * time.Hour, StartTime: time.Now()},
		},
		ShareHistory: make([]ShareEntry, 0, 1000),    // Keep last 1000 shares
		Difficulties: make([]DifficultyEntry, 0, 100), // Keep last 100 difficulty changes
	}
}

// AddShare records a share submission
func (ms *MinerStats) AddShare(difficulty *big.Int, valid bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	ms.TotalShares++
	if valid {
		ms.ValidShares++
	} else {
		ms.InvalidShares++
	}

	// Add to share history
	ms.ShareHistory = append(ms.ShareHistory, ShareEntry{
		Timestamp:  now,
		Difficulty: new(big.Int).Set(difficulty),
		Valid:      valid,
	})

	// Maintain history size
	if len(ms.ShareHistory) > 1000 {
		ms.ShareHistory = ms.ShareHistory[1:]
	}

	// Update time windows
	for _, window := range ms.Windows {
		if now.Sub(window.StartTime) > window.Duration {
			// Reset window if it's expired
			window.StartTime = now
			window.Shares = 0
			window.Blocks = 0
			window.Hashrate = 0
		}
		window.Shares++
	}

	// Update hashrate calculations
	ms.updateHashrate()
}

// AddBlock records a found block
func (ms *MinerStats) AddBlock() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	ms.BlocksFound++
	ms.LastBlock = now

	// Update time windows
	for _, window := range ms.Windows {
		if now.Sub(window.StartTime) <= window.Duration {
			window.Blocks++
		}
	}
}

// RecordDifficultyChange records a difficulty adjustment
func (ms *MinerStats) RecordDifficultyChange(difficulty *big.Int, reason string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry := DifficultyEntry{
		Timestamp:  time.Now(),
		Difficulty: new(big.Int).Set(difficulty),
		Reason:     reason,
	}

	ms.Difficulties = append(ms.Difficulties, entry)
	if len(ms.Difficulties) > 100 {
		ms.Difficulties = ms.Difficulties[1:]
	}
}

// updateHashrate calculates current and average hashrates
func (ms *MinerStats) updateHashrate() {
	// Calculate hashrate based on recent shares
	if len(ms.ShareHistory) < 2 {
		return
	}

	// Use last 10 minutes of shares for current hashrate
	cutoff := time.Now().Add(-10 * time.Minute)
	var recentShares int64
	var oldestTime time.Time

	for i := len(ms.ShareHistory) - 1; i >= 0; i-- {
		share := ms.ShareHistory[i]
		if share.Timestamp.Before(cutoff) {
			break
		}
		if oldestTime.IsZero() {
			oldestTime = share.Timestamp
		}
		recentShares++
	}

	if recentShares > 0 {
		timespan := time.Since(oldestTime).Seconds()
		if timespan > 0 {
			ms.CurrentHashrate = float64(recentShares) / timespan
		}
	}

	// Calculate average hashrate over 24 hours
	dayWindow := ms.Windows[24*time.Hour]
	if dayWindow != nil {
		timespan := time.Since(dayWindow.StartTime).Seconds()
		if timespan > 0 {
			ms.AverageHashrate = float64(dayWindow.Shares) / timespan
		}
	}
}

// GetStats returns current statistics
func (ms *MinerStats) GetStats() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	stats := map[string]interface{}{
		"total_shares":      ms.TotalShares,
		"valid_shares":      ms.ValidShares,
		"invalid_shares":    ms.InvalidShares,
		"blocks_found":      ms.BlocksFound,
		"current_hashrate":  ms.CurrentHashrate,
		"average_hashrate": ms.AverageHashrate,
		"last_share":       ms.LastShare,
		"last_block":       ms.LastBlock,
	}

	// Add window statistics
	windows := make(map[string]interface{})
	for duration, window := range ms.Windows {
		windows[duration.String()] = map[string]interface{}{
			"shares":   window.Shares,
			"blocks":   window.Blocks,
			"hashrate": window.Hashrate,
		}
	}
	stats["windows"] = windows

	return stats
}

// NewPoolStats creates a new pool statistics tracker
func NewPoolStats() *PoolStats {
	return &PoolStats{
		CurrentDifficulty: new(big.Int),
		NetworkDifficulty: new(big.Int),
		Windows: map[time.Duration]*TimeWindow{
			time.Hour:          {Duration: time.Hour, StartTime: time.Now()},
			24 * time.Hour:     {Duration: 24 * time.Hour, StartTime: time.Now()},
			7 * 24 * time.Hour: {Duration: 7 * 24 * time.Hour, StartTime: time.Now()},
		},
		BlockHistory: make([]BlockEntry, 0, 1000), // Keep last 1000 blocks
	}
}

// AddBlock records a found block
func (ps *PoolStats) AddBlock(height uint64, hash []byte, miner string, reward *big.Int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	ps.BlocksFound++
	ps.LastBlockTime = now

	// Add to block history
	ps.BlockHistory = append(ps.BlockHistory, BlockEntry{
		Timestamp: now,
		Height:    height,
		Hash:      hash,
		Miner:     miner,
		Reward:    new(big.Int).Set(reward),
	})

	// Maintain history size
	if len(ps.BlockHistory) > 1000 {
		ps.BlockHistory = ps.BlockHistory[1:]
	}

	// Update time windows
	for _, window := range ps.Windows {
		if now.Sub(window.StartTime) > window.Duration {
			window.StartTime = now
			window.Blocks = 0
		}
		window.Blocks++
	}
}

// UpdateHashrate updates pool hashrate statistics
func (ps *PoolStats) UpdateHashrate(poolHashrate, networkHashrate float64, activeWorkers, connectedWorkers int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.TotalHashrate = poolHashrate
	ps.NetworkHashrate = networkHashrate
	ps.ActiveWorkers = activeWorkers
	ps.ConnectedWorkers = connectedWorkers

	// Update time windows
	now := time.Now()
	for _, window := range ps.Windows {
		if now.Sub(window.StartTime) > window.Duration {
			window.StartTime = now
			window.Hashrate = 0
		}
		window.Hashrate = poolHashrate
	}
}

// GetStats returns current pool statistics
func (ps *PoolStats) GetStats() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	stats := map[string]interface{}{
		"total_hashrate":      ps.TotalHashrate,
		"network_hashrate":    ps.NetworkHashrate,
		"active_workers":      ps.ActiveWorkers,
		"connected_workers":   ps.ConnectedWorkers,
		"blocks_found":        ps.BlocksFound,
		"last_block_time":     ps.LastBlockTime,
		"current_difficulty":  ps.CurrentDifficulty,
		"network_difficulty": ps.NetworkDifficulty,
		"shares_per_second":  ps.SharesPerSecond,
	}

	// Add window statistics
	windows := make(map[string]interface{})
	for duration, window := range ps.Windows {
		windows[duration.String()] = map[string]interface{}{
			"blocks":   window.Blocks,
			"hashrate": window.Hashrate,
		}
	}
	stats["windows"] = windows

	// Add recent blocks
	recentBlocks := make([]map[string]interface{}, 0, 10)
	for i := len(ps.BlockHistory) - 1; i >= max(0, len(ps.BlockHistory)-10); i-- {
		block := ps.BlockHistory[i]
		recentBlocks = append(recentBlocks, map[string]interface{}{
			"timestamp": block.Timestamp,
			"height":    block.Height,
			"hash":      block.Hash,
			"miner":     block.Miner,
			"reward":    block.Reward,
		})
	}
	stats["recent_blocks"] = recentBlocks

	return stats
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

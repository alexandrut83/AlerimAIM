package main

import (
	"math/big"
	"sync"
	"time"
)

// VarDiffConfig holds configuration for variable difficulty
type VarDiffConfig struct {
	TargetTime      time.Duration // Target time between shares (e.g., 10 seconds)
	RetargetTime    time.Duration // Time between difficulty adjustments
	VariancePercent float64       // Allowed variance in share time (e.g., 30%)
	MaximumStep     float64       // Maximum difficulty adjustment step (e.g., 200%)
	MinimumStep     float64       // Minimum difficulty adjustment step (e.g., 50%)
	MinimumDiff     *big.Int      // Minimum allowed difficulty
	MaximumDiff     *big.Int      // Maximum allowed difficulty
	BufferSize      int           // Number of shares to keep for variance calculation
}

// VarDiffManager manages variable difficulty for miners
type VarDiffManager struct {
	mu       sync.RWMutex
	config   *VarDiffConfig
	miners   map[string]*MinerVarDiff
	pool     *MiningPool
}

// MinerVarDiff tracks vardiff state for a single miner
type MinerVarDiff struct {
	mu            sync.Mutex
	currentDiff   *big.Int
	shares        []time.Time
	lastRetarget  time.Time
	lastShareTime time.Time
	timeBuffer    []float64 // Buffer of share times for variance calculation
}

// NewVarDiffManager creates a new vardiff manager
func NewVarDiffManager(pool *MiningPool) *VarDiffManager {
	return &VarDiffManager{
		config: &VarDiffConfig{
			TargetTime:      10 * time.Second,
			RetargetTime:    120 * time.Second,
			VariancePercent: 30.0,
			MaximumStep:     200.0,
			MinimumStep:     50.0,
			MinimumDiff:     new(big.Int).Set(blockchain.InitialDifficulty),
			MaximumDiff:     new(big.Int).Mul(blockchain.InitialDifficulty, big.NewInt(1000000)),
			BufferSize:      30,
		},
		miners: make(map[string]*MinerVarDiff),
		pool:   pool,
	}
}

// GetMinerDiff gets or creates miner vardiff state
func (v *VarDiffManager) GetMinerDiff(minerID string) *MinerVarDiff {
	v.mu.Lock()
	defer v.mu.Unlock()

	if miner, exists := v.miners[minerID]; exists {
		return miner
	}

	miner := &MinerVarDiff{
		currentDiff:  new(big.Int).Set(v.config.MinimumDiff),
		shares:       make([]time.Time, 0, v.config.BufferSize),
		lastRetarget: time.Now(),
		timeBuffer:   make([]float64, 0, v.config.BufferSize),
	}
	v.miners[minerID] = miner
	return miner
}

// RecordShare records a share submission and updates difficulty if needed
func (v *VarDiffManager) RecordShare(minerID string) {
	miner := v.GetMinerDiff(minerID)
	miner.mu.Lock()
	defer miner.mu.Unlock()

	now := time.Now()

	// Calculate time since last share
	if !miner.lastShareTime.IsZero() {
		timeDiff := now.Sub(miner.lastShareTime).Seconds()
		miner.timeBuffer = append(miner.timeBuffer, timeDiff)
		if len(miner.timeBuffer) > v.config.BufferSize {
			miner.timeBuffer = miner.timeBuffer[1:]
		}
	}

	miner.lastShareTime = now
	miner.shares = append(miner.shares, now)
	if len(miner.shares) > v.config.BufferSize {
		miner.shares = miner.shares[1:]
	}

	// Check if it's time to adjust difficulty
	if now.Sub(miner.lastRetarget) >= v.config.RetargetTime {
		v.adjustDifficulty(minerID, miner)
	}
}

// adjustDifficulty calculates and sets new difficulty for a miner
func (v *VarDiffManager) adjustDifficulty(minerID string, miner *MinerVarDiff) {
	if len(miner.timeBuffer) < 2 {
		return
	}

	// Calculate average share time
	var totalTime float64
	for _, t := range miner.timeBuffer {
		totalTime += t
	}
	averageTime := totalTime / float64(len(miner.timeBuffer))

	// Calculate variance
	var variance float64
	for _, t := range miner.timeBuffer {
		diff := t - averageTime
		variance += diff * diff
	}
	variance /= float64(len(miner.timeBuffer))
	
	// Skip adjustment if variance is too high
	if variance > (averageTime * v.config.VariancePercent / 100.0) {
		return
	}

	// Calculate ideal adjustment factor
	targetSeconds := v.config.TargetTime.Seconds()
	adjustment := targetSeconds / averageTime

	// Apply adjustment limits
	if adjustment > v.config.MaximumStep/100.0 {
		adjustment = v.config.MaximumStep/100.0
	} else if adjustment < v.config.MinimumStep/100.0 {
		adjustment = v.config.MinimumStep/100.0
	}

	// Calculate new difficulty
	newDiff := new(big.Float).SetInt(miner.currentDiff)
	newDiff.Mul(newDiff, big.NewFloat(adjustment))
	
	finalDiff, _ := newDiff.Int(nil)

	// Apply min/max limits
	if finalDiff.Cmp(v.config.MinimumDiff) < 0 {
		finalDiff.Set(v.config.MinimumDiff)
	} else if finalDiff.Cmp(v.config.MaximumDiff) > 0 {
		finalDiff.Set(v.config.MaximumDiff)
	}

	// Only update if difficulty changed significantly (>1%)
	diffChange := new(big.Float).Quo(new(big.Float).SetInt(finalDiff), new(big.Float).SetInt(miner.currentDiff))
	changeValue, _ := diffChange.Float64()
	if changeValue < 0.99 || changeValue > 1.01 {
		// Record the change
		reason := "VarDiff adjustment"
		if stats, ok := v.pool.miners[minerID]; ok {
			stats.RecordDifficultyChange(finalDiff, reason)
		}

		// Update difficulty
		miner.currentDiff.Set(finalDiff)
		miner.lastRetarget = time.Now()
		miner.timeBuffer = miner.timeBuffer[:0]

		// Notify stratum client
		if v.pool.stratum != nil {
			if client, exists := v.pool.stratum.clients[minerID]; exists {
				client.difficulty = finalDiff
				client.sendResponse(StratumResponse{
					Method: "mining.set_difficulty",
					Params: []interface{}{fmt.Sprintf("%x", finalDiff)},
				})
			}
		}
	}
}

// GetDifficulty returns current difficulty for a miner
func (v *VarDiffManager) GetDifficulty(minerID string) *big.Int {
	miner := v.GetMinerDiff(minerID)
	miner.mu.Lock()
	defer miner.mu.Unlock()
	return new(big.Int).Set(miner.currentDiff)
}

// GetStats returns vardiff statistics for a miner
func (v *VarDiffManager) GetStats(minerID string) map[string]interface{} {
	miner := v.GetMinerDiff(minerID)
	miner.mu.Lock()
	defer miner.mu.Unlock()

	var averageTime float64
	if len(miner.timeBuffer) > 0 {
		var total float64
		for _, t := range miner.timeBuffer {
			total += t
		}
		averageTime = total / float64(len(miner.timeBuffer))
	}

	return map[string]interface{}{
		"current_diff":     miner.currentDiff,
		"average_time":     averageTime,
		"buffer_size":      len(miner.timeBuffer),
		"last_retarget":    miner.lastRetarget,
		"last_share_time":  miner.lastShareTime,
		"target_time":      v.config.TargetTime.Seconds(),
		"variance_percent": v.config.VariancePercent,
	}
}

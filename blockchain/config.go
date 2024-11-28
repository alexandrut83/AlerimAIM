package blockchain

import (
	"math/big"
	"time"
)

const (
	// NetworkName is the name of the cryptocurrency
	NetworkName = "Alerim"
	
	// CoinSymbol is the symbol of the cryptocurrency
	CoinSymbol = "AIM"
	
	// BlockTime is the target time between blocks
	BlockTime = 60 * time.Second
	
	// InitialBlockReward is the reward for mining a block
	InitialBlockReward = 0.01
	
	// MaximumSupply is the maximum number of coins that can exist
	MaximumSupply = 1000000
	
	// Version is the current version of the protocol
	Version = "0.1.0"
)

var (
	// Difficulty is the initial mining difficulty
	InitialDifficulty = big.NewInt(1000000)
	
	// BlocksPerAdjustment is the number of blocks between difficulty adjustments
	BlocksPerAdjustment = 2016
	
	// GenesisBlock is the first block of the blockchain
	GenesisBlock = Block{
		Version:    1,
		Timestamp:  1640995200, // 2022-01-01 00:00:00 UTC
		Difficulty: InitialDifficulty,
		Nonce:      0,
		PrevHash:   [32]byte{},
	}
)

// ConsensusParams contains the parameters for the consensus algorithm
type ConsensusParams struct {
	Algorithm           string
	MergeminingEnabled bool
	MinimumDifficulty  *big.Int
}

var DefaultConsensusParams = ConsensusParams{
	Algorithm:           "sha256",
	MergeminingEnabled: true,
	MinimumDifficulty:  big.NewInt(1000),
}

package main

import (
	"time"
)

// User represents a registered user in the system
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
	Status    string    `json:"status"`
}

// Miner represents a mining worker in the network
type Miner struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	Hashrate    float64   `json:"hashrate"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`
	TotalShares int64     `json:"total_shares"`
}

// Wallet represents a cryptocurrency wallet
type Wallet struct {
	Address     string    `json:"address"`
	PublicKey   string    `json:"public_key"`
	Balance     float64   `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
	Status      string    `json:"status"`
}

// Global state variables
var (
	users        []*User
	activeMiners []*Miner
	wallets      []*Wallet
)

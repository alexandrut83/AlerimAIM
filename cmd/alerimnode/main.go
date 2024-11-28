package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alexandrut83/alerimAIM/blockchain"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
)

var (
	port = flag.Int("port", 8545, "Node port")
	p2pPort = flag.Int("p2p", 9000, "P2P port")
	peers = flag.String("peers", "", "Comma-separated list of peer addresses")
)

// Global state for mining statistics
type MiningStats struct {
	TotalHashrate float64
	ActiveMiners  int
	Difficulty    *big.Int
	mu           sync.RWMutex
}

var stats = &MiningStats{
	Difficulty: new(big.Int),
}

func main() {
	flag.Parse()

	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Initialize blockchain
	bc := blockchain.NewBlockchain()

	// Initialize P2P network
	network, err := blockchain.NewNetwork(bc, *p2pPort)
	if err != nil {
		log.Fatal(err)
	}

	// Connect to initial peers
	if *peers != "" {
		for _, peer := range strings.Split(*peers, ",") {
			if err := network.Connect(peer); err != nil {
				log.Printf("Failed to connect to peer %s: %v", peer, err)
			}
		}
	}

	// Initialize HTTP server
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Static files for admin panel
	router.Static("/admin", "./wallet/web")

	// API endpoints
	api := router.Group("/api")
	{
		// Blockchain endpoints
		api.GET("/status", func(c *gin.Context) {
			latestBlock := bc.GetLatestBlock()
			c.JSON(http.StatusOK, gin.H{
				"height": len(bc.GetBlocks()),
				"latest_block": latestBlock.Hash,
				"peers": len(network.GetPeers()),
			})
		})

		api.POST("/transaction", func(c *gin.Context) {
			var tx blockchain.Transaction
			if err := c.BindJSON(&tx); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := bc.AddTransaction(&tx); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			network.BroadcastTransaction(&tx)
			c.JSON(http.StatusOK, gin.H{"hash": tx.Hash})
		})

		// Admin panel endpoints
		api.GET("/stats", func(c *gin.Context) {
			stats.mu.RLock()
			defer stats.mu.RUnlock()
			
			c.JSON(http.StatusOK, gin.H{
				"hashrate": stats.TotalHashrate,
				"activeMiners": stats.ActiveMiners,
				"difficulty": stats.Difficulty,
				"totalUsers": len(users),
			})
		})

		api.GET("/miners", authMiddleware(), func(c *gin.Context) {
			c.JSON(http.StatusOK, activeMiners)
		})

		api.POST("/miners", authMiddleware(), func(c *gin.Context) {
			var miner Miner
			if err := c.BindJSON(&miner); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			activeMiners = append(activeMiners, &miner)
			c.JSON(http.StatusOK, miner)
		})

		api.GET("/users", authMiddleware(), func(c *gin.Context) {
			c.JSON(http.StatusOK, users)
		})

		api.POST("/users", authMiddleware(), func(c *gin.Context) {
			var user User
			if err := c.BindJSON(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			users = append(users, &user)
			c.JSON(http.StatusOK, user)
		})

		api.GET("/wallets", authMiddleware(), func(c *gin.Context) {
			c.JSON(http.StatusOK, wallets)
		})

		api.POST("/wallets", authMiddleware(), func(c *gin.Context) {
			wallet, err := blockchain.GenerateWallet()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			
			wallets = append(wallets, wallet)
			c.JSON(http.StatusOK, wallet)
		})
	}

	// Start HTTP server
	log.Printf("Starting Alerim node on port %d...", *port)
	go func() {
		if err := router.Run(fmt.Sprintf(":%d", *port)); err != nil {
			log.Fatal(err)
		}
	}()

	// Start mining statistics updater
	go updateMiningStats()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	network.Stop()
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No authorization token provided"})
			return
		}

		// Validate token here
		// For now, we'll accept any token
		c.Next()
	}
}

func updateMiningStats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats.mu.Lock()
		// Update mining statistics here
		// This would typically come from your mining pool implementation
		stats.TotalHashrate = calculateNetworkHashrate()
		stats.ActiveMiners = len(activeMiners)
		stats.Difficulty.Set(blockchain.GetCurrentDifficulty())
		stats.mu.Unlock()
	}
}

func calculateNetworkHashrate() float64 {
	// Implement network hashrate calculation
	// This would typically be based on recent block times and difficulties
	return 0.0
}

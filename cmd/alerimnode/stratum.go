package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// StratumServer handles Stratum protocol connections
type StratumServer struct {
	mu       sync.RWMutex
	pool     *MiningPool
	rewards  *RewardManager
	clients  map[string]*StratumClient
	listener net.Listener
}

// StratumClient represents a connected mining client
type StratumClient struct {
	mu         sync.Mutex
	conn       net.Conn
	reader     *bufio.Reader
	encoder    *json.Encoder
	minerID    string
	difficulty *big.Int
	lastShare  time.Time
	server     *StratumServer
}

// StratumRequest represents a JSON-RPC request from a client
type StratumRequest struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumResponse represents a JSON-RPC response to a client
type StratumResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
	Method string      `json:"method,omitempty"`
	Params []interface{} `json:"params,omitempty"`
}

// NewStratumServer creates a new stratum server instance
func NewStratumServer(pool *MiningPool, rewards *RewardManager, port int) (*StratumServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	return &StratumServer{
		pool:     pool,
		rewards:  rewards,
		clients:  make(map[string]*StratumClient),
		listener: listener,
	}, nil
}

// Start begins accepting stratum connections
func (s *StratumServer) Start() {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				continue
			}

			client := &StratumClient{
				conn:       conn,
				reader:     bufio.NewReader(conn),
				encoder:    json.NewEncoder(conn),
				difficulty: s.pool.vardiff.GetDifficulty(""),
				server:     s,
			}

			go client.handleConnection()
		}
	}()
}

// handleConnection processes messages from a stratum client
func (c *StratumClient) handleConnection() {
	defer c.conn.Close()

	for {
		// Read JSON-RPC request
		data, err := c.reader.ReadBytes('\n')
		if err != nil {
			log.Printf("Error reading from client: %v", err)
			return
		}

		var req StratumRequest
		if err := json.Unmarshal(data, &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			continue
		}

		// Handle different stratum methods
		switch req.Method {
		case "mining.subscribe":
			c.handleSubscribe(req)
		case "mining.authorize":
			c.handleAuthorize(req)
		case "mining.submit":
			c.handleSubmit(req)
		default:
			c.sendError(req.ID, "Unknown method")
		}
	}
}

func (c *StratumClient) handleSubscribe(req StratumRequest) {
	// Generate unique subscription ID
	subscriptionID := fmt.Sprintf("subscription-%d", time.Now().UnixNano())
	
	response := StratumResponse{
		ID: req.ID,
		Result: []interface{}{
			subscriptionID,
			"AlerimStratum/1.0.0",
		},
	}
	
	c.sendResponse(response)

	// Set initial difficulty
	c.sendResponse(StratumResponse{
		ID:     req.ID,
		Method: "mining.set_difficulty",
		Params: []interface{}{fmt.Sprintf("%x", c.difficulty)},
	})
}

func (c *StratumClient) handleAuthorize(req StratumRequest) {
	if len(req.Params) < 2 {
		c.sendError(req.ID, "Invalid parameters")
		return
	}

	username, ok := req.Params[0].(string)
	if !ok {
		c.sendError(req.ID, "Invalid username")
		return
	}

	c.mu.Lock()
	c.minerID = username
	c.mu.Unlock()

	c.server.mu.Lock()
	c.server.clients[username] = c
	c.server.mu.Unlock()

	// Send successful authorization response
	c.sendResponse(StratumResponse{
		ID:     req.ID,
		Result: true,
	})

	// Send initial work
	c.sendWork()
}

func (c *StratumClient) handleSubmit(req StratumRequest) {
	if len(req.Params) < 4 {
		c.sendError(req.ID, "Invalid parameters")
		return
	}

	// Extract share parameters
	workerName := req.Params[0].(string)
	jobID := req.Params[1].(string)
	nonce := req.Params[2].(string)
	hash := req.Params[3].(string)

	// Verify share
	if err := c.server.pool.SubmitShare(workerName, parseNonce(nonce), parseHash(hash)); err != nil {
		c.sendError(req.ID, err.Error())
		return
	}

	// Record share for rewards
	c.server.rewards.AddShare(workerName)
	c.lastShare = time.Now()

	// Send success response
	c.sendResponse(StratumResponse{
		ID:     req.ID,
		Result: true,
	})
}

func (c *StratumClient) sendWork() {
	block := c.server.pool.currentBlock
	if block == nil {
		return
	}

	// Format work data for stratum
	workData := []interface{}{
		fmt.Sprintf("%x", block.PreviousHash),
		fmt.Sprintf("%x", block.MerkleRoot),
		fmt.Sprintf("%x", block.Timestamp.Unix()),
		fmt.Sprintf("%x", c.difficulty),
	}

	notification := StratumResponse{
		Method: "mining.notify",
		Params: workData,
	}

	c.sendResponse(notification)
}

func (c *StratumClient) sendResponse(response StratumResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if err := c.encoder.Encode(response); err != nil {
		log.Printf("Error sending response: %v", err)
	}
}

func (c *StratumClient) sendError(id interface{}, message string) {
	response := StratumResponse{
		ID:    id,
		Error: []interface{}{20, message, nil},
	}
	c.sendResponse(response)
}

// Helper functions for parsing share submissions
func parseNonce(s string) uint64 {
	var nonce uint64
	fmt.Sscanf(s, "%x", &nonce)
	return nonce
}

func parseHash(s string) []byte {
	hash := make([]byte, 32)
	fmt.Sscanf(s, "%x", &hash)
	return hash
}

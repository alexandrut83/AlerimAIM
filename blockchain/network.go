package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// Peer represents a connected peer in the network
type Peer struct {
	Address  string
	Conn     net.Conn
	LastSeen time.Time
}

// Network manages P2P communication
type Network struct {
	blockchain  *Blockchain
	peers       map[string]*Peer
	listener    net.Listener
	port        int
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// Message types
const (
	MsgTypeBlock        = "block"
	MsgTypeTransaction  = "transaction"
	MsgTypeGetBlocks    = "getblocks"
	MsgTypeGetMempool   = "getmempool"
	MsgTypePing         = "ping"
)

// Message represents a P2P network message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewNetwork creates a new P2P network
func NewNetwork(blockchain *Blockchain, port int) (*Network, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	network := &Network{
		blockchain: blockchain,
		peers:      make(map[string]*Peer),
		port:       port,
		ctx:        ctx,
		cancel:     cancel,
	}
	
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		cancel()
		return nil, err
	}
	
	network.listener = listener
	
	go network.acceptConnections()
	go network.maintainPeers()
	
	return network, nil
}

// Connect connects to a peer
func (n *Network) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	
	peer := &Peer{
		Address:  address,
		Conn:     conn,
		LastSeen: time.Now(),
	}
	
	n.mu.Lock()
	n.peers[address] = peer
	n.mu.Unlock()
	
	go n.handlePeer(peer)
	
	return nil
}

// BroadcastTransaction broadcasts a transaction to all peers
func (n *Network) BroadcastTransaction(tx *Transaction) {
	msg := Message{
		Type:    MsgTypeTransaction,
		Payload: tx.Serialize(),
	}
	
	n.broadcast(msg)
}

// BroadcastBlock broadcasts a block to all peers
func (n *Network) BroadcastBlock(block *Block) {
	msg := Message{
		Type:    MsgTypeBlock,
		Payload: block.Serialize(),
	}
	
	n.broadcast(msg)
}

// broadcast sends a message to all connected peers
func (n *Network) broadcast(msg Message) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}
	
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	for _, peer := range n.peers {
		peer.Conn.Write(msgBytes)
	}
}

// acceptConnections accepts incoming peer connections
func (n *Network) acceptConnections() {
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			conn, err := n.listener.Accept()
			if err != nil {
				continue
			}
			
			peer := &Peer{
				Address:  conn.RemoteAddr().String(),
				Conn:     conn,
				LastSeen: time.Now(),
			}
			
			n.mu.Lock()
			n.peers[peer.Address] = peer
			n.mu.Unlock()
			
			go n.handlePeer(peer)
		}
	}
}

// handlePeer handles communication with a peer
func (n *Network) handlePeer(peer *Peer) {
	defer func() {
		peer.Conn.Close()
		n.mu.Lock()
		delete(n.peers, peer.Address)
		n.mu.Unlock()
	}()
	
	decoder := json.NewDecoder(peer.Conn)
	
	for {
		select {
		case <-n.ctx.Done():
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			
			peer.LastSeen = time.Now()
			
			switch msg.Type {
			case MsgTypeBlock:
				var block Block
				if err := json.Unmarshal(msg.Payload, &block); err != nil {
					continue
				}
				// Handle new block
				n.blockchain.AddBlock([]*Transaction{})
				
			case MsgTypeTransaction:
				var tx Transaction
				if err := json.Unmarshal(msg.Payload, &tx); err != nil {
					continue
				}
				// Handle new transaction
				n.blockchain.AddTransaction(&tx)
				
			case MsgTypeGetBlocks:
				// Send blocks
				
			case MsgTypeGetMempool:
				// Send mempool transactions
				
			case MsgTypePing:
				// Respond to ping
			}
		}
	}
}

// maintainPeers removes inactive peers
func (n *Network) maintainPeers() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.mu.Lock()
			for addr, peer := range n.peers {
				if time.Since(peer.LastSeen) > 5*time.Minute {
					peer.Conn.Close()
					delete(n.peers, addr)
				}
			}
			n.mu.Unlock()
		}
	}
}

// Stop stops the network
func (n *Network) Stop() {
	n.cancel()
	n.listener.Close()
	
	n.mu.Lock()
	for _, peer := range n.peers {
		peer.Conn.Close()
	}
	n.mu.Unlock()
}

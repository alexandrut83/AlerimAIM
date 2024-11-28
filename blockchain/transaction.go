package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

// Transaction represents a transaction in the blockchain
type Transaction struct {
	Version  uint32
	Inputs   []TxInput
	Outputs  []TxOutput
	LockTime uint32
	Hash     [32]byte
}

// TxInput represents a transaction input
type TxInput struct {
	PrevTxHash  [32]byte
	PrevTxIndex uint32
	Script      []byte
	Sequence    uint32
}

// TxOutput represents a transaction output
type TxOutput struct {
	Value  uint64
	Script []byte
}

// NewTransaction creates a new transaction
func NewTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		Version:  1,
		Inputs:   inputs,
		Outputs:  outputs,
		LockTime: 0,
	}
	tx.Hash = tx.CalculateHash()
	return tx
}

// CalculateHash calculates the SHA-256 hash of the transaction
func (tx *Transaction) CalculateHash() [32]byte {
	buf := bytes.NewBuffer(nil)
	
	binary.Write(buf, binary.LittleEndian, tx.Version)
	
	// Write inputs
	binary.Write(buf, binary.LittleEndian, uint32(len(tx.Inputs)))
	for _, input := range tx.Inputs {
		buf.Write(input.PrevTxHash[:])
		binary.Write(buf, binary.LittleEndian, input.PrevTxIndex)
		binary.Write(buf, binary.LittleEndian, uint32(len(input.Script)))
		buf.Write(input.Script)
		binary.Write(buf, binary.LittleEndian, input.Sequence)
	}
	
	// Write outputs
	binary.Write(buf, binary.LittleEndian, uint32(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		binary.Write(buf, binary.LittleEndian, output.Value)
		binary.Write(buf, binary.LittleEndian, uint32(len(output.Script)))
		buf.Write(output.Script)
	}
	
	binary.Write(buf, binary.LittleEndian, tx.LockTime)
	
	return sha256.Sum256(buf.Bytes())
}

// Sign signs the transaction with the given private key
func (tx *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
	hash := tx.CalculateHash()
	
	for i := range tx.Inputs {
		r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
		if err != nil {
			return err
		}
		
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[i].Script = signature
	}
	
	return nil
}

// Verify verifies the transaction signature with the given public key
func (tx *Transaction) Verify(publicKey *ecdsa.PublicKey) bool {
	hash := tx.CalculateHash()
	
	for _, input := range tx.Inputs {
		if len(input.Script) != 64 {
			return false
		}
		
		r := new(big.Int).SetBytes(input.Script[:32])
		s := new(big.Int).SetBytes(input.Script[32:])
		
		if !ecdsa.Verify(publicKey, hash[:], r, s) {
			return false
		}
	}
	
	return true
}

// IsCoinbase checks if this is a coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && bytes.Equal(tx.Inputs[0].PrevTxHash[:], make([]byte, 32))
}

// CreateCoinbase creates a new coinbase transaction with the given reward
func CreateCoinbase(reward uint64, recipientScript []byte) *Transaction {
	input := TxInput{
		PrevTxHash:  [32]byte{},
		PrevTxIndex: 0xFFFFFFFF,
		Script:      []byte{},
		Sequence:    0xFFFFFFFF,
	}
	
	output := TxOutput{
		Value:  reward,
		Script: recipientScript,
	}
	
	return NewTransaction([]TxInput{input}, []TxOutput{output})
}

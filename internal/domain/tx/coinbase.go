package tx

import (
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// NewCoinbaseTx creates a coinbase transaction that awards the block reward to the miner.
// It has a single input with a zero hash and vout=0xFFFFFFFF (coinbase marker),
// and a single output paying the reward to the miner address.
// Uses height=0 for backward compatibility.
func NewCoinbaseTx(minerAddress string, reward int64) *Transaction {
	return NewCoinbaseTxWithHeight(minerAddress, reward, 0)
}

// NewCoinbaseTxWithHeight creates a coinbase transaction with a specific block height
// encoded in the coinbase data. This ensures unique coinbase transaction IDs
// per block (Bitcoin BIP34 convention).
func NewCoinbaseTxWithHeight(minerAddress string, reward int64, height uint64) *Transaction {
	input := TxInput{
		txID:      block.Hash{}, // zero hash = coinbase marker
		vout:      0xFFFFFFFF,   // max uint32 = coinbase marker
		signature: nil,
		pubKey:    nil,
	}
	output := TxOutput{
		value:   reward,
		address: minerAddress,
	}
	tx := &Transaction{
		inputs:       []TxInput{input},
		outputs:      []TxOutput{output},
		coinbaseData: fmt.Sprintf("height:%d", height),
	}
	tx.id = tx.ComputeID()
	return tx
}

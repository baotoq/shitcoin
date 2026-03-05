package tx

import "github.com/baotoq/shitcoin/internal/domain/block"

// NewCoinbaseTx creates a coinbase transaction that awards the block reward to the miner.
// It has a single input with a zero hash and vout=0xFFFFFFFF (coinbase marker),
// and a single output paying the reward to the miner address.
func NewCoinbaseTx(minerAddress string, reward int64) *Transaction {
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
		inputs:  []TxInput{input},
		outputs: []TxOutput{output},
	}
	tx.id = tx.ComputeID()
	return tx
}

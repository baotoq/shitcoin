package tx

// NewCoinbaseTx creates a coinbase transaction that awards the block reward to the miner.
// It has a single input with a zero hash and vout=0xFFFFFFFF (coinbase marker),
// and a single output paying the reward to the miner address.
func NewCoinbaseTx(minerAddress string, reward int64) *Transaction {
	panic("not implemented")
}

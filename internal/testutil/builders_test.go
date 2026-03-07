package testutil

import (
	"math/big"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustCreateBlock_Genesis(t *testing.T) {
	b := MustCreateBlock(t, 0, block.Hash{})

	assert.Equal(t, uint64(0), b.Height())
	assert.False(t, b.Hash().IsZero(), "block hash should not be zero")

	// Verify valid PoW
	pow := &block.ProofOfWork{}
	assert.True(t, pow.Validate(b), "block should have valid PoW")
}

func TestMustCreateBlock_NonGenesis(t *testing.T) {
	genesis := MustCreateBlock(t, 0, block.Hash{})
	b := MustCreateBlock(t, 5, genesis.Hash())

	assert.Equal(t, uint64(5), b.Height())
	assert.False(t, b.Hash().IsZero(), "block hash should not be zero")
	assert.Equal(t, genesis.Hash(), b.PrevBlockHash(), "prevHash should link to genesis")

	pow := &block.ProofOfWork{}
	assert.True(t, pow.Validate(b), "block should have valid PoW")
}

func TestMustCreateBlockWithAddr(t *testing.T) {
	minerAddr := "1MinerTestAddress"
	b := MustCreateBlockWithAddr(t, 0, block.Hash{}, minerAddr)

	assert.Equal(t, uint64(0), b.Height())
	assert.False(t, b.Hash().IsZero())

	// Verify coinbase pays to specified address
	rawTxs := b.RawTransactions()
	require.Len(t, rawTxs, 1, "block should have 1 transaction (coinbase)")
	coinbaseTx, ok := rawTxs[0].(*tx.Transaction)
	require.True(t, ok, "transaction should be *tx.Transaction")
	assert.True(t, coinbaseTx.IsCoinbase())
	require.Len(t, coinbaseTx.Outputs(), 1)
	assert.Equal(t, minerAddr, coinbaseTx.Outputs()[0].Address())
}

func TestMustCreateBlockChain(t *testing.T) {
	count := 5
	blocks := MustCreateBlockChain(t, count)

	require.Len(t, blocks, count, "chain should have %d blocks", count)

	// Check height sequence
	for i, b := range blocks {
		assert.Equal(t, uint64(i), b.Height(), "block %d should have height %d", i, i)
	}

	// Check hash linkage: each block's prevHash == previous block's hash
	assert.True(t, blocks[0].PrevBlockHash().IsZero(), "genesis prevHash should be zero")
	for i := 1; i < count; i++ {
		assert.Equal(t, blocks[i-1].Hash(), blocks[i].PrevBlockHash(),
			"block %d prevHash should equal block %d hash", i, i-1)
	}
}

func TestMustCreateWallet(t *testing.T) {
	w := MustCreateWallet(t)

	assert.NotEmpty(t, w.Address(), "wallet should have non-empty address")
	assert.NotNil(t, w.PrivateKey(), "wallet should have non-nil private key")
}

func TestMustBuildSignedTx(t *testing.T) {
	w := MustCreateWallet(t)

	// Create a mock UTXO repo and set with a coinbase UTXO for the wallet
	mockRepo := NewMockUTXORepo()
	utxoSet := utxo.NewSet(mockRepo)

	// Create a coinbase tx paying to our wallet
	coinbaseTx := tx.NewCoinbaseTx(w.Address(), 5000000000)

	// Apply coinbase to the UTXO set
	_, err := utxoSet.ApplyBlock(0, []*tx.Transaction{coinbaseTx})
	require.NoError(t, err)

	// Build a signed spend tx
	spendTx := MustBuildSignedTx(t, utxoSet, w.PrivateKey(), w.Address())

	// Verify the transaction is signed and valid
	assert.False(t, spendTx.ID().IsZero(), "tx should have non-zero ID")
	assert.True(t, tx.VerifyTransaction(spendTx), "transaction signature should be valid")

	// Verify it has inputs and outputs
	assert.NotEmpty(t, spendTx.Inputs(), "tx should have inputs")
	assert.NotEmpty(t, spendTx.Outputs(), "tx should have outputs")

	// Verify it's not a coinbase
	assert.False(t, spendTx.IsCoinbase(), "spend tx should not be coinbase")

	// Verify input signature fields are set
	for _, input := range spendTx.Inputs() {
		assert.NotEmpty(t, input.Signature(), "input should have signature")
		assert.NotEmpty(t, input.PubKey(), "input should have pubkey")
	}
}

func TestMustCreateBlock_PoWDifficulty(t *testing.T) {
	// Verify that blocks are mined with low difficulty (bits=1)
	b := MustCreateBlock(t, 0, block.Hash{})

	// With bits=1, target = 1 << 255, which is very large
	// so hash should be below this target
	target := block.BitsToTarget(1)
	hashInt := new(big.Int).SetBytes(b.Hash().Bytes())
	assert.True(t, hashInt.Cmp(target) == -1, "hash should be below difficulty target")
}

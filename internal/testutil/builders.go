// Package testutil provides shared test builders and mock repository implementations
// for use across all test packages. Eliminates duplication of mock setup code.
package testutil

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/domain/wallet"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
)

const (
	// TestDifficultyBits is the difficulty used for test mining (very low for speed).
	TestDifficultyBits uint32 = 1
	// TestBlockReward is the block reward used in test coinbase transactions.
	TestBlockReward int64 = 5000000000 // 50 BTC in satoshis
)

// MustCreateBlock creates a mined block at the given height with the given prevHash.
// Uses a default test miner address. Panics via t.Fatal on any error.
func MustCreateBlock(t *testing.T, height uint64, prevHash block.Hash) *block.Block {
	t.Helper()
	return MustCreateBlockWithAddr(t, height, prevHash, "1TestAddr")
}

// MustCreateBlockWithAddr creates a mined block at the given height with a specific miner address.
// Creates a coinbase transaction, computes merkle root, creates and mines the block.
func MustCreateBlockWithAddr(t *testing.T, height uint64, prevHash block.Hash, minerAddr string) *block.Block {
	t.Helper()

	// Create coinbase transaction
	coinbaseTx := tx.NewCoinbaseTxWithHeight(minerAddr, TestBlockReward, height)
	txs := []any{coinbaseTx}

	// Compute merkle root from transaction hash
	txHash := coinbaseTx.ID()
	merkleRoot := block.ComputeMerkleRoot([]block.Hash{txHash})

	// Create block
	var b *block.Block
	var err error
	if height == 0 {
		b, err = block.NewGenesisBlock("test genesis", TestDifficultyBits, txs, merkleRoot)
	} else {
		b, err = block.NewBlock(prevHash, height, TestDifficultyBits, txs, merkleRoot)
	}
	require.NoError(t, err)

	// Mine the block
	pow := &block.ProofOfWork{}
	err = pow.Mine(b)
	require.NoError(t, err)

	return b
}

// MustCreateBlockChain creates a chain of count blocks starting from genesis.
// Returns a slice of [genesis, block1, block2, ...] with correct hash linkage.
func MustCreateBlockChain(t *testing.T, count int) []*block.Block {
	t.Helper()

	blocks := make([]*block.Block, 0, count)
	prevHash := block.Hash{}

	for i := 0; i < count; i++ {
		b := MustCreateBlock(t, uint64(i), prevHash)
		blocks = append(blocks, b)
		prevHash = b.Hash()
	}

	return blocks
}

// MustCreateWallet creates a new wallet with a generated key pair.
// Panics via t.Fatal on any error.
func MustCreateWallet(t *testing.T) *wallet.Wallet {
	t.Helper()

	w, err := wallet.NewWallet()
	require.NoError(t, err)
	return w
}

// MustBuildSignedTx builds a signed transaction that spends from the first available UTXO
// belonging to fromAddr. The UTXO set must already contain UTXOs for fromAddr.
// Creates a transaction spending the first UTXO to a dummy address, signs it with privKey.
func MustBuildSignedTx(t *testing.T, utxoSet *utxo.Set, privKey *btcec.PrivateKey, fromAddr string) *tx.Transaction {
	t.Helper()

	// Get UTXOs for the address
	utxos, err := utxoSet.GetByAddress(fromAddr)
	require.NoError(t, err)
	require.NotEmpty(t, utxos, "no UTXOs found for address %s", fromAddr)

	// Use the first UTXO
	spendUTXO := utxos[0]

	// Create input referencing the UTXO
	input := tx.NewTxInput(spendUTXO.TxID(), spendUTXO.Vout())

	// Create output sending to a dummy address (spend entire value minus small fee)
	output := tx.NewTxOutput(spendUTXO.Value()-1000, "1DummyRecipientAddr")

	// Build transaction
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})

	// Sign
	err = tx.SignTransaction(spendTx, privKey)
	require.NoError(t, err)

	return spendTx
}

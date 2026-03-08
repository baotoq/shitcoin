package bbolt

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxModelFromDomain_Coinbase(t *testing.T) {
	w := testutil.MustCreateWallet(t)
	coinbaseTx := tx.NewCoinbaseTxWithHeight(w.Address(), 5_000_000_000, 0)

	model := TxModelFromDomain(coinbaseTx)

	assert.Equal(t, coinbaseTx.ID().String(), model.ID)
	require.Len(t, model.Inputs, 1)
	require.Len(t, model.Outputs, 1)
	assert.Equal(t, int64(5_000_000_000), model.Outputs[0].Value)
}

func TestTxModel_RoundTrip_Coinbase(t *testing.T) {
	w := testutil.MustCreateWallet(t)
	coinbaseTx := tx.NewCoinbaseTxWithHeight(w.Address(), 5_000_000_000, 0)

	model := TxModelFromDomain(coinbaseTx)
	restored, err := model.ToDomain()
	require.NoError(t, err)

	assert.Equal(t, coinbaseTx.ID(), restored.ID())
	require.Len(t, restored.Outputs(), len(coinbaseTx.Outputs()))
	assert.Equal(t, coinbaseTx.Outputs()[0].Value(), restored.Outputs()[0].Value())
	assert.Equal(t, coinbaseTx.Outputs()[0].Address(), restored.Outputs()[0].Address())
}

func TestTxModel_RoundTrip_SignedTx(t *testing.T) {
	w := testutil.MustCreateWallet(t)

	// Create a coinbase tx to provide UTXOs
	coinbaseTx := tx.NewCoinbaseTxWithHeight(w.Address(), 5_000_000_000, 0)

	// Build a UTXO set with the coinbase output
	mockRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(mockRepo)
	_, err := utxoSet.ApplyBlock(0, []*tx.Transaction{coinbaseTx})
	require.NoError(t, err)

	// Build a signed transaction
	signedTx := testutil.MustBuildSignedTx(t, utxoSet, w.PrivateKey(), w.Address())

	// Round-trip through TxModel
	model := TxModelFromDomain(signedTx)
	restored, err := model.ToDomain()
	require.NoError(t, err)

	assert.Equal(t, signedTx.ID(), restored.ID())
	require.Len(t, restored.Inputs(), len(signedTx.Inputs()))

	// Verify inputs have non-empty signature and pubkey after round-trip
	for i, input := range restored.Inputs() {
		assert.NotEmpty(t, input.Signature(), "input %d signature should not be empty", i)
		assert.NotEmpty(t, input.PubKey(), "input %d pubkey should not be empty", i)
	}
}

func TestBlockModelFromDomain_WithTransactions(t *testing.T) {
	b := testutil.MustCreateBlock(t, 0, block.Hash{})

	// Convert to model
	model := BlockModelFromDomain(b)
	require.NotEmpty(t, model.Transactions, "block model should have transactions")

	// Convert back to domain
	restored, err := model.ToDomain()
	require.NoError(t, err)

	assert.Equal(t, b.Hash(), restored.Hash())
	require.Len(t, restored.RawTransactions(), len(b.RawTransactions()))

	// Verify first tx ID matches
	originalTx := b.RawTransactions()[0].(*tx.Transaction)
	restoredTx := restored.RawTransactions()[0].(*tx.Transaction)
	assert.Equal(t, originalTx.ID(), restoredTx.ID())
}

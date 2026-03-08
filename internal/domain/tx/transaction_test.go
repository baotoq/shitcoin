package tx

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// --- TxInput tests ---

func TestNewTxInput(t *testing.T) {
	txID := block.DoubleSHA256([]byte("prev-tx"))
	input := NewTxInput(txID, 0)

	assert.Equal(t, txID, input.TxID())
	assert.Equal(t, uint32(0), input.Vout())
	assert.Nil(t, input.Signature())
	assert.Nil(t, input.PubKey())
}

func TestTxInputSetSignatureAndPubKey(t *testing.T) {
	txID := block.DoubleSHA256([]byte("prev-tx"))
	input := NewTxInput(txID, 1)

	sig := []byte{0x30, 0x44}
	pk := []byte{0x02, 0xAB}

	input.SetSignature(sig)
	input.SetPubKey(pk)

	assert.Equal(t, sig, input.Signature())
	assert.Equal(t, pk, input.PubKey())
}

// --- TxOutput tests ---

func TestNewTxOutput(t *testing.T) {
	output := NewTxOutput(5000000000, "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")

	assert.Equal(t, int64(5000000000), output.Value())
	assert.Equal(t, "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", output.Address())
}

// --- Transaction tests ---

func TestNewTransaction(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	tx := NewTransaction(inputs, outputs)

	assert.False(t, tx.ID().IsZero())
	assert.Len(t, tx.Inputs(), 1)
	assert.Len(t, tx.Outputs(), 1)
}

func TestTransactionComputeIDDeterministic(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	tx1 := NewTransaction(inputs, outputs)
	tx2 := NewTransaction(inputs, outputs)

	assert.Equal(t, tx1.ID(), tx2.ID())
}

func TestTransactionComputeIDExcludesSignature(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	input := NewTxInput(prevTxID, 0)
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	// Create transaction without signature
	tx1 := NewTransaction([]TxInput{input}, outputs)
	id1 := tx1.ComputeID()

	// Set signature on input and create new transaction
	input.SetSignature([]byte{0x30, 0x44, 0x02, 0x20})
	input.SetPubKey([]byte{0x02, 0xAB, 0xCD})
	tx2 := NewTransaction([]TxInput{input}, outputs)
	id2 := tx2.ComputeID()

	assert.Equal(t, id1, id2, "ComputeID should exclude signature")
}

func TestTransactionIsCoinbase(t *testing.T) {
	// Regular transaction should not be coinbase
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	regularTx := NewTransaction(
		[]TxInput{NewTxInput(prevTxID, 0)},
		[]TxOutput{NewTxOutput(1000, "addr1")},
	)
	assert.False(t, regularTx.IsCoinbase())

	// Coinbase transaction
	coinbaseTx := NewCoinbaseTx("miner-addr", 5000000000)
	assert.True(t, coinbaseTx.IsCoinbase())
}

func TestReconstructTransaction(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	originalTx := NewTransaction(inputs, outputs)
	reconstructed := ReconstructTransaction(originalTx.ID(), inputs, outputs)

	assert.Equal(t, originalTx.ID(), reconstructed.ID())
}

// --- Coinbase tests ---

func TestNewCoinbaseTx(t *testing.T) {
	reward := int64(5000000000) // 50 coins
	minerAddr := "1MinerAddress"

	tx := NewCoinbaseTx(minerAddr, reward)

	assert.True(t, tx.IsCoinbase())
	require.Len(t, tx.Inputs(), 1)
	assert.True(t, tx.Inputs()[0].TxID().IsZero())
	assert.Equal(t, uint32(0xFFFFFFFF), tx.Inputs()[0].Vout())
	require.Len(t, tx.Outputs(), 1)
	assert.Equal(t, reward, tx.Outputs()[0].Value())
	assert.Equal(t, minerAddr, tx.Outputs()[0].Address())
	assert.False(t, tx.ID().IsZero())
}

// --- Signing tests ---

func TestSignAndVerifyTransaction(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	require.NoError(t, SignTransaction(tx, privKey))

	// Verify signatures were set
	assert.NotEmpty(t, tx.Inputs()[0].Signature())
	assert.NotEmpty(t, tx.Inputs()[0].PubKey())

	// Verify the transaction
	assert.True(t, VerifyTransaction(tx))
}

func TestVerifyTransactionTamperedOutput(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	require.NoError(t, SignTransaction(tx, privKey))

	// Tamper with the output - create a new transaction with different output but same signed inputs
	tamperedOutputs := []TxOutput{NewTxOutput(9999, "attacker-addr")}
	tamperedTx := &Transaction{
		id:      tx.ID(),
		inputs:  tx.Inputs(),
		outputs: tamperedOutputs,
	}

	assert.False(t, VerifyTransaction(tamperedTx))
}

func TestVerifyTransactionWrongKey(t *testing.T) {
	privKey1, err := btcec.NewPrivateKey()
	require.NoError(t, err)
	privKey2, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Sign with key1
	require.NoError(t, SignTransaction(tx, privKey1))

	// Replace public key with key2's public key (but keep key1's signature)
	tx.inputs[0].SetPubKey(privKey2.PubKey().SerializeCompressed())

	assert.False(t, VerifyTransaction(tx))
}

func TestVerifyCoinbaseTransaction(t *testing.T) {
	coinbaseTx := NewCoinbaseTx("miner-addr", 5000000000)

	// Coinbase transactions should always verify (no signatures to check)
	assert.True(t, VerifyTransaction(coinbaseTx))
}

// --- Validator tests ---

func TestValidateTransactionRejectsNegativeOutputValues(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(-100, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	require.Error(t, err)
}

func TestValidateTransactionRejectsZeroOutputValues(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(0, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	require.Error(t, err)
}

func TestValidateTransactionRejectsSumMismatch(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(2000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Input value is 1000 but output is 2000
	err := ValidateTransaction(tx, []int64{1000})
	require.Error(t, err)
}

func TestValidateTransactionAcceptsExactSpend(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	assert.NoError(t, err)
}

func TestValidateTransactionAcceptsImplicitFee(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(900, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Input 1000, output 900 -- 100 satoshi fee
	err := ValidateTransaction(tx, []int64{1000})
	assert.NoError(t, err)
}

func TestValidateTransactionRejectsNoInputs(t *testing.T) {
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction([]TxInput{}, outputs)

	err := ValidateTransaction(tx, nil)
	require.Error(t, err)
}

func TestValidateTransactionRejectsNoOutputs(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	tx := NewTransaction(inputs, []TxOutput{})

	err := ValidateTransaction(tx, nil)
	require.Error(t, err)
}

// --- ValidateCoinbase tests ---

func TestValidateCoinbaseAcceptsValid(t *testing.T) {
	tx := NewCoinbaseTx("miner-addr", 5000000000)

	err := ValidateCoinbase(tx, 5000000000)
	assert.NoError(t, err)
}

func TestValidateCoinbaseRejectsWrongReward(t *testing.T) {
	tx := NewCoinbaseTx("miner-addr", 5000000000)

	err := ValidateCoinbase(tx, 2500000000)
	require.Error(t, err)
}

func TestValidateCoinbaseRejectsNonCoinbase(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateCoinbase(tx, 1000)
	require.Error(t, err)
}

// --- Change output tests ---

func TestCreateTransactionWithChangeExact(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{1000}

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 1000, "change-addr", 0)
	require.NoError(t, err)

	// Exact spend -- no change output needed
	require.Len(t, tx.Outputs(), 1)
	assert.Equal(t, int64(1000), tx.Outputs()[0].Value())
	assert.Equal(t, "recipient", tx.Outputs()[0].Address())
}

func TestCreateTransactionWithChangeHasChange(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{5000}

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 3000, "change-addr", 0)
	require.NoError(t, err)

	// Should have 2 outputs: payment + change
	require.Len(t, tx.Outputs(), 2)

	// First output: payment
	assert.Equal(t, int64(3000), tx.Outputs()[0].Value())
	assert.Equal(t, "recipient", tx.Outputs()[0].Address())

	// Second output: change
	assert.Equal(t, int64(2000), tx.Outputs()[1].Value())
	assert.Equal(t, "change-addr", tx.Outputs()[1].Address())
}

func TestCreateTransactionWithChangeInsufficientFunds(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{500}

	_, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 1000, "change-addr", 0)
	assert.ErrorIs(t, err, ErrInsufficientFunds)
}

func TestCreateTransactionWithChangeMultipleInputs(t *testing.T) {
	txID1 := block.DoubleSHA256([]byte("prev-tx-1"))
	txID2 := block.DoubleSHA256([]byte("prev-tx-2"))
	inputs := []TxInput{NewTxInput(txID1, 0), NewTxInput(txID2, 1)}
	inputValues := []int64{3000, 4000} // total 7000

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 5000, "change-addr", 0)
	require.NoError(t, err)

	require.Len(t, tx.Outputs(), 2)
	assert.Equal(t, int64(5000), tx.Outputs()[0].Value())
	assert.Equal(t, int64(2000), tx.Outputs()[1].Value())
}

func TestCreateTransactionWithChangeFee(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{5000}

	// Fee of 500: change = 5000 - 3000 - 500 = 1500
	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 3000, "change-addr", 500)
	require.NoError(t, err)

	require.Len(t, tx.Outputs(), 2)
	assert.Equal(t, int64(3000), tx.Outputs()[0].Value())
	assert.Equal(t, int64(1500), tx.Outputs()[1].Value())
}

func TestCreateTransactionWithChangeFeeExactSpend(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{5000}

	// Fee of 2000: change = 5000 - 3000 - 2000 = 0 (no change output)
	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 3000, "change-addr", 2000)
	require.NoError(t, err)

	require.Len(t, tx.Outputs(), 1)
	assert.Equal(t, int64(3000), tx.Outputs()[0].Value())
}

func TestCreateTransactionWithChangeFeeInsufficientFunds(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{5000}

	// Fee of 3000 + amount 3000 = 6000 > 5000 available
	_, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 3000, "change-addr", 3000)
	assert.ErrorIs(t, err, ErrInsufficientFunds)
}

// --- SignTransaction coinbase early-return ---

func TestSignTransaction_CoinbaseNoop(t *testing.T) {
	coinbaseTx := NewCoinbaseTx("miner-addr", 5_000_000_000)

	// Signing a coinbase should be a no-op (return nil, inputs unchanged)
	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	err = SignTransaction(coinbaseTx, privKey)
	assert.NoError(t, err)

	// Inputs should remain unsigned
	assert.Nil(t, coinbaseTx.Inputs()[0].Signature())
	assert.Nil(t, coinbaseTx.Inputs()[0].PubKey())
}

// --- VerifyTransaction invalid signature/pubkey edge cases ---

func TestVerifyTransaction_InvalidSignatures(t *testing.T) {
	tests := []struct {
		name      string
		signature []byte
		pubKey    []byte
	}{
		{
			name:      "empty signature bytes",
			signature: []byte{},
			pubKey:    []byte{0x02, 0xAB},
		},
		{
			name:      "empty pubkey bytes",
			signature: []byte{0x30, 0x44},
			pubKey:    []byte{},
		},
		{
			name:      "invalid (non-parseable) signature bytes",
			signature: []byte{0xFF, 0xFF, 0xFF, 0xFF},
			pubKey:    nil, // will set a real pubkey below
		},
		{
			name:      "invalid (non-parseable) pubkey bytes",
			signature: nil, // will set a real signature below
			pubKey:    []byte{0xFF, 0xFF, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevTxID := block.DoubleSHA256([]byte("prev-tx"))
			input := NewTxInput(prevTxID, 0)

			// For cases that need a real counterpart, generate one
			privKey, err := btcec.NewPrivateKey()
			require.NoError(t, err)

			sig := tt.signature
			pk := tt.pubKey

			if tt.name == "invalid (non-parseable) signature bytes" {
				// Use a valid pubkey with invalid signature
				pk = privKey.PubKey().SerializeCompressed()
			}
			if tt.name == "invalid (non-parseable) pubkey bytes" {
				// Use a valid signature with invalid pubkey
				outputs := []TxOutput{NewTxOutput(1000, "addr1")}
				tmpTx := NewTransaction([]TxInput{input}, outputs)
				require.NoError(t, SignTransaction(tmpTx, privKey))
				sig = tmpTx.Inputs()[0].Signature()
			}

			input.SetSignature(sig)
			input.SetPubKey(pk)

			tx := &Transaction{
				id:      block.DoubleSHA256([]byte("test-id")),
				inputs:  []TxInput{input},
				outputs: []TxOutput{NewTxOutput(1000, "addr1")},
			}

			assert.False(t, VerifyTransaction(tx))
		})
	}
}

// --- ValidateCoinbase multiple outputs ---

func TestValidateCoinbase_MultipleOutputs(t *testing.T) {
	// Construct a coinbase transaction with 2 outputs by manipulating internal fields
	input := TxInput{
		txID: block.Hash{}, // zero hash
		vout: 0xFFFFFFFF,   // coinbase marker
	}
	multiOutputCoinbase := &Transaction{
		inputs: []TxInput{input},
		outputs: []TxOutput{
			NewTxOutput(3_000_000_000, "miner1"),
			NewTxOutput(2_000_000_000, "miner2"),
		},
	}
	multiOutputCoinbase.id = multiOutputCoinbase.ComputeID()

	// Should be recognized as coinbase
	assert.True(t, multiOutputCoinbase.IsCoinbase())

	// ValidateCoinbase should reject it due to multiple outputs
	err := ValidateCoinbase(multiOutputCoinbase, 5_000_000_000)
	assert.ErrorIs(t, err, ErrInvalidCoinbase)
}

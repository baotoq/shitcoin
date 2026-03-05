package tx

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/btcsuite/btcd/btcec/v2"
)

// --- TxInput tests ---

func TestNewTxInput(t *testing.T) {
	txID := block.DoubleSHA256([]byte("prev-tx"))
	input := NewTxInput(txID, 0)

	if input.TxID() != txID {
		t.Errorf("TxID() = %v; want %v", input.TxID(), txID)
	}
	if input.Vout() != 0 {
		t.Errorf("Vout() = %d; want 0", input.Vout())
	}
	if input.Signature() != nil {
		t.Errorf("Signature() should be nil for unsigned input")
	}
	if input.PubKey() != nil {
		t.Errorf("PubKey() should be nil for unsigned input")
	}
}

func TestTxInputSetSignatureAndPubKey(t *testing.T) {
	txID := block.DoubleSHA256([]byte("prev-tx"))
	input := NewTxInput(txID, 1)

	sig := []byte{0x30, 0x44}
	pk := []byte{0x02, 0xAB}

	input.SetSignature(sig)
	input.SetPubKey(pk)

	if len(input.Signature()) != 2 || input.Signature()[0] != 0x30 {
		t.Errorf("Signature() = %x; want %x", input.Signature(), sig)
	}
	if len(input.PubKey()) != 2 || input.PubKey()[0] != 0x02 {
		t.Errorf("PubKey() = %x; want %x", input.PubKey(), pk)
	}
}

// --- TxOutput tests ---

func TestNewTxOutput(t *testing.T) {
	output := NewTxOutput(5000000000, "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")

	if output.Value() != 5000000000 {
		t.Errorf("Value() = %d; want 5000000000", output.Value())
	}
	if output.Address() != "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" {
		t.Errorf("Address() = %q; want %q", output.Address(), "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa")
	}
}

// --- Transaction tests ---

func TestNewTransaction(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	tx := NewTransaction(inputs, outputs)

	if tx.ID().IsZero() {
		t.Error("Transaction ID should not be zero")
	}
	if len(tx.Inputs()) != 1 {
		t.Errorf("Inputs() length = %d; want 1", len(tx.Inputs()))
	}
	if len(tx.Outputs()) != 1 {
		t.Errorf("Outputs() length = %d; want 1", len(tx.Outputs()))
	}
}

func TestTransactionComputeIDDeterministic(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	tx1 := NewTransaction(inputs, outputs)
	tx2 := NewTransaction(inputs, outputs)

	if tx1.ID() != tx2.ID() {
		t.Errorf("Same inputs/outputs should produce same ID: %v != %v", tx1.ID(), tx2.ID())
	}
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

	if id1 != id2 {
		t.Errorf("ComputeID should exclude signature: %v != %v", id1, id2)
	}
}

func TestTransactionIsCoinbase(t *testing.T) {
	// Regular transaction should not be coinbase
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	regularTx := NewTransaction(
		[]TxInput{NewTxInput(prevTxID, 0)},
		[]TxOutput{NewTxOutput(1000, "addr1")},
	)
	if regularTx.IsCoinbase() {
		t.Error("Regular transaction should not be coinbase")
	}

	// Coinbase transaction
	coinbaseTx := NewCoinbaseTx("miner-addr", 5000000000)
	if !coinbaseTx.IsCoinbase() {
		t.Error("Coinbase transaction should return true for IsCoinbase()")
	}
}

func TestReconstructTransaction(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}

	originalTx := NewTransaction(inputs, outputs)
	reconstructed := ReconstructTransaction(originalTx.ID(), inputs, outputs)

	if reconstructed.ID() != originalTx.ID() {
		t.Errorf("Reconstructed ID = %v; want %v", reconstructed.ID(), originalTx.ID())
	}
}

// --- Coinbase tests ---

func TestNewCoinbaseTx(t *testing.T) {
	reward := int64(5000000000) // 50 coins
	minerAddr := "1MinerAddress"

	tx := NewCoinbaseTx(minerAddr, reward)

	if !tx.IsCoinbase() {
		t.Error("NewCoinbaseTx should create a coinbase transaction")
	}
	if len(tx.Inputs()) != 1 {
		t.Fatalf("Coinbase should have exactly 1 input, got %d", len(tx.Inputs()))
	}
	if !tx.Inputs()[0].TxID().IsZero() {
		t.Error("Coinbase input should have zero hash TxID")
	}
	if tx.Inputs()[0].Vout() != 0xFFFFFFFF {
		t.Errorf("Coinbase input vout = %d; want 0xFFFFFFFF", tx.Inputs()[0].Vout())
	}
	if len(tx.Outputs()) != 1 {
		t.Fatalf("Coinbase should have exactly 1 output, got %d", len(tx.Outputs()))
	}
	if tx.Outputs()[0].Value() != reward {
		t.Errorf("Coinbase output value = %d; want %d", tx.Outputs()[0].Value(), reward)
	}
	if tx.Outputs()[0].Address() != minerAddr {
		t.Errorf("Coinbase output address = %q; want %q", tx.Outputs()[0].Address(), minerAddr)
	}
	if tx.ID().IsZero() {
		t.Error("Coinbase transaction should have a computed ID")
	}
}

// --- Signing tests ---

func TestSignAndVerifyTransaction(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	if err := SignTransaction(tx, privKey); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	// Verify signatures were set
	if len(tx.Inputs()[0].Signature()) == 0 {
		t.Error("Signature should be set after signing")
	}
	if len(tx.Inputs()[0].PubKey()) == 0 {
		t.Error("PubKey should be set after signing")
	}

	// Verify the transaction
	if !VerifyTransaction(tx) {
		t.Error("VerifyTransaction should return true for validly signed transaction")
	}
}

func TestVerifyTransactionTamperedOutput(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	if err := SignTransaction(tx, privKey); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	// Tamper with the output - create a new transaction with different output but same signed inputs
	tamperedOutputs := []TxOutput{NewTxOutput(9999, "attacker-addr")}
	tamperedTx := &Transaction{
		id:      tx.ID(),
		inputs:  tx.Inputs(),
		outputs: tamperedOutputs,
	}

	if VerifyTransaction(tamperedTx) {
		t.Error("VerifyTransaction should return false for tampered transaction")
	}
}

func TestVerifyTransactionWrongKey(t *testing.T) {
	privKey1, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key 1: %v", err)
	}
	privKey2, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key 2: %v", err)
	}

	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Sign with key1
	if err := SignTransaction(tx, privKey1); err != nil {
		t.Fatalf("SignTransaction failed: %v", err)
	}

	// Replace public key with key2's public key (but keep key1's signature)
	tx.inputs[0].SetPubKey(privKey2.PubKey().SerializeCompressed())

	if VerifyTransaction(tx) {
		t.Error("VerifyTransaction should return false when pubkey doesn't match signature")
	}
}

func TestVerifyCoinbaseTransaction(t *testing.T) {
	coinbaseTx := NewCoinbaseTx("miner-addr", 5000000000)

	// Coinbase transactions should always verify (no signatures to check)
	if !VerifyTransaction(coinbaseTx) {
		t.Error("VerifyTransaction should return true for coinbase transactions")
	}
}

// --- Validator tests ---

func TestValidateTransactionRejectsNegativeOutputValues(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(-100, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	if err == nil {
		t.Error("ValidateTransaction should reject negative output values")
	}
}

func TestValidateTransactionRejectsZeroOutputValues(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(0, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	if err == nil {
		t.Error("ValidateTransaction should reject zero output values")
	}
}

func TestValidateTransactionRejectsSumMismatch(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(2000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Input value is 1000 but output is 2000
	err := ValidateTransaction(tx, []int64{1000})
	if err == nil {
		t.Error("ValidateTransaction should reject when sum(outputs) > sum(inputs)")
	}
}

func TestValidateTransactionAcceptsExactSpend(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateTransaction(tx, []int64{1000})
	if err != nil {
		t.Errorf("ValidateTransaction should accept exact spend, got: %v", err)
	}
}

func TestValidateTransactionAcceptsImplicitFee(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(900, "addr1")}
	tx := NewTransaction(inputs, outputs)

	// Input 1000, output 900 -- 100 satoshi fee
	err := ValidateTransaction(tx, []int64{1000})
	if err != nil {
		t.Errorf("ValidateTransaction should accept implicit fee, got: %v", err)
	}
}

func TestValidateTransactionRejectsNoInputs(t *testing.T) {
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction([]TxInput{}, outputs)

	err := ValidateTransaction(tx, nil)
	if err == nil {
		t.Error("ValidateTransaction should reject transaction with no inputs")
	}
}

func TestValidateTransactionRejectsNoOutputs(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	tx := NewTransaction(inputs, []TxOutput{})

	err := ValidateTransaction(tx, nil)
	if err == nil {
		t.Error("ValidateTransaction should reject transaction with no outputs")
	}
}

// --- ValidateCoinbase tests ---

func TestValidateCoinbaseAcceptsValid(t *testing.T) {
	tx := NewCoinbaseTx("miner-addr", 5000000000)

	err := ValidateCoinbase(tx, 5000000000)
	if err != nil {
		t.Errorf("ValidateCoinbase should accept valid coinbase, got: %v", err)
	}
}

func TestValidateCoinbaseRejectsWrongReward(t *testing.T) {
	tx := NewCoinbaseTx("miner-addr", 5000000000)

	err := ValidateCoinbase(tx, 2500000000)
	if err == nil {
		t.Error("ValidateCoinbase should reject when output value doesn't match expected reward")
	}
}

func TestValidateCoinbaseRejectsNonCoinbase(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	outputs := []TxOutput{NewTxOutput(1000, "addr1")}
	tx := NewTransaction(inputs, outputs)

	err := ValidateCoinbase(tx, 1000)
	if err == nil {
		t.Error("ValidateCoinbase should reject non-coinbase transaction")
	}
}

// --- Change output tests ---

func TestCreateTransactionWithChangeExact(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{1000}

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 1000, "change-addr")
	if err != nil {
		t.Fatalf("CreateTransactionWithChange failed: %v", err)
	}

	// Exact spend -- no change output needed
	if len(tx.Outputs()) != 1 {
		t.Errorf("Exact spend should have 1 output, got %d", len(tx.Outputs()))
	}
	if tx.Outputs()[0].Value() != 1000 {
		t.Errorf("Output value = %d; want 1000", tx.Outputs()[0].Value())
	}
	if tx.Outputs()[0].Address() != "recipient" {
		t.Errorf("Output address = %q; want %q", tx.Outputs()[0].Address(), "recipient")
	}
}

func TestCreateTransactionWithChangeHasChange(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{5000}

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 3000, "change-addr")
	if err != nil {
		t.Fatalf("CreateTransactionWithChange failed: %v", err)
	}

	// Should have 2 outputs: payment + change
	if len(tx.Outputs()) != 2 {
		t.Fatalf("Should have 2 outputs (payment + change), got %d", len(tx.Outputs()))
	}

	// First output: payment
	if tx.Outputs()[0].Value() != 3000 {
		t.Errorf("Payment output value = %d; want 3000", tx.Outputs()[0].Value())
	}
	if tx.Outputs()[0].Address() != "recipient" {
		t.Errorf("Payment output address = %q; want %q", tx.Outputs()[0].Address(), "recipient")
	}

	// Second output: change
	if tx.Outputs()[1].Value() != 2000 {
		t.Errorf("Change output value = %d; want 2000", tx.Outputs()[1].Value())
	}
	if tx.Outputs()[1].Address() != "change-addr" {
		t.Errorf("Change output address = %q; want %q", tx.Outputs()[1].Address(), "change-addr")
	}
}

func TestCreateTransactionWithChangeInsufficientFunds(t *testing.T) {
	prevTxID := block.DoubleSHA256([]byte("prev-tx"))
	inputs := []TxInput{NewTxInput(prevTxID, 0)}
	inputValues := []int64{500}

	_, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 1000, "change-addr")
	if err == nil {
		t.Error("Should return error for insufficient funds")
	}
	if err != ErrInsufficientFunds {
		t.Errorf("Expected ErrInsufficientFunds, got: %v", err)
	}
}

func TestCreateTransactionWithChangeMultipleInputs(t *testing.T) {
	txID1 := block.DoubleSHA256([]byte("prev-tx-1"))
	txID2 := block.DoubleSHA256([]byte("prev-tx-2"))
	inputs := []TxInput{NewTxInput(txID1, 0), NewTxInput(txID2, 1)}
	inputValues := []int64{3000, 4000} // total 7000

	tx, err := CreateTransactionWithChange(inputs, inputValues, "recipient", 5000, "change-addr")
	if err != nil {
		t.Fatalf("CreateTransactionWithChange failed: %v", err)
	}

	if len(tx.Outputs()) != 2 {
		t.Fatalf("Should have 2 outputs, got %d", len(tx.Outputs()))
	}
	if tx.Outputs()[0].Value() != 5000 {
		t.Errorf("Payment = %d; want 5000", tx.Outputs()[0].Value())
	}
	if tx.Outputs()[1].Value() != 2000 {
		t.Errorf("Change = %d; want 2000", tx.Outputs()[1].Value())
	}
}

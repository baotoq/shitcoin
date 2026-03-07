package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/baotoq/shitcoin/internal/config"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/wallet"
	"github.com/baotoq/shitcoin/internal/svc"
)

// demo dispatches to the appropriate demo subcommand.
func (c *CLI) demo(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: demo <subcommand>")
		fmt.Println("  doublespend  - Demonstrate double-spend detection")
		os.Exit(1)
	}

	switch args[0] {
	case "doublespend":
		c.demoDoubleSpend()
	default:
		fmt.Printf("Unknown demo: %s\n", args[0])
		fmt.Println("Available demos: doublespend")
		os.Exit(1)
	}
}

// demoDoubleSpend runs a scripted in-process scenario demonstrating how the
// blockchain detects and rejects double-spend attacks at both the mempool and
// UTXO set levels.
func (c *CLI) demoDoubleSpend() {
	fmt.Println("=== Double-Spend Attack Demo ===")
	fmt.Println()
	fmt.Println("This demo shows how the blockchain prevents spending the same coins twice.")
	fmt.Println("We will create two transactions that try to spend the same UTXO and observe")
	fmt.Println("the rejection at both the mempool layer and the UTXO set layer.")
	fmt.Println()

	// Create a temp data directory for isolation
	tmpDir, err := os.MkdirTemp("", "shitcoin-demo-*")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build an isolated config with low difficulty for fast mining
	demoCfg := c.config
	demoCfg.Consensus.InitialDifficulty = 1
	demoCfg.Consensus.HalvingInterval = 0 // no halving for demo simplicity
	demoCfg.Storage.DBPath = fmt.Sprintf("%s/demo.db", tmpDir)
	demoCfg.Storage.WalletPath = fmt.Sprintf("%s/wallets.json", tmpDir)

	// Create isolated service context
	demoSvc := svc.NewServiceContext(demoCfg)
	defer demoSvc.Close()

	// Create 3 wallets: Alice (miner), Bob, Charlie
	alice, err := wallet.NewWallet()
	if err != nil {
		fmt.Printf("Error creating Alice's wallet: %v\n", err)
		os.Exit(1)
	}
	bob, err := wallet.NewWallet()
	if err != nil {
		fmt.Printf("Error creating Bob's wallet: %v\n", err)
		os.Exit(1)
	}
	charlie, err := wallet.NewWallet()
	if err != nil {
		fmt.Printf("Error creating Charlie's wallet: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Alice  (miner): %s\n", alice.Address())
	fmt.Printf("Bob:            %s\n", bob.Address())
	fmt.Printf("Charlie:        %s\n", charlie.Address())
	fmt.Println()

	// Initialize chain with Alice as the miner (creates genesis block)
	ctx := context.Background()
	if err := demoSvc.Chain.Initialize(ctx, alice.Address()); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	// Mine 2 more blocks so Alice has enough coins
	for i := 0; i < 2; i++ {
		if _, err := demoSvc.Chain.MineBlock(ctx, alice.Address(), nil, 0); err != nil {
			fmt.Printf("Error mining block: %v\n", err)
			os.Exit(1)
		}
	}

	balance, err := demoSvc.UTXOSet.GetBalance(alice.Address())
	if err != nil {
		fmt.Printf("Error getting balance: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Alice has mined 3 blocks and has %d coins (%d satoshis)\n",
		balance/config.SatoshiPerCoin, balance)
	fmt.Println()

	// Get Alice's UTXOs for building transactions
	utxos, err := demoSvc.UTXOSet.GetByAddress(alice.Address())
	if err != nil {
		fmt.Printf("Error getting UTXOs: %v\n", err)
		os.Exit(1)
	}

	// Use the first UTXO for the double-spend attempt
	spendUTXO := utxos[0]
	sendAmount := int64(10) * config.SatoshiPerCoin

	fmt.Println("--- Step 1: Create Transaction A (Alice -> Bob) ---")
	fmt.Printf("Using UTXO: %s (vout %d, value %d satoshis)\n",
		spendUTXO.TxID().String()[:16]+"...", spendUTXO.Vout(), spendUTXO.Value())
	fmt.Printf("Sending %d coins to Bob...\n", sendAmount/config.SatoshiPerCoin)
	fmt.Println()

	// Build TX-A: Alice sends 10 coins to Bob
	inputA := tx.NewTxInput(spendUTXO.TxID(), spendUTXO.Vout())
	txA, err := tx.CreateTransactionWithChange(
		[]tx.TxInput{inputA},
		[]int64{spendUTXO.Value()},
		bob.Address(),
		sendAmount,
		alice.Address(),
		0,
	)
	if err != nil {
		fmt.Printf("Error creating TX-A: %v\n", err)
		os.Exit(1)
	}
	if err := tx.SignTransaction(txA, alice.PrivateKey()); err != nil {
		fmt.Printf("Error signing TX-A: %v\n", err)
		os.Exit(1)
	}

	// Add TX-A to mempool
	if err := demoSvc.Mempool.Add(txA); err != nil {
		fmt.Printf("Error adding TX-A to mempool: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SUCCESS: Transaction A (%s...) added to mempool\n", txA.ID().String()[:16])
	fmt.Println()

	// Step 2: Try to double-spend the same UTXO
	fmt.Println("--- Step 2: Create Transaction B (Alice -> Charlie, SAME UTXO!) ---")
	fmt.Println("Attempting to send the SAME coins to Charlie (double-spend attack)...")
	fmt.Println()

	// Build TX-B: Alice tries to send the same UTXO to Charlie
	inputB := tx.NewTxInput(spendUTXO.TxID(), spendUTXO.Vout())
	txB, err := tx.CreateTransactionWithChange(
		[]tx.TxInput{inputB},
		[]int64{spendUTXO.Value()},
		charlie.Address(),
		sendAmount,
		alice.Address(),
		0,
	)
	if err != nil {
		fmt.Printf("Error creating TX-B: %v\n", err)
		os.Exit(1)
	}
	if err := tx.SignTransaction(txB, alice.PrivateKey()); err != nil {
		fmt.Printf("Error signing TX-B: %v\n", err)
		os.Exit(1)
	}

	// Try to add TX-B to mempool -- should fail with ErrDoubleSpend
	err = demoSvc.Mempool.Add(txB)
	if err != nil {
		fmt.Printf("REJECTED! Error: %v\n", err)
		fmt.Println("The mempool detected that these coins are already claimed by Transaction A.")
	} else {
		fmt.Println("ERROR: TX-B was unexpectedly accepted (this should not happen)")
		os.Exit(1)
	}
	fmt.Println()

	// Step 3: Mine a block to confirm TX-A
	fmt.Println("--- Step 3: Mining a block to confirm Transaction A ---")

	txs, totalFees := demoSvc.Mempool.DrainByFee(0)
	blk, err := demoSvc.Chain.MineBlock(ctx, alice.Address(), txs, totalFees)
	if err != nil {
		fmt.Printf("Error mining block: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Mined block #%d with %d transaction(s)\n", blk.Height(), len(txs))
	fmt.Println("Transaction A is now confirmed in the blockchain.")
	fmt.Println()

	// Step 4: Try TX-B again after TX-A is confirmed
	fmt.Println("--- Step 4: Trying Transaction B again after confirmation ---")
	fmt.Println("The UTXO has been consumed by Transaction A's block.")
	fmt.Println("Attempting to add Transaction B to mempool again...")
	fmt.Println()

	// Create a fresh mempool (the old one's spent tracking was cleared by drain)
	// But the UTXO set now reflects the confirmed block, so ErrUTXONotFound is expected.
	freshMempool := mempool.New(demoSvc.UTXOSet)
	err = freshMempool.Add(txB)
	if err != nil {
		fmt.Printf("REJECTED! Error: %v\n", err)
		fmt.Println("The UTXO set confirms these coins have already been spent.")
	} else {
		fmt.Println("ERROR: TX-B was unexpectedly accepted (this should not happen)")
		os.Exit(1)
	}
	fmt.Println()

	// Print educational summary
	fmt.Println("=== Summary ===")
	fmt.Println("The blockchain prevents double-spending through two layers:")
	fmt.Println("1. MEMPOOL: Tracks which UTXOs are already claimed by pending transactions")
	fmt.Println("2. UTXO SET: Only unspent outputs can be used as inputs; once spent, they're gone")
	fmt.Println()
	fmt.Println("This is why blockchain transactions are trustworthy without a central authority.")
}

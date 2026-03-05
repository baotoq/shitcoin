package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/wallet"
	"github.com/baotoq/shitcoin/internal/svc"
)

// CLI handles command-line dispatch for all shitcoin subcommands.
type CLI struct {
	svc *svc.ServiceContext
}

// New creates a new CLI instance with the given service context.
func New(svc *svc.ServiceContext) *CLI {
	return &CLI{svc: svc}
}

// Run dispatches to the appropriate subcommand based on args.
// args should be the remaining arguments after global flag parsing (flag.Args()).
func (c *CLI) Run(args []string) {
	if len(args) < 1 {
		c.printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "createwallet":
		c.createWallet()
	case "listaddresses":
		c.listAddresses()
	case "getbalance":
		c.getBalance(args[1:])
	case "send":
		c.send(args[1:])
	case "mine":
		c.mine(args[1:])
	case "startnode":
		c.startNode(args[1:])
	case "printchain":
		c.printChain()
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		c.printUsage()
		os.Exit(1)
	}
}

// printUsage prints all available commands.
func (c *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createwallet              - Generate a new wallet address")
	fmt.Println("  listaddresses             - List all wallet addresses")
	fmt.Println("  getbalance -address ADDR  - Get balance for address")
	fmt.Println("  send -from ADDR -to ADDR -amount AMOUNT - Send coins")
	fmt.Println("  mine -address ADDR        - Mine a new block")
	fmt.Println("  startnode [-port PORT] [-mine ADDR] - Start a node")
	fmt.Println("  printchain                - Print all blocks in the chain")
}

// createWallet generates a new wallet and persists it.
func (c *CLI) createWallet() {
	w, err := wallet.NewWallet()
	if err != nil {
		fmt.Printf("Error creating wallet: %v\n", err)
		os.Exit(1)
	}

	if err := c.svc.WalletRepo.Save(w); err != nil {
		fmt.Printf("Error saving wallet: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("New address: %s\n", w.Address())
}

// listAddresses prints all stored wallet addresses.
func (c *CLI) listAddresses() {
	addresses, err := c.svc.WalletRepo.ListAddresses()
	if err != nil {
		fmt.Printf("Error listing addresses: %v\n", err)
		os.Exit(1)
	}

	if len(addresses) == 0 {
		fmt.Println("No wallets found.")
		return
	}

	for _, addr := range addresses {
		fmt.Println(addr)
	}
}

// getBalance prints the balance for a given address.
func (c *CLI) getBalance(args []string) {
	fs := flag.NewFlagSet("getbalance", flag.ExitOnError)
	address := fs.String("address", "", "Address to check balance for")
	fs.Parse(args)

	if *address == "" {
		fmt.Println("Error: -address is required")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := c.svc.Chain.Initialize(ctx, ""); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	balance, err := c.svc.UTXOSet.GetBalance(*address)
	if err != nil {
		fmt.Printf("Error getting balance: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Balance of %s: %d satoshis\n", *address, balance)
}

// send builds a transaction from wallet UTXOs, signs it, and adds to mempool.
func (c *CLI) send(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	from := fs.String("from", "", "Source address")
	to := fs.String("to", "", "Destination address")
	amount := fs.Int64("amount", 0, "Amount in satoshis")
	fs.Parse(args)

	if *from == "" || *to == "" || *amount <= 0 {
		fmt.Println("Error: -from, -to, and -amount (> 0) are required")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := c.svc.Chain.Initialize(ctx, ""); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	// Load sender wallet
	senderWallet, err := c.svc.WalletRepo.GetByAddress(*from)
	if err != nil {
		fmt.Printf("Error: sender wallet not found: %v\n", err)
		os.Exit(1)
	}

	// Get sender's UTXOs
	utxos, err := c.svc.UTXOSet.GetByAddress(*from)
	if err != nil {
		fmt.Printf("Error getting UTXOs: %v\n", err)
		os.Exit(1)
	}

	// Select UTXOs to cover amount (simple greedy)
	var accumulated int64
	var inputs []tx.TxInput
	var inputValues []int64
	for _, u := range utxos {
		inputs = append(inputs, tx.NewTxInput(u.TxID(), u.Vout()))
		inputValues = append(inputValues, u.Value())
		accumulated += u.Value()
		if accumulated >= *amount {
			break
		}
	}

	if accumulated < *amount {
		fmt.Printf("Error: insufficient funds. Have %d, need %d\n", accumulated, *amount)
		os.Exit(1)
	}

	// Create transaction with change
	transaction, err := tx.CreateTransactionWithChange(inputs, inputValues, *to, *amount, *from)
	if err != nil {
		fmt.Printf("Error creating transaction: %v\n", err)
		os.Exit(1)
	}

	// Sign transaction
	if err := tx.SignTransaction(transaction, senderWallet.PrivateKey()); err != nil {
		fmt.Printf("Error signing transaction: %v\n", err)
		os.Exit(1)
	}

	// Add to mempool
	if err := c.svc.Mempool.Add(transaction); err != nil {
		fmt.Printf("Error adding to mempool: %v\n", err)
		os.Exit(1)
	}

	txID := transaction.ID().String()
	fmt.Printf("Transaction %s... added to mempool\n", txID[:16])
}

// mine drains the mempool and mines a new block.
func (c *CLI) mine(args []string) {
	fs := flag.NewFlagSet("mine", flag.ExitOnError)
	address := fs.String("address", "", "Miner address to receive block reward")
	fs.Parse(args)

	if *address == "" {
		fmt.Println("Error: -address is required")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := c.svc.Chain.Initialize(ctx, *address); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	// Drain mempool
	txs := c.svc.Mempool.DrainAll()

	// Mine block
	blk, err := c.svc.Chain.MineBlock(ctx, *address, txs)
	if err != nil {
		fmt.Printf("Error mining block: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Mined block #%d with %d transaction(s)\n", blk.Height(), len(txs))
	fmt.Printf("Hash: %s\n", blk.Hash().String())
	fmt.Printf("Chain height: %d\n", c.svc.Chain.Height())
}

// startNode starts the node with optional auto-mining.
func (c *CLI) startNode(args []string) {
	fs := flag.NewFlagSet("startnode", flag.ExitOnError)
	port := fs.Int("port", 0, "Port for future P2P use")
	mineAddr := fs.String("mine", "", "Miner address for auto-mining")
	fs.Parse(args)

	_ = port // stored for future P2P use

	ctx := context.Background()

	initAddr := ""
	if *mineAddr != "" {
		initAddr = *mineAddr
	}

	if err := c.svc.Chain.Initialize(ctx, initAddr); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Node started. Chain height: %d\n", c.svc.Chain.Height())

	if *mineAddr != "" {
		c.autoMine(*mineAddr)
	} else {
		fmt.Println("No mining address provided. Node running idle. Press Ctrl+C to stop.")
		c.waitForSignal()
	}
}

// printChain prints all blocks in the chain.
func (c *CLI) printChain() {
	ctx := context.Background()
	if err := c.svc.Chain.Initialize(ctx, ""); err != nil {
		fmt.Printf("Error initializing chain: %v\n", err)
		os.Exit(1)
	}

	height := c.svc.Chain.Height()
	blocks, err := c.svc.ChainRepo.GetBlocksInRange(ctx, 0, height)
	if err != nil {
		fmt.Printf("Error getting blocks: %v\n", err)
		os.Exit(1)
	}

	for _, blk := range blocks {
		fmt.Printf("============ Block #%d ============\n", blk.Height())
		fmt.Printf("Hash:      %s\n", blk.Hash().String())
		fmt.Printf("Prev Hash: %s\n", blk.PrevBlockHash().String())
		fmt.Printf("Bits:      %d\n", blk.Bits())
		fmt.Printf("Timestamp: %d\n", blk.Timestamp())
		fmt.Printf("Nonce:     %d\n", blk.Header().Nonce())
		fmt.Printf("Tx Count:  %d\n", len(blk.RawTransactions()))

		for _, rawTx := range blk.RawTransactions() {
			transaction, ok := rawTx.(*tx.Transaction)
			if !ok {
				continue
			}
			fmt.Printf("  TX: %s\n", transaction.ID().String())
			fmt.Printf("    Inputs:  %d\n", len(transaction.Inputs()))
			fmt.Printf("    Outputs: %d\n", len(transaction.Outputs()))
		}
		fmt.Println()
	}
}

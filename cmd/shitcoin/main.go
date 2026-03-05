package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/baotoq/shitcoin/internal/config"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
)

func main() {
	configFile := flag.String("f", "etc/shitcoin.yaml", "the config file")
	flag.Parse()

	// Suppress go-zero framework noise for clean demo output
	logx.Disable()
	stat.DisableLog()

	// Load configuration
	var c config.Config
	conf.MustLoad(*configFile, &c)
	c.Consensus.ApplyDefaults()

	// Create service context (opens DB, wires dependencies)
	serviceCtx := svc.NewServiceContext(c)
	defer serviceCtx.Close()

	ctx := context.Background()

	// Initialize chain (creates genesis if new, loads existing if not)
	// Use a default miner address for the demo
	minerAddress := "1DemoMinerAddress"

	if err := serviceCtx.Chain.Initialize(ctx, minerAddress); err != nil {
		panic(fmt.Sprintf("failed to initialize chain: %v", err))
	}

	genesis := serviceCtx.Chain.LatestBlock()
	startHeight := genesis.Height()

	if startHeight == 0 {
		fmt.Println("=== Genesis Block Created ===")
	} else {
		fmt.Println("=== Chain Loaded from Disk ===")
	}
	fmt.Printf("  Hash:    %s\n", genesis.Hash().String()[:16]+"...")
	fmt.Printf("  Height:  %d\n", genesis.Height())
	fmt.Printf("  Message: %s\n", genesis.Message())
	fmt.Printf("  Bits:    %d\n", genesis.Bits())
	fmt.Println()

	// Mine blocks
	const blocksToMine = 15
	fmt.Printf("=== Mining %d blocks ===\n", blocksToMine)
	fmt.Println()

	prevBits := serviceCtx.Chain.LatestBlock().Bits()

	for i := 0; i < blocksToMine; i++ {
		start := time.Now()
		blk, err := serviceCtx.Chain.MineBlock(ctx, minerAddress, nil)
		if err != nil {
			panic(fmt.Sprintf("failed to mine block: %v", err))
		}
		elapsed := time.Since(start)

		fmt.Printf("Block #%-4d | hash: %s... | nonce: %-10d | bits: %-3d | time: %v\n",
			blk.Height(),
			blk.Hash().String()[:16],
			blk.Header().Nonce(),
			blk.Bits(),
			elapsed.Round(time.Millisecond),
		)

		// Check for difficulty adjustment
		if blk.Bits() != prevBits {
			fmt.Printf("  >> Difficulty adjusted: %d -> %d\n", prevBits, blk.Bits())
		}
		prevBits = blk.Bits()
	}

	fmt.Println()
	fmt.Println("=== Chain Summary ===")
	fmt.Printf("  Total blocks: %d\n", serviceCtx.Chain.Height()+1)
	fmt.Printf("  Chain height: %d\n", serviceCtx.Chain.Height())
	fmt.Printf("  Current bits: %d\n", serviceCtx.Chain.LatestBlock().Bits())
	fmt.Printf("  Latest hash:  %s\n", serviceCtx.Chain.LatestBlock().Hash())
}

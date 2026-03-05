package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// autoMine runs a continuous mining loop until SIGINT or SIGTERM is received.
func (c *CLI) autoMine(minerAddress string) {
	fmt.Printf("Auto-mining enabled for address: %s\n", minerAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Mining stopped.")
			return
		default:
			txs := c.svc.Mempool.DrainAll()
			blk, err := c.svc.Chain.MineBlock(ctx, minerAddress, txs)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Println("Mining stopped.")
					return
				}
				fmt.Printf("Mining error: %v\n", err)
				continue
			}
			fmt.Printf("Mined block #%d (%s) with %d tx\n", blk.Height(), blk.Hash().String()[:16], len(txs))
		}
	}
}

// waitForSignal blocks until SIGINT or SIGTERM is received.
func (c *CLI) waitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nShutting down...")
}

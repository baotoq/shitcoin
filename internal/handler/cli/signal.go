package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/events"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/handler/ws"
)

// autoMine runs a continuous mining loop until SIGINT or SIGTERM is received.
func (c *CLI) autoMine(minerAddress string) {
	fmt.Printf("Auto-mining enabled for address: %s\n", minerAddress)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up mining progress callback to publish events
	c.svc.Chain.OnMiningProgress = func(p block.MiningProgress) {
		c.svc.EventBus.Publish(events.Event{
			Type: events.EventMiningProgress,
			Payload: ws.MiningProgressPayload{
				Nonce:       p.Nonce,
				HashHex:     p.Hash,
				TargetHex:   p.Target,
				Difficulty:  p.Difficulty,
				BlockHeight: c.svc.Chain.Height() + 1,
			},
		})
	}

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
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMiningStopped,
				Payload: ws.MiningStoppedPayload{BlockHeight: c.svc.Chain.Height(), Reason: "cancelled"},
			})
			fmt.Println("Mining stopped.")
			return
		default:
			nextHeight := c.svc.Chain.Height() + 1
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMiningStarted,
				Payload: ws.MiningStartedPayload{BlockHeight: nextHeight},
			})

			txs, totalFees := c.svc.Mempool.DrainByFee(0)
			blk, err := c.svc.Chain.MineBlock(ctx, minerAddress, txs, totalFees)
			if err != nil {
				if ctx.Err() != nil {
					c.svc.EventBus.Publish(events.Event{
						Type:    events.EventMiningStopped,
						Payload: ws.MiningStoppedPayload{BlockHeight: nextHeight, Reason: "cancelled"},
					})
					fmt.Println("Mining stopped.")
					return
				}
				fmt.Printf("Mining error: %v\n", err)
				continue
			}

			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventNewBlock,
				Payload: map[string]any{"height": blk.Height(), "hash": blk.Hash().String()},
			})
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMempoolChanged,
				Payload: ws.MempoolChangedPayload{Count: c.svc.Mempool.Count()},
			})

			fmt.Printf("Mined block #%d (%s) with %d tx (fees: %d)\n", blk.Height(), blk.Hash().String()[:16], len(txs), totalFees)
		}
	}
}

// autoMineWithP2P runs a continuous mining loop with P2P block broadcasting.
// When a block is received from a peer during mining, the current mining context
// is cancelled so the node accepts the peer's block and restarts mining.
func (c *CLI) autoMineWithP2P(minerAddress string, srv *p2p.Server) {
	fmt.Printf("Auto-mining enabled for address: %s (P2P broadcast active)\n", minerAddress)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// Set up mining progress callback to publish events
	c.svc.Chain.OnMiningProgress = func(p block.MiningProgress) {
		c.svc.EventBus.Publish(events.Event{
			Type: events.EventMiningProgress,
			Payload: ws.MiningProgressPayload{
				Nonce:       p.Nonce,
				HashHex:     p.Hash,
				TargetHex:   p.Target,
				Difficulty:  p.Difficulty,
				BlockHeight: c.svc.Chain.Height() + 1,
			},
		})
	}

	// Set up signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		rootCancel()
	}()

	// mineCancel cancels the current mining attempt when a peer block arrives
	var mineCancel context.CancelFunc

	// Register callback: when a block is received from a peer, cancel mining
	srv.OnBlockReceived(func(blk *block.Block) {
		fmt.Printf("Received block #%d from peer, cancelling current mining\n", blk.Height())
		c.svc.EventBus.Publish(events.Event{
			Type:    events.EventNewBlock,
			Payload: map[string]any{"height": blk.Height(), "hash": blk.Hash().String(), "source": "peer"},
		})
		if mineCancel != nil {
			mineCancel()
		}
	})

	for {
		select {
		case <-rootCtx.Done():
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMiningStopped,
				Payload: ws.MiningStoppedPayload{BlockHeight: c.svc.Chain.Height(), Reason: "cancelled"},
			})
			fmt.Println("Mining stopped.")
			return
		default:
			// Create a cancellable mining context
			mineCtx, cancel := context.WithCancel(rootCtx)
			mineCancel = cancel

			nextHeight := c.svc.Chain.Height() + 1
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMiningStarted,
				Payload: ws.MiningStartedPayload{BlockHeight: nextHeight},
			})

			txs, totalFees := c.svc.Mempool.DrainByFee(0)
			blk, err := c.svc.Chain.MineBlock(mineCtx, minerAddress, txs, totalFees)
			cancel() // clean up mine context

			if err != nil {
				if rootCtx.Err() != nil {
					c.svc.EventBus.Publish(events.Event{
						Type:    events.EventMiningStopped,
						Payload: ws.MiningStoppedPayload{BlockHeight: nextHeight, Reason: "cancelled"},
					})
					fmt.Println("Mining stopped.")
					return
				}
				// Mining was likely cancelled by peer block -- restart loop
				continue
			}

			// Broadcast mined block to peers
			srv.BroadcastBlock(blk, "")

			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventNewBlock,
				Payload: map[string]any{"height": blk.Height(), "hash": blk.Hash().String(), "source": "mined"},
			})
			c.svc.EventBus.Publish(events.Event{
				Type:    events.EventMempoolChanged,
				Payload: ws.MempoolChangedPayload{Count: c.svc.Mempool.Count()},
			})

			fmt.Printf("Mined block #%d (%s) with %d tx (fees: %d)\n", blk.Height(), blk.Hash().String()[:16], len(txs), totalFees)
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

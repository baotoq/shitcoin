package p2p

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// syncState tracks the state of initial block download and fork resolution.
type syncState struct {
	syncing    atomic.Bool
	targetAddr string // address of the peer we're syncing from

	// Fork detection state
	reorging     bool           // true when we're collecting blocks for reorg
	forkBlocks   []*block.Block // buffered blocks from peer during fork detection
	peerHeight   uint64         // the peer's chain height
}

// IsSyncing returns true if the server is currently performing initial block download.
func (s *Server) IsSyncing() bool {
	return s.syncStatus.syncing.Load()
}

// startSync initiates the initial block download from a peer with a longer chain.
// Called after handshake when peer.height > local height.
func (s *Server) startSync(peer *Peer) {
	localHeight := s.chain.Height()
	peerHeight := peer.Height()

	if peerHeight <= localHeight {
		return
	}

	slog.Info("starting initial block download",
		"peer", peer.Addr(),
		"local_height", localHeight,
		"peer_height", peerHeight,
	)

	s.syncStatus.syncing.Store(true)
	s.syncStatus.targetAddr = peer.Addr()
	s.syncStatus.peerHeight = peerHeight
	s.syncStatus.reorging = false
	s.syncStatus.forkBlocks = nil

	// Request blocks from local tip + 1 to peer's height
	s.requestSyncBlocks(peer, localHeight+1)
}

// requestSyncBlocks sends a CmdGetBlocks request to the peer for blocks starting at startHeight.
func (s *Server) requestSyncBlocks(peer *Peer, startHeight uint64) {
	getBlocks := GetBlocksPayload{
		StartHeight: startHeight,
		EndHeight:   0, // 0 means "up to your tip"
	}
	msg, err := NewMessage(CmdGetBlocks, getBlocks)
	if err != nil {
		slog.Error("failed to create getblocks message", "err", err)
		s.abortSync()
		return
	}
	peer.Send(msg)
}

// handleSyncBlock processes a block received during IBD.
// Returns true if the block was handled as part of sync.
func (s *Server) handleSyncBlock(peer *Peer, msg Message) bool {
	if !s.syncStatus.syncing.Load() {
		return false
	}

	// Only accept sync blocks from the sync source peer
	if peer.Addr() != s.syncStatus.targetAddr {
		return false
	}

	var blockPayload BlockPayload
	if err := json.Unmarshal(msg.Payload, &blockPayload); err != nil {
		slog.Warn("invalid block payload during sync", "addr", peer.Addr(), "err", err)
		s.abortSync()
		s.removePeer(peer.Addr())
		return true
	}

	blk, err := blockPayload.ToBlock()
	if err != nil {
		slog.Warn("failed to deserialize block during sync", "addr", peer.Addr(), "err", err)
		s.abortSync()
		s.removePeer(peer.Addr())
		return true
	}

	// Validate PoW
	if !s.pow.Validate(blk) {
		slog.Warn("invalid PoW during sync, aborting", "addr", peer.Addr(), "height", blk.Height())
		s.abortSync()
		s.removePeer(peer.Addr())
		return true
	}

	// If we're in reorg mode, buffer the block for fork resolution
	if s.syncStatus.reorging {
		return s.handleReorgBlock(peer, blk)
	}

	// Normal sync: verify prevHash matches our current chain tip
	latestBlock := s.chain.LatestBlock()
	if latestBlock == nil {
		slog.Warn("chain not initialized during sync")
		s.abortSync()
		return true
	}

	if blk.PrevBlockHash() != latestBlock.Hash() {
		// PrevHash mismatch: this means the peer has a different chain (fork).
		// Initiate fork detection by requesting the peer's full chain from the beginning.
		slog.Info("sync detected fork, initiating fork resolution",
			"our_tip", latestBlock.Hash().String()[:16],
			"block_prev", blk.PrevBlockHash().String()[:16],
			"block_height", blk.Height(),
		)

		s.syncStatus.reorging = true
		s.syncStatus.forkBlocks = nil

		// Request blocks from height 1 (after genesis) to find the fork point.
		// Genesis is the same (verified during handshake).
		s.requestSyncBlocks(peer, 1)
		return true
	}

	// Normal sync: apply block
	return s.applySyncBlock(peer, blk)
}

// handleReorgBlock processes a block received during fork resolution.
// Buffers blocks and performs reorg when all are collected.
func (s *Server) handleReorgBlock(peer *Peer, blk *block.Block) bool {
	ctx := context.Background()

	// Find fork point by comparing this block with our chain at the same height
	ourBlock, err := s.chainRepo.GetBlockByHeight(ctx, blk.Height())
	if err != nil {
		// We don't have a block at this height -- this is beyond our chain tip,
		// so buffer it as a new block
		s.syncStatus.forkBlocks = append(s.syncStatus.forkBlocks, blk)

		// Check if we've collected all blocks
		if blk.Height() >= s.syncStatus.peerHeight {
			return s.executeReorg(peer)
		}
		return true
	}

	if ourBlock.Hash() == blk.Hash() {
		// Same block at this height -- not yet at the fork point.
		// Continue receiving. Clear any buffered fork blocks since we haven't
		// reached the divergence yet.
		s.syncStatus.forkBlocks = nil
		return true
	}

	// Different block at same height: this is the start of the fork.
	// Buffer this and all subsequent blocks.
	s.syncStatus.forkBlocks = append(s.syncStatus.forkBlocks, blk)

	// Check if we've collected all blocks
	if blk.Height() >= s.syncStatus.peerHeight {
		return s.executeReorg(peer)
	}

	return true
}

// executeReorg performs the chain reorganization using the buffered fork blocks.
func (s *Server) executeReorg(peer *Peer) bool {
	forkBlocks := s.syncStatus.forkBlocks
	if len(forkBlocks) == 0 {
		slog.Warn("no fork blocks collected, aborting reorg")
		s.abortSync()
		return true
	}

	// Fork point is just before the first fork block
	forkHeight := forkBlocks[0].Height() - 1

	slog.Info("executing chain reorganization",
		"fork_height", forkHeight,
		"new_blocks", len(forkBlocks),
		"new_tip_height", forkBlocks[len(forkBlocks)-1].Height(),
	)

	ctx := context.Background()
	if err := s.chain.Reorganize(ctx, forkHeight, forkBlocks, s.mempool); err != nil {
		slog.Error("chain reorganization failed", "err", err)
		s.abortSync()
		return true
	}

	slog.Info("chain reorganization complete",
		"new_height", s.chain.Height(),
		"new_tip", s.chain.LatestBlock().Hash().String()[:16],
	)

	// Clear sync state
	s.syncStatus.syncing.Store(false)
	s.syncStatus.targetAddr = ""
	s.syncStatus.reorging = false
	s.syncStatus.forkBlocks = nil

	// Broadcast the new tip to other peers (exclude the source)
	s.BroadcastBlock(s.chain.LatestBlock(), peer.Addr())

	return true
}

// applySyncBlock applies a single block during normal (non-fork) sync.
func (s *Server) applySyncBlock(peer *Peer, blk *block.Block) bool {
	// Extract transactions
	txs := make([]*tx.Transaction, 0, len(blk.RawTransactions()))
	for _, rawTx := range blk.RawTransactions() {
		if t, ok := rawTx.(*tx.Transaction); ok {
			txs = append(txs, t)
		}
	}

	// Apply UTXO changes and save block
	ctx := context.Background()
	if s.utxoSet != nil && len(txs) > 0 {
		undoEntry, err := s.utxoSet.ApplyBlock(blk.Height(), txs)
		if err != nil {
			slog.Error("failed to apply block UTXOs during sync", "height", blk.Height(), "err", err)
			s.abortSync()
			s.removePeer(peer.Addr())
			return true
		}

		if err := s.chainRepo.SaveBlockWithUTXOs(ctx, blk, undoEntry); err != nil {
			slog.Error("failed to save block during sync", "height", blk.Height(), "err", err)
			s.abortSync()
			return true
		}
	} else {
		if err := s.chainRepo.SaveBlock(ctx, blk); err != nil {
			slog.Error("failed to save block during sync", "height", blk.Height(), "err", err)
			s.abortSync()
			return true
		}
	}

	// Update chain tip
	s.chain.SetLatestBlock(blk)

	slog.Info("synced block", "height", blk.Height(), "hash", blk.Hash().String()[:16])

	// Check if we've caught up to the peer's height
	if blk.Height() >= peer.Height() {
		slog.Info("initial block download complete", "height", blk.Height())
		s.syncStatus.syncing.Store(false)
		s.syncStatus.targetAddr = ""
		return true
	}

	// If this is the last block in the batch (maxGetBlocksBatch boundary),
	// request the next batch
	if (blk.Height()-s.chain.Height()+blk.Height())%maxGetBlocksBatch == 0 || blk.Height()%maxGetBlocksBatch == 0 {
		// Request next batch -- the GetBlocks handler caps at 500 per request
		s.requestSyncBlocks(peer, blk.Height()+1)
	}

	return true
}

// abortSync cancels the current IBD and resets sync state.
func (s *Server) abortSync() {
	slog.Warn("aborting initial block download")
	s.syncStatus.syncing.Store(false)
	s.syncStatus.targetAddr = ""
	s.syncStatus.reorging = false
	s.syncStatus.forkBlocks = nil
}

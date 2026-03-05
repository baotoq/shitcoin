package p2p

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// syncState tracks the state of initial block download.
type syncState struct {
	syncing    atomic.Bool
	targetAddr string // address of the peer we're syncing from
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

	// Verify prevHash matches our current chain tip
	latestBlock := s.chain.LatestBlock()
	if latestBlock == nil {
		slog.Warn("chain not initialized during sync")
		s.abortSync()
		return true
	}

	if blk.PrevBlockHash() != latestBlock.Hash() {
		slog.Warn("sync block prevHash mismatch",
			"expected", latestBlock.Hash().String()[:16],
			"got", blk.PrevBlockHash().String()[:16],
			"height", blk.Height(),
		)
		s.abortSync()
		s.removePeer(peer.Addr())
		return true
	}

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
}

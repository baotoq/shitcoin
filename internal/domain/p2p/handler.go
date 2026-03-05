package p2p

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// handleMessage dispatches incoming messages to the appropriate handler.
func (s *Server) handleMessage(peer *Peer, msg Message) {
	switch msg.Command {
	case CmdVersion:
		// Version messages after handshake are protocol violations
		slog.Warn("unexpected version message after handshake", "addr", peer.Addr())
		s.removePeer(peer.Addr())
	case CmdVerack:
		// Verack after handshake is protocol violation
		slog.Warn("unexpected verack message after handshake", "addr", peer.Addr())
	case CmdInv:
		s.handleInv(peer, msg)
	case CmdGetData:
		s.handleGetData(peer, msg)
	case CmdGetBlocks:
		s.handleGetBlocks(peer, msg)
	case CmdBlock:
		s.handleBlock(peer, msg)
	case CmdTx:
		s.handleTx(peer, msg)
	default:
		slog.Debug("unknown command", "addr", peer.Addr(), "cmd", msg.Command)
	}
}

// handleInv processes an inventory announcement.
// For each hash not yet seen, sends a CmdGetData request to the announcing peer.
func (s *Server) handleInv(peer *Peer, msg Message) {
	var inv InvPayload
	if err := json.Unmarshal(msg.Payload, &inv); err != nil {
		slog.Warn("invalid inv payload", "addr", peer.Addr(), "err", err)
		return
	}

	for _, hash := range inv.Hashes {
		if s.MarkSeen(inv.Type, hash) {
			// Already seen this hash, skip
			continue
		}

		// Request the full data from the announcing peer
		getData := InvPayload{
			Type:   inv.Type,
			Hashes: []string{hash},
		}
		getDataMsg, err := NewMessage(CmdGetData, getData)
		if err != nil {
			slog.Error("failed to create getdata message", "err", err)
			continue
		}
		peer.Send(getDataMsg)
	}
}

// handleGetData processes a data request.
// For "block" type, looks up the block by hash and sends CmdBlock.
// For "tx" type, looks up from mempool and sends CmdTx.
func (s *Server) handleGetData(peer *Peer, msg Message) {
	var inv InvPayload
	if err := json.Unmarshal(msg.Payload, &inv); err != nil {
		slog.Warn("invalid getdata payload", "addr", peer.Addr(), "err", err)
		return
	}

	for _, hashStr := range inv.Hashes {
		switch inv.Type {
		case "block":
			hash, err := block.HashFromHex(hashStr)
			if err != nil {
				slog.Warn("invalid block hash in getdata", "hash", hashStr, "err", err)
				continue
			}

			ctx := context.Background()
			blk, err := s.chainRepo.GetBlock(ctx, hash)
			if err != nil {
				slog.Debug("block not found for getdata", "hash", hashStr[:16])
				continue
			}

			payload := BlockPayloadFromDomain(blk)
			blockMsg, err := NewMessage(CmdBlock, payload)
			if err != nil {
				slog.Error("failed to create block message", "err", err)
				continue
			}
			peer.Send(blockMsg)

		case "tx":
			hash, err := block.HashFromHex(hashStr)
			if err != nil {
				slog.Warn("invalid tx hash in getdata", "hash", hashStr, "err", err)
				continue
			}

			transaction := s.mempool.GetByID(hash)
			if transaction == nil {
				slog.Debug("tx not found for getdata", "hash", hashStr[:16])
				continue
			}

			payload := TxPayloadFromDomain(transaction)
			txMsg, err := NewMessage(CmdTx, payload)
			if err != nil {
				slog.Error("failed to create tx message", "err", err)
				continue
			}
			peer.Send(txMsg)
		}
	}
}

// handleTx processes a received transaction.
// Deserializes, validates via mempool.Add(), and re-broadcasts inv on success.
func (s *Server) handleTx(peer *Peer, msg Message) {
	var txPayload TxPayload
	if err := json.Unmarshal(msg.Payload, &txPayload); err != nil {
		slog.Warn("invalid tx payload", "addr", peer.Addr(), "err", err)
		return
	}

	transaction, err := txPayload.ToTransaction()
	if err != nil {
		slog.Warn("failed to deserialize tx", "addr", peer.Addr(), "err", err)
		return
	}

	// Attempt to add to mempool (validates signature, UTXO existence, double-spend)
	if err := s.mempool.Add(transaction); err != nil {
		slog.Debug("rejected tx from peer", "addr", peer.Addr(), "txid", transaction.ID().String()[:16], "err", err)
		return
	}

	slog.Info("accepted tx from peer", "addr", peer.Addr(), "txid", transaction.ID().String()[:16])

	// Re-broadcast inv to other peers (exclude sender)
	s.BroadcastTx(transaction, peer.Addr())
}

// handleBlock processes a received block.
// During IBD, routes to sync handler. Otherwise, validates PoW, checks prev-hash,
// applies to chain, and re-broadcasts.
func (s *Server) handleBlock(peer *Peer, msg Message) {
	// During sync, route blocks from the sync source to the sync handler
	if s.handleSyncBlock(peer, msg) {
		return
	}

	var blockPayload BlockPayload
	if err := json.Unmarshal(msg.Payload, &blockPayload); err != nil {
		slog.Warn("invalid block payload", "addr", peer.Addr(), "err", err)
		return
	}

	blk, err := blockPayload.ToBlock()
	if err != nil {
		slog.Warn("failed to deserialize block", "addr", peer.Addr(), "err", err)
		return
	}

	// Validate PoW
	if !s.pow.Validate(blk) {
		slog.Warn("rejected block with invalid PoW", "addr", peer.Addr(), "hash", blk.Hash().String()[:16])
		return
	}

	// Check that prevHash matches our chain tip
	latestBlock := s.chain.LatestBlock()
	if latestBlock == nil {
		slog.Warn("chain not initialized, cannot accept block")
		return
	}

	if blk.PrevBlockHash() != latestBlock.Hash() {
		slog.Warn("block prevHash does not match chain tip",
			"addr", peer.Addr(),
			"block_prev", blk.PrevBlockHash().String()[:16],
			"our_tip", latestBlock.Hash().String()[:16],
		)
		return
	}

	// Extract transactions for UTXO application
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
			slog.Error("failed to apply block UTXOs", "err", err)
			return
		}

		if err := s.chainRepo.SaveBlockWithUTXOs(ctx, blk, undoEntry); err != nil {
			slog.Error("failed to save block with UTXOs", "err", err)
			return
		}
	} else {
		if err := s.chainRepo.SaveBlock(ctx, blk); err != nil {
			slog.Error("failed to save block", "err", err)
			return
		}
	}

	// Update chain tip
	s.chain.SetLatestBlock(blk)

	// Remove block's transactions from mempool
	txIDs := make([]block.Hash, len(txs))
	for i, t := range txs {
		txIDs[i] = t.ID()
	}
	s.mempool.Remove(txIDs)

	slog.Info("accepted block from peer",
		"addr", peer.Addr(),
		"height", blk.Height(),
		"hash", blk.Hash().String()[:16],
	)

	// Invoke callback if registered
	if s.onBlockReceived != nil {
		s.onBlockReceived(blk)
	}

	// Re-broadcast inv to other peers (exclude sender)
	s.BroadcastBlock(blk, peer.Addr())
}

// maxGetBlocksBatch is the maximum number of blocks returned per CmdGetBlocks request.
const maxGetBlocksBatch = 500

// handleGetBlocks processes a GetBlocks request by serving the requested block range.
// Caps response at maxGetBlocksBatch blocks to prevent memory exhaustion.
func (s *Server) handleGetBlocks(peer *Peer, msg Message) {
	var payload GetBlocksPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		slog.Warn("invalid getblocks payload", "addr", peer.Addr(), "err", err)
		return
	}

	ctx := context.Background()

	// If EndHeight is 0, resolve to current chain height
	endHeight := payload.EndHeight
	if endHeight == 0 {
		endHeight = s.chain.Height()
	}

	// Validate range
	if payload.StartHeight > endHeight {
		return
	}

	// Cap batch size
	if endHeight-payload.StartHeight+1 > maxGetBlocksBatch {
		endHeight = payload.StartHeight + maxGetBlocksBatch - 1
	}

	blocks, err := s.chainRepo.GetBlocksInRange(ctx, payload.StartHeight, endHeight)
	if err != nil {
		slog.Error("failed to get blocks in range", "start", payload.StartHeight, "end", endHeight, "err", err)
		return
	}

	// Send each block as a CmdBlock message
	for _, blk := range blocks {
		bp := BlockPayloadFromDomain(blk)
		blockMsg, err := NewMessage(CmdBlock, bp)
		if err != nil {
			slog.Error("failed to create block message for sync", "height", blk.Height(), "err", err)
			continue
		}
		peer.Send(blockMsg)
	}

	slog.Info("served blocks to peer", "addr", peer.Addr(), "start", payload.StartHeight, "end", endHeight, "count", len(blocks))
}

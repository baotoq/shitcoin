package p2p

import "log/slog"

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
	default:
		slog.Debug("unknown command", "addr", peer.Addr(), "cmd", msg.Command)
	}
}

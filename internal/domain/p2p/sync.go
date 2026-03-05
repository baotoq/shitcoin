package p2p

import "sync/atomic"

// syncState tracks the state of initial block download.
type syncState struct {
	syncing atomic.Bool
}

// IsSyncing returns true if the server is currently performing initial block download.
func (s *Server) IsSyncing() bool {
	return s.syncStatus.syncing.Load()
}

package utxo

// UndoEntry records all UTXO changes made by applying a single block.
// Used to reverse block application during chain reorganization.
// All fields are exported for JSON serialization.
type UndoEntry struct {
	BlockHeight uint64     `json:"block_height"`
	Spent       []SpentUTXO `json:"spent"`
	Created     []UTXORef   `json:"created"`
}

// SpentUTXO records a UTXO that was consumed (spent) by a block's transactions.
// Stores full UTXO data so it can be restored during undo.
type SpentUTXO struct {
	TxID    string `json:"txid"`
	Vout    uint32 `json:"vout"`
	Value   int64  `json:"value"`
	Address string `json:"address"`
}

// UTXORef is a reference to a UTXO by its transaction ID and output index.
// Used to track which UTXOs were created by a block (for removal during undo).
type UTXORef struct {
	TxID string `json:"txid"`
	Vout uint32 `json:"vout"`
}

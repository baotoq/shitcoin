package bbolt

import (
	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// UTXOModel is a JSON-serializable storage model for UTXOs.
type UTXOModel struct {
	TxID    string `json:"txid"`
	Vout    uint32 `json:"vout"`
	Value   int64  `json:"value"`
	Address string `json:"address"`
}

// UTXOModelFromDomain converts a domain UTXO to a storage model.
func UTXOModelFromDomain(u utxo.UTXO) UTXOModel {
	return UTXOModel{
		TxID:    u.TxID().String(),
		Vout:    u.Vout(),
		Value:   u.Value(),
		Address: u.Address(),
	}
}

// ToDomain converts a storage model back to a domain UTXO.
func (m UTXOModel) ToDomain() (utxo.UTXO, error) {
	txID, err := block.HashFromHex(m.TxID)
	if err != nil {
		return utxo.UTXO{}, err
	}
	return utxo.NewUTXO(txID, m.Vout, m.Value, m.Address), nil
}
